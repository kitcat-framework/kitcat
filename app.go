package kitcat

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/kitcat-framework/kitcat/kitdi"
	"github.com/kitcat-framework/kitcat/kitexit"
	"github.com/kitcat-framework/kitcat/kitreflect"
	"github.com/kitcat-framework/kitcat/kitslog"
	"github.com/mitchellh/mapstructure"
	"github.com/samber/lo"
	godiffpatch "github.com/sourcegraph/go-diff-patch"
	"github.com/spf13/viper"
	"go.uber.org/dig"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type AppConfig struct {
	environment      *Environment
	LoggerOutput     string        `cfg:"_logger_output"`
	HooksMaxLifetime time.Duration `cfg:"_hooks_max_lifetime"`
	Host             string        `cfg:"host"`
}

func (c *AppConfig) InitConfig(prefix string) ConfigUnmarshal {
	viper.SetDefault("_hooks_max_lifetime", "10s")
	viper.SetDefault("_logger_output", "stdout")
	viper.SetDefault("_override_config_file", false)

	prefix = prefix + ".kitcat"
	viper.SetDefault(prefix+".host", "localhost:8080")

	return func() error {
		err := viper.Unmarshal(c, func(config *mapstructure.DecoderConfig) {
			config.TagName = "cfg"
		})
		if err != nil {
			return fmt.Errorf("unable to unmarshal app config: %w", err)
		}

		return ConfigUnmarshalHandler(prefix, c, "unable to unmarshal app config: %w")()
	}
}

func init() {
	RegisterConfig(new(AppConfig))
}

type App struct {
	config    *AppConfig
	container *dig.Container
}

var configs = make([]Config, 0)

// configsAny is used to rovide them to dig, without that all configs will be the same type (Config) and
// could not be used in constructors
var configsAny = make([]any, 0)

var GetLoggerFunc = getDefaultLogger

func New() *App {
	val, _ := lo.Find(configs, func(c Config) bool {
		_, ok := c.(*AppConfig)
		return ok
	})

	a := &App{
		config:    val.(*AppConfig),
		container: dig.New(),
	}

	_ = a.container.Provide(func() kitdi.Invokable { return kitdi.Invokable{} })

	a.loadConfigs()
	a.Provides(a.config.environment)
	a.init()

	return a
}

func (a *App) Run() {
	mesureStart := time.Now()

	a.configureModules()
	a.startModules()

	slog.Info("kitcat: started", slog.Duration("elapsed_time", time.Since(mesureStart)))

	if a.config.environment.Equal(EnvironmentDevelopment) {
		f, err := os.Create("dig.dot")
		if err == nil {
			_ = dig.Visualize(a.container, f)
		}
	}

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	<-stopChan

	a.stopModules()

	slog.Info("kitcat: graceful shutdown")
	os.Exit(0)
}

func (a *App) stopModules() {
	a.Invoke(func(m modules) error {
		slog.Info("stopping modules", slog.Int("count", len(m.Modules)))
		cancelFuncs := make([]context.CancelFunc, 0, len(m.Modules))
		for _, mod := range m.Modules {
			timeoutCtx, cancelFunc := context.WithTimeout(context.Background(), a.config.HooksMaxLifetime)
			cancelFuncs = append(cancelFuncs, cancelFunc)

			slog.Debug("stop module", kitslog.Module(mod.Name()))
			if err := mod.OnStop(timeoutCtx, a); err != nil {
				return fmt.Errorf("kitcat: error while stopping module %s: %w", mod.Name(), err)
			}
		}

		for _, cancelFunc := range cancelFuncs {
			cancelFunc()
		}

		return nil
	})
}

// LoadConfigs loads the config file and environment variables
// Environment variable is always prioritized over config file
// In a config you can set $<SOMETHING>  of ${SOMETHING} to get the value of an environment variable
//
// The config can be override by setting the environment variable OVERRIDE_CONFIG_FILE to true, it
// is used when adding module to an app to get the config for the module without having to write it.
//
// But the default behavior is to create a patch file that can be applied to the config file, in order
// to let the user add comments and stuffs for his own needs.
func (a *App) loadConfigs() {
	viper.SetConfigFile("config.yaml")
	viper.SetDefault("_environment", "development")

	unmarshalers := map[string][]ConfigUnmarshal{}

	for _, env := range AllEnvironments {
		for _, config := range configs {

			if _, ok := unmarshalers[env.Name]; !ok {
				unmarshalers[env.Name] = make([]ConfigUnmarshal, 0)
			}

			unmarshalers[env.Name] = append(unmarshalers[env.Name], config.InitConfig(env.Name))

		}

	}

	viper.AutomaticEnv()

	errReadConfig := viper.ReadInConfig()

	envStr := viper.GetString("_environment")
	if envStr == "" {
		kitexit.Abnormal(errors.New("kitcat: environment is not set"))
	} else if strings.HasPrefix(envStr, "$") {
		envStr = os.ExpandEnv(envStr)
	}

	var env Environment
	if err := env.UnmarshalText([]byte(envStr)); err != nil {
		kitexit.Abnormal(fmt.Errorf("kitcat: invalid environment: %w", err))
	}

	configOverrideStr := viper.GetString("_override_config_file")
	if configOverrideStr == "" {
		configOverrideStr = "false"
	} else if strings.HasPrefix(configOverrideStr, "$") {
		configOverrideStr = os.ExpandEnv(configOverrideStr)
	}

	configOverride, err := strconv.ParseBool(configOverrideStr)
	if err != nil {
		kitexit.Abnormal(fmt.Errorf("kitcat: invalid override config file value %s: %w", configOverrideStr, err))
	}

	if errReadConfig != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if errors.As(errReadConfig, &configFileNotFoundError) ||
			strings.Contains(errReadConfig.Error(), "no such file or directory") {
			if err := viper.WriteConfigAs("config.yaml"); err != nil {
				kitexit.Abnormal(fmt.Errorf("kitcat: error writing config file: %w", err))
			}

		} else {
			kitexit.Abnormal(fmt.Errorf("kitcat: error reading config file: %w", errReadConfig))
		}
	} else {
		if env.Equal(EnvironmentProduction) {
			// do nothing
		} else if configOverride {
			if err := viper.WriteConfigAs("config.yaml"); err != nil {
				kitexit.Abnormal(fmt.Errorf("kitcat: error writing config file: %w", err))
			}
		} else {
			// create a patch file

			temp, err := os.MkdirTemp("", "kitcat")
			if err != nil {
				kitexit.Abnormal(fmt.Errorf("kitcat: error creating temporary directory: %w", err))
			}

			tempFileName := filepath.Join(temp, fmt.Sprintf("config%s.yaml", uuid.New().String()))
			if err := viper.WriteConfigAs(tempFileName); err != nil {
				kitexit.Abnormal(fmt.Errorf("kitcat: error writing new config in temp file: %w", err))
			}

			newFileContent, err := os.ReadFile(tempFileName)
			if err != nil {
				kitexit.Abnormal(fmt.Errorf("kitcat: error reading temp config file: %w", err))
			}

			oldFileContent, err := os.ReadFile("config.yaml")
			if err != nil {
				kitexit.Abnormal(fmt.Errorf("kitcat: error reading config file: %w", err))
			}

			patch := godiffpatch.GeneratePatch("config.yml", string(oldFileContent), string(newFileContent))

			if err := os.WriteFile("config.yml.patch", []byte(patch), 0644); err != nil {
				kitexit.Abnormal(fmt.Errorf("kitcat: error writing patch file: %w", err))
			}
		}
	}

	for _, k := range viper.AllKeys() {
		val := viper.GetString(k)
		if strings.HasPrefix(val, "$") {
			viper.Set(k, os.ExpandEnv(val))
		} else {
			// if we do not do that the non-expanded values will be discarded from sub configs.
			// maybe a bug ?
			viper.Set(k, val)
		}
	}

	for _, unmarshaler := range unmarshalers[env.Name] {
		if err := unmarshaler(); err != nil {
			kitexit.Abnormal(fmt.Errorf("kitcat: error while unmarshaling config for env %s: %w", env.Name, err))
		}
	}

	a.config.environment = &env

	for _, config := range configsAny {
		a.Provides(config)
	}
}

func (a *App) startModules() {
	a.Invoke(func(m modules) {
		slog.Info("starting modules", slog.Int("count", len(m.Modules)))
		cancelFuncs := make([]context.CancelFunc, 0, len(m.Modules))
		for _, mod := range m.Modules {
			timeoutCtx, cancelFunc := context.WithTimeout(context.Background(), a.config.HooksMaxLifetime)
			cancelFuncs = append(cancelFuncs, cancelFunc)
			slog.Debug("start module", kitslog.Module(mod.Name()))
			if err := mod.OnStart(timeoutCtx, a); err != nil {
				kitexit.Abnormal(fmt.Errorf("kitcat: error while starting module %s: %w", mod.Name(), err))
			}
		}

		for _, cancelFunc := range cancelFuncs {
			cancelFunc()
		}
	})
}

func (a *App) configureModules() {
	a.Invoke(func(m configurables) {
		slog.Info("configuring modules", slog.Int("count", len(m.Configurables)))
		cancelFuncs := make([]context.CancelFunc, 0, len(m.Configurables))

		// sort for high (number) priority first
		slices.SortFunc(m.Configurables, func(a, b Configurable) int {
			return int(b.Priority()) - int(a.Priority())
		})

		for _, adaptable := range m.Configurables {
			timeoutCtx, cancelFunc := context.WithTimeout(context.Background(), a.config.HooksMaxLifetime)
			cancelFuncs = append(cancelFuncs, cancelFunc)
			slog.Debug("configuring module", kitslog.Module(adaptable.Name()))
			if err := adaptable.Configure(timeoutCtx, a); err != nil {
				kitexit.Abnormal(fmt.Errorf("kitcat: error while configuring module %s: %w", adaptable.Name(), err))
			}
		}

		for _, cancelFunc := range cancelFuncs {
			cancelFunc()
		}
	})
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

func (a *App) Modules(invokers ...any) {
	for _, f := range invokers {
		a.Invoke(f)
	}
}

func (a *App) init() {
	a.Provides(a)

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

	fmt.Println(a.config.environment)
	logger = GetLoggerFunc(a.config.environment, out)

	slog.SetDefault(logger)
	a.Provides(logger)
}

func RegisterConfig[T Config](config T) {
	configs = append(configs, config)
	configsAny = append(configsAny, config)
}

func getDefaultLogger(environment *Environment, out *os.File) *slog.Logger {
	var logger *slog.Logger

	if environment.String() == EnvironmentProduction.String() {
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

	if !kitreflect.EnsureMinParams(reflection, 1) {
		return false
	}

	if reflection.In(0).Name() != "Invokable" {
		return false
	}

	return true
}
