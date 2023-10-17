package kitcat

import (
	"context"
	"fmt"
	"github.com/expectedsh/dig"
	"github.com/expectedsh/kitcat/kitdi"
	"github.com/expectedsh/kitcat/kitexit"
	"github.com/expectedsh/kitcat/kitreflect"
	"github.com/expectedsh/kitcat/kitslog"
	"log/slog"
	"os"
	"os/signal"
	"reflect"
	"slices"
	"strings"
	"syscall"
	"time"
)

type Config struct {
	Environment      *Environment  `env:"KITCAT_ENV" envDefault:"development"`
	LoggerOutput     string        `env:"KITCAT_LOGGER_OUTPUT" envDefault:"stdout"`
	HooksMaxLifetime time.Duration `env:"KITCAT_HOOK_MAX_LIFETIME" envDefault:"10s"`
}

type App struct {
	config    *Config
	container *dig.Container
}

var GetLoggerFunc = getDefaultLogger

func New(cfg *Config) *App {
	a := &App{
		config:    cfg,
		container: dig.New(),
	}

	a.init()

	return a
}

func (a *App) Run() {
	err := a.container.Invoke(func(m configurables) {
		slog.Info("configuring modules", slog.Int("count", len(m.Configurables)))
		cancelFuncs := make([]context.CancelFunc, 0, len(m.Configurables))

		// sort for high (number) priority first
		slices.SortFunc(m.Configurables, func(a, b Configurable) int {
			return int(b.Priority()) - int(a.Priority())
		})

		for _, adaptable := range m.Configurables {
			timeoutCtx, cancelFunc := context.WithTimeout(context.Background(), a.config.HooksMaxLifetime)
			cancelFuncs = append(cancelFuncs, cancelFunc)
			slog.Info("configuring module", kitslog.Module(adaptable.Name()))
			if err := adaptable.Configure(timeoutCtx, a); err != nil {
				kitexit.Abnormal(fmt.Errorf("kitcat: error while configuring module %s: %w", adaptable.Name(), err))
			}
		}

		for _, cancelFunc := range cancelFuncs {
			cancelFunc()
		}
	})
	if err != nil {
		kitexit.Abnormal(fmt.Errorf("kitcat: error while configuring modules: %w", err))
		return
	}

	err = a.container.Invoke(func(m modules) {
		slog.Info("starting modules", slog.Int("count", len(m.Modules)))
		cancelFuncs := make([]context.CancelFunc, 0, len(m.Modules))
		for _, mod := range m.Modules {
			timeoutCtx, cancelFunc := context.WithTimeout(context.Background(), a.config.HooksMaxLifetime)
			cancelFuncs = append(cancelFuncs, cancelFunc)
			slog.Info("start module", kitslog.Module(mod.Name()))
			if err := mod.OnStart(timeoutCtx, a); err != nil {
				kitexit.Abnormal(fmt.Errorf("kitcat: error while starting module %s: %w", mod.Name(), err))
			}
		}

		for _, cancelFunc := range cancelFuncs {
			cancelFunc()
		}
	})

	if err != nil {
		kitexit.Abnormal(fmt.Errorf("kitcat: error while starting modules: %w", err))
		return
	}

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	<-stopChan

	a.Invoke(func(m modules) error {
		slog.Info("stopping modules", slog.Int("count", len(m.Modules)))
		cancelFuncs := make([]context.CancelFunc, 0, len(m.Modules))
		for _, mod := range m.Modules {
			timeoutCtx, cancelFunc := context.WithTimeout(context.Background(), a.config.HooksMaxLifetime)
			cancelFuncs = append(cancelFuncs, cancelFunc)

			slog.Info("stop module", kitslog.Module(mod.Name()))
			if err := mod.OnStop(timeoutCtx, a); err != nil {
				return fmt.Errorf("kitcat: error while stopping module %s: %w", mod.Name(), err)
			}
		}

		for _, cancelFunc := range cancelFuncs {
			cancelFunc()
		}

		return nil
	})

	slog.Info("kitcat: graceful shutdown")
	os.Exit(0)
}

func (a *App) Provides(constructors ...any) {
	for _, constructor := range constructors {
		var (
			ctype   = reflect.ValueOf(constructor)
			applier kitdi.Applier
			err     error
		)

		if ann, ok := constructor.(*kitdi.Annotation); ok {
			applier = ann
		} else if ctype.Kind() != reflect.Func {
			applier = kitdi.Supply(constructor)
		} else if isProvidableInvoker(constructor) {
			applier = kitdi.ProvidableInvoke(constructor)
		}

		if applier != nil {
			err = applier.Apply(a.container)
		} else {
			err = a.container.Provide(constructor)
		}

		if err != nil {
			kitexit.Abnormal(err)
		}
	}
}

func (a *App) Invoke(function any, opts ...dig.InvokeOption) {
	err := a.container.Invoke(function, opts...)
	if err != nil {
		kitexit.Abnormal(err)
	}
}

func (a *App) init() {
	a.Provides(a, a.config)

	var (
		out    = os.Stdout
		logger *slog.Logger
	)

	switch strings.ToLower(a.config.LoggerOutput) {
	case "stderr":
		out = os.Stderr
	default:
		out = os.Stdout
	}

	logger = GetLoggerFunc(a.config.Environment, out)

	slog.SetDefault(logger)
	a.Provides(logger)
}

func getDefaultLogger(environment *Environment, out *os.File) *slog.Logger {
	var logger *slog.Logger

	if environment.String() == Production.String() {
		logger = slog.New(slog.NewJSONHandler(out, nil))
	} else {
		logger = slog.New(slog.NewTextHandler(out, nil))
	}

	slog.SetDefault(logger)

	return logger
}

func isProvidableInvoker(any any) bool {
	reflection := reflect.TypeOf(any)
	if reflection.Kind() != reflect.Func {
		return false
	}

	if !kitreflect.EnsureInOutLength(reflection, 1, 0) {
		return false
	}

	if reflection.In(0).Kind() != reflect.Ptr {
		return false
	}

	if reflection.In(0).AssignableTo(reflect.TypeOf((*App)(nil)).Elem()) {
		return true
	}

	return true
}
