package kitevent

import (
	"context"
	"github.com/expectedsh/kitcat"
	"github.com/expectedsh/kitcat/kitslog"
	"log/slog"
	"reflect"
)

type Config struct {
	StoreName string `env:"KITEVENT_STORE_NAME"`
}

type Module struct {
	config *Config
	logger *slog.Logger

	CurrentStore Store
}

func New(config *Config) func() kitcat.ProvidableModule {
	return func() kitcat.ProvidableModule {
		return kitcat.NewProvidableModule(&Module{
			config: config,
			logger: slog.With(kitslog.Module("kitevent")),
		})
	}
}

func (m *Module) OnStart(_ context.Context, app *kitcat.App) error {
	app.Provide(func() *Config { return m.config })
	app.Provides(NewInMemoryEventStore)

	app.Invoke(m.useStores)
	app.Invoke(m.useHandlers)

	return m.CurrentStore.OnStart()
}

func (m *Module) useHandlers(h handlers) error {
	for _, handler := range h.Handlers {
		if !IsHandler(handler) {
			m.logger.Warn("invalid handler, must implement method Handle(context.Context, kitevent.Event)",
				slog.String("handler", reflect.TypeOf(handler).String()))
			continue
		}

		eventName := reflect.New(reflect.ValueOf(handler).
			MethodByName("Handle").
			Type().In(1)).
			Interface().(Event).
			EventName()

		m.CurrentStore.AddEventHandler(eventName, handler)
	}

	return nil
}

func (m *Module) useStores(app *kitcat.App, st stores) error {
	store, err := kitcat.UseImplementation(kitcat.UseImplementationParams[Store]{
		ModuleName:                m.Name(),
		ImplementationTerminology: "store",
		ConfigImplementationName:  m.config.StoreName,
		Implementations:           st.Stores,
	})
	if err != nil {
		return err
	}

	m.CurrentStore = store
	app.Provide(func() Producer { return store })

	return nil
}

func (m *Module) OnStop(_ context.Context, _ *kitcat.App) error {
	return m.CurrentStore.OnStop()
}

func (m *Module) Name() string {
	return "kitevent"
}
