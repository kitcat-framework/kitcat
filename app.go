package kitcat

import (
	"context"
	"fmt"
	"github.com/expectedsh/dig"
	"github.com/expectedsh/kitcat/kitexit"
	"log/slog"
	"os"
	"os/signal"
	"reflect"
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
	err := a.container.Invoke(func(m modules) {
		cancelFuncs := make([]context.CancelFunc, 0, len(m.Modules))
		for _, mod := range m.Modules {
			timeoutCtx, cancelFunc := context.WithTimeout(context.Background(), a.config.HooksMaxLifetime)
			cancelFuncs = append(cancelFuncs, cancelFunc)
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
		cancelFuncs := make([]context.CancelFunc, 0, len(m.Modules))
		for _, mod := range m.Modules {
			timeoutCtx, cancelFunc := context.WithTimeout(context.Background(), a.config.HooksMaxLifetime)
			cancelFuncs = append(cancelFuncs, cancelFunc)

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

func (a *App) Provide(constructor any, opts ...dig.ProvideOption) {
	if isDisguisedInvoker(constructor) {
		constructor = constructor.(func(app *App) func(app *App))
		a.Invoke(constructor)
	}

	err := a.container.Provide(constructor, opts...)
	if err != nil {
		kitexit.Abnormal(err)
	}
}

func (a *App) Provides(constructors ...any) {
	for _, constructor := range constructors {
		a.Provide(constructor)
	}
}

func (a *App) Invoke(function any, opts ...dig.InvokeOption) {
	err := a.container.Invoke(function, opts...)
	if err != nil {
		kitexit.Abnormal(err)
	}
}

func (a *App) init() {
	// provide app globally

	a.Provides(func() *App { return a })

	// provide config globally

	a.Provides(func() *Config { return a.config })

	// init logger

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
	a.Provides(func() *slog.Logger { return logger })
}

func getDefaultLogger(environment *Environment, out *os.File) *slog.Logger {
	var logger *slog.Logger

	if environment.String() == Production.String() {
		logger = slog.New(slog.NewTextHandler(out, nil))
	} else {
		logger = slog.New(slog.NewJSONHandler(out, nil))
	}

	slog.SetDefault(logger)

	return logger
}

func isDisguisedInvoker(any any) bool {
	reflection := reflect.TypeOf(any)
	if reflection.Kind() != reflect.Func {
		return false
	}

	if reflection.NumIn() != 1 {
		return false
	}

	if reflection.NumOut() > 0 {
		return false
	}

	if reflection.In(0).Kind() != reflect.Ptr {
		return false
	}

	if reflection.In(0).Elem().Name() != "App" &&
		reflection.In(0).Elem().PkgPath() != "github.com/expectedsh/kitcat" {
		return false
	}

	return true
}
