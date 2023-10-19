package kitmail

import (
	"context"
	"github.com/expectedsh/kitcat"
	"github.com/expectedsh/kitcat/kitdi"
	"github.com/expectedsh/kitcat/kitslog"
	"log/slog"
)

type Config struct {
	Host     string `env:"SMTP_HOST"`
	Port     int    `env:"SMTP_PORT"`
	Identity string `env:"SMTP_IDENTITY"`
	Username string `env:"SMTP_USERNAME"`
	Password string `env:"SMTP_PASSWORD"`

	SenderName string `env:"KITMAIL_SENDER_NAME" envDefault:"smtp"`
}

type Module struct {
	Config *Config

	CurrentSender Sender
	logger        *slog.Logger
}

func New(config *Config) func(app *kitcat.App) {
	return func(app *kitcat.App) {
		mod := &Module{
			Config: config,
			logger: slog.With(kitslog.Module("kitmail")),
		}

		app.Provides(
			mod,
			kitcat.ConfigurableAnnotation(mod),
			SenderAnnotation(NewSmtpSender),
			config,
		)
	}
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
