package kitmail

import (
	"context"
	"github.com/expectedsh/kitcat"
	"github.com/expectedsh/kitcat/kitdi"
	"github.com/expectedsh/kitcat/kitslog"
	"github.com/spf13/viper"
	"log/slog"
)

type Config struct {
	SenderName string `cfg:"sender_name"`
}

func (c *Config) InitConfig(prefix string) kitcat.ConfigUnmarshal {
	prefix = prefix + ".kitmail"
	viper.SetDefault(prefix+".sender_name", "smtp")

	return kitcat.ConfigUnmarshalHandler(prefix, c, "unable to unmarshal kitmail config: %w")
}

func init() {
	kitcat.RegisterConfig(new(Config))
}

type Module struct {
	Config *Config

	CurrentSender Sender
	logger        *slog.Logger
}

func New(_ kitdi.Invokable, app *kitcat.App, config *Config) {
	mod := &Module{
		Config: config,
		logger: slog.With(kitslog.Module("kitmail")),
	}

	app.Provides(
		kitcat.ProvideConfigurableModule(mod),
		ProvideSender(NewSmtpSender),
	)

}

func (m *Module) Configure(_ context.Context, app *kitcat.App) error {
	app.Invoke(m.setCurrentSender)

	return nil
}

func (m *Module) Priority() uint8 { return 0 }

func (m *Module) setCurrentSender(a *kitcat.App, s senders) error {
	implementation, err := kitcat.UseImplementation(kitcat.UseImplementationParams[Sender]{
		ModuleName:                m.Name(),
		ImplementationTerminology: "sender",
		ConfigImplementationName:  m.Config.SenderName,
		Implementations:           s.Senders,
	})
	if err != nil {
		return err
	}

	m.CurrentSender = implementation
	m.logger.Info("using sender", slog.String("sender", m.CurrentSender.Name()))
	a.Provides(kitdi.Annotate(m.CurrentSender, kitdi.As(new(Sender))))

	return nil
}

func (m *Module) Name() string {
	return "kitmail"
}
