package kitcache

import (
	"context"
	"github.com/dgraph-io/ristretto"
	"github.com/expectedsh/kitcat"
	"github.com/expectedsh/kitcat/kitdi"
	"github.com/expectedsh/kitcat/kitslog"
	"log/slog"
)

type Config struct {
	StoreName      string `env:"KITCACHE_STORE_NAME" envDefault:"in_memory"`
	InMemoryConfig *ristretto.Config
}

type Module struct {
	Config       *Config
	CurrentStore Store

	logger *slog.Logger
}

func New(config *Config) func(app *kitcat.App) {
	return func(app *kitcat.App) {
		mod := &Module{
			Config: config,
			logger: slog.With(kitslog.Module("kitcache")),
		}

		app.Provides(
			mod,
			kitcat.ConfigurableAnnotation(mod),
			StoreAnnotation(NewInMemoryStore),
			config,
		)
	}
}

func (m *Module) Configure(_ context.Context, app *kitcat.App) error {
	app.Invoke(m.setCurrentStore)

	return nil
}

func (m *Module) Priority() uint8 { return 0 }

func (m *Module) setCurrentStore(a *kitcat.App, s stores) error {
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

func (m *Module) Name() string {
	return "kitmail"
}
