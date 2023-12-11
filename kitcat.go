package kitcat

import (
	"context"
	"fmt"
	"github.com/expectedsh/dig"
	"github.com/expectedsh/kitcat/kitdi"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

type (
	Nameable interface {
		Name() string
	}

	// Mod is an interface that can be implemented to provide a module
	Mod interface {
		OnStart(ctx context.Context, app *App) error
		OnStop(ctx context.Context, app *App) error
		Nameable
	}

	// Configurable is an optional interface that can be implemented by module that have specific dependencies
	// that rely on other modules.
	//
	// The Configurable.Configure method is called before every module OnStart methods.
	//
	// This is to prevent module that require the exported interface to fail requiring dependency
	//
	// The higher the priority is, the sooner the module will be configured.
	Configurable interface {
		Configure(ctx context.Context, app *App) error
		Priority() uint8
		Nameable
	}

	modules struct {
		dig.In
		Modules []Mod `group:"mod"`
	}

	configurables struct {
		dig.In
		Configurables []Configurable `group:"adaptable"`
	}

	ConfigUnmarshal func() error
	Config          interface {
		InitConfig(prefix string) ConfigUnmarshal
	}
)

func ModuleAnnotation(mod Mod) *kitdi.Annotation {
	return kitdi.Annotate(mod, kitdi.Group("mod"), kitdi.As(new(Mod)))
}

// ProvideConfigurableModule is used to inject a Configurable
func ProvideConfigurableModule(mod Configurable) *kitdi.Annotation {
	return kitdi.Annotate(mod, kitdi.Group("adaptable"), kitdi.As(new(Configurable)))
}

func ConfigUnmarshalHandler(prefix string, a any, msgf string, i ...any) ConfigUnmarshal {
	return func() error {
		err := viper.Sub(prefix).Unmarshal(a, func(config *mapstructure.DecoderConfig) {
			config.TagName = "cfg"
		})
		if err != nil {
			return fmt.Errorf(msgf, append(i, err)...)
		}

		return nil
	}
}
