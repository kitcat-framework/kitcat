package kitcache

import (
	"context"
	"github.com/kitcat-framework/kitcat"
	"github.com/kitcat-framework/kitcat/kitdi"
	"github.com/kitcat-framework/kitcat/kitslog"
	"github.com/spf13/viper"
	"log/slog"
)

type Config struct {
	StoreName string `cfg:"store_name"`
}

func (c *Config) InitConfig(prefix string) kitcat.ConfigUnmarshal {
	prefix = prefix + ".kitcache"
	viper.SetDefault(prefix+".store_name", "in_memory")

	return kitcat.ConfigUnmarshalHandler(prefix, c, "unable to unmarshal kitcache config: %w")
}

func init() {
	kitcat.RegisterConfig(new(Config))
}

type KitCache struct {
	Config       *Config
	CurrentStore Store

	logger *slog.Logger
}

func Module(config *Config, app *kitcat.App) {
	mod := &KitCache{
		Config: config,
		logger: slog.With(kitslog.Module("kitcache")),
	}

	app.Provides(
		kitcat.ProvideConfigurableModule(mod),
		ProvideStore(NewInMemoryStore),
	)
}

func (m *KitCache) Configure(_ context.Context, app *kitcat.App) error {
	app.Invoke(m.setCurrentStore)

	return nil
}

func (m *KitCache) Priority() uint8 { return 0 }

func (m *KitCache) setCurrentStore(a *kitcat.App, s stores) error {
	implementation, err := kitcat.UseImplementation(kitcat.UseImplementationParams[Store]{
		ModuleName:                m.Name(),
		ImplementationTerminology: "store",
		ConfigImplementationName:  m.Config.StoreName,
		Implementations:           s.Stores,
	})
	if err != nil {
		return err
	}

	m.CurrentStore = implementation
	m.logger.Info("using cache store", slog.String("sender", m.CurrentStore.Name()))
	a.Provides(kitdi.Annotate(m.CurrentStore, kitdi.As(new(Store))))

	return nil
}

func (m *KitCache) Name() string {
	return "kitmail"
}
