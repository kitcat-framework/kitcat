package kitevent

import (
	"context"
	"github.com/expectedsh/kitcat"
	"github.com/expectedsh/kitcat/kitdi"
	"github.com/expectedsh/kitcat/kitslog"
	"log/slog"
	"reflect"
)

type Config struct {
	StoreName string `env:"KITEVENT_STORE_NAME" envDefault:"in-memory"`
}

type Module struct {
	config *Config
	logger *slog.Logger

	CurrentStore Store
}

func New(config *Config) func(a *kitcat.App) {
	return func(a *kitcat.App) {
		mod := &Module{
			config: config,
			logger: slog.With(kitslog.Module("kitevent")),
		}
		a.Provides(
			mod,
			kitcat.ModuleAnnotation(mod),
			StoreAnnotation(NewInMemoryEventStore),
			config,
		)
	}
}

func (m *Module) Configure(_ context.Context, app *kitcat.App) error {
	app.Invoke(m.setCurrentStore)

	return nil
}

func (m *Module) Priority() uint8 { return 0 }

func (m *Module) OnStart(_ context.Context, app *kitcat.App) error {
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

func (m *Module) setCurrentStore(app *kitcat.App, st stores) error {
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

	app.Provides(kitdi.Annotate(store, kitdi.As(new(Producer))))

	return nil
}

func (m *Module) OnStop(_ context.Context, _ *kitcat.App) error {
	return m.CurrentStore.OnStop()
}

func (m *Module) Name() string {
	return "kitevent"
}
