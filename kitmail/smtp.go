package kitmail

import (
	"fmt"
	"github.com/expectedsh/kitcat"
	"github.com/expectedsh/kitcat/kitslog"
	"github.com/spf13/viper"
	"log/slog"
	"net/smtp"
)

type SmtpConfig struct {
	Host     string `cfg:"host"`
	Port     int    `cfg:"port"`
	Username string `cfg:"username"`
	Password string `cfg:"password"`
}

func (c *SmtpConfig) InitConfig(prefix string) kitcat.ConfigUnmarshal {
	prefix = prefix + ".kitmail.config_senders.smtp"
	viper.SetDefault(prefix+".host", "localhost")
	viper.SetDefault(prefix+".port", 25)
	viper.SetDefault(prefix+".username", "")
	viper.SetDefault(prefix+".password", "")

	return kitcat.ConfigUnmarshalHandler(prefix, c, "unable to unmarshal smtp config: %w")
}

func init() {
	kitcat.RegisterConfig(new(SmtpConfig))
}

type SmtpSender struct {
	Config *SmtpConfig
	logger *slog.Logger
}

func NewSmtpSender(config *SmtpConfig, logger *slog.Logger) *SmtpSender {
	return &SmtpSender{
		Config: config,
		logger: logger.With(
			kitslog.Module("kitmail"),
			slog.String("sender", "in_memory")),
	}
}

func (s *SmtpSender) Send(e Email) error {
	err := e.Send(fmt.Sprintf("%s:%d", s.Config.Host, s.Config.Port),
		smtp.CRAMMD5Auth(s.Config.Username, s.Config.Password))
	if err != nil {
		return fmt.Errorf("kitmail: failed to send email: %w", err)
	}

	return nil
}

func (s *SmtpSender) Name() string {
	return "smtp"
}
