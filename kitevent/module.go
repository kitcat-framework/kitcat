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

func (m *Module) registerHandlers(h consumers) error {
	if len(h.Consumers) == 0 {
		return nil
	}

	m.logger.Info("registering consumers", slog.Int("count", len(h.Consumers)))

	for _, consumer := range h.Consumers {
		if !IsHandler(consumer) {
			m.logger.Warn("invalid consumer, must implement method Consume(context.Context, <kitevent.Event>)",
				slog.String("consumer", reflect.TypeOf(consumer).String()))
			continue
		}

		eventName := reflect.New(reflect.ValueOf(consumer).
			MethodByName("Consume").
			Type().In(1).Elem()).
			Interface().(Event).
			EventName()

		m.logger.Info("registering consumer",
			slog.String("consumer", consumer.Name()),
			slog.String("event", eventName.Name))
		m.CurrentStore.AddConsumer(eventName, consumer)
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
