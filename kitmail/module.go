package kitmail

import (
	"context"
	"github.com/expectedsh/kitcat"
)

type Config struct {
	Host     string `env:"SMTP_HOST"`
	Port     int    `env:"SMTP_PORT"`
	Identity string `env:"SMTP_IDENTITY"`
	Username string `env:"SMTP_USERNAME"`
	Password string `env:"SMTP_PASSWORD"`

	SenderName string `env:"KITMAIL_SENDER_NAME"`
}

type Module struct {
	Config *Config

	CurrentSender Sender
}

func New(config *Config) func() *Module {
	return func() *Module {
		return &Module{
			Config: config,
		}
	}
}

func (m *Module) OnStart(_ context.Context, app *kitcat.App) error {
	app.Provides(
		func() *Config { return m.Config },
		NewSmtpSender,
	)

	app.Invoke(m.useSender)

	return nil
}

func (m *Module) useSender(a *kitcat.App, s senders) error {
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
	a.Provide(func() Sender { return implementation })

	return nil
}

func (m *Module) OnStop(_ context.Context, app *kitcat.App) error {
	return nil
}

func (m *Module) Name() string {
	return "kitmail"
}
