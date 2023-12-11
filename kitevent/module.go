package kitevent

import (
	"context"
	"github.com/expectedsh/kitcat"
	"github.com/expectedsh/kitcat/kitdi"
	"github.com/expectedsh/kitcat/kitslog"
	"github.com/spf13/viper"
	"log/slog"
	"reflect"
)

type Config struct {
	StoreName string `cfg:"store_name"`
}

func (c *Config) InitConfig(prefix string) kitcat.ConfigUnmarshal {
	prefix = prefix + ".kitevent"
	viper.SetDefault(prefix+".store_name", "in-memory")

	return kitcat.ConfigUnmarshalHandler(prefix, c, "unable to unmarshal kitevent config: %w")
}

func init() {
	kitcat.RegisterConfig(new(Config))
}

type Module struct {
	config *Config
	logger *slog.Logger

	CurrentStore Store
}

func New(_ kitdi.Invokable, a *kitcat.App, config *Config) {
	mod := &Module{
		config: config,
		logger: slog.With(kitslog.Module("kitevent")),
	}

	a.Provides(
		kitcat.ModuleAnnotation(mod),
		kitcat.ProvideConfigurableModule(mod),
		ProvideStore(NewInMemoryEventStore),
	)
}

func (m *Module) Configure(_ context.Context, app *kitcat.App) error {
	app.Invoke(m.setCurrentStore)

	return nil
}

func (m *Module) Priority() uint8 { return 0 }

func (m *Module) OnStart(ctx context.Context, app *kitcat.App) error {
	app.Invoke(m.registerHandlers)

	return m.CurrentStore.OnStart(ctx)
}

func (m *Module) registerHandlers(h handlers) error {
	if len(h.Handlers) == 0 {
		return nil
	}

	m.logger.Info("registering handlers", slog.Int("count", len(h.Handlers)))

	for _, handler := range h.Handlers {
		if !IsHandler(handler) {
			m.logger.Warn("invalid Handler, must implement method Handle(context.Context, kitevent.Event)",
				slog.String("handler", reflect.TypeOf(handler).String()))
			continue
		}

		eventName := reflect.New(reflect.ValueOf(handler).
			MethodByName("Handle").
			Type().In(1).Elem()).
			Interface().(Event).
			EventName()

		m.logger.Info("registering handler",
			slog.String("handler", handler.Name()),
			slog.String("event", eventName.Name))
		m.CurrentStore.AddEventHandler(eventName, handler)
	}

	return nil
}

func (m *Module) setCurrentStore(app *kitcat.App, st stores) error {
	m.logger.Debug("stores", slog.Int("count", len(st.Stores)), slog.String("want", m.config.StoreName))
	store, err := kitcat.UseImplementation(kitcat.UseImplementationParams[Store]{
		ModuleName:                m.Name(),
		ImplementationTerminology: "store",
		ConfigImplementationName:  m.config.StoreName,
		Implementations:           st.Stores,
	})
	if err != nil {
		return err
	}

	m.logger.Info("using store", slog.String("store", store.Name()))
	m.CurrentStore = store

	app.Provides(kitdi.Annotate(store, kitdi.As(new(Producer))))

	return nil
}

func (m *Module) OnStop(ctx context.Context, _ *kitcat.App) error {
	return m.CurrentStore.OnStop(ctx)
}

func (m *Module) Name() string {
	return "kitevent"
}
