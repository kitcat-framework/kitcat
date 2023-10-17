package kitmail

import (
	"fmt"
	"github.com/expectedsh/kitcat/kitslog"
	"log/slog"
	"net/smtp"
)

type SmtpSender struct {
	Config *Config
	logger *slog.Logger
}

func NewSmtpSender(config *Config, logger *slog.Logger) *SmtpSender {
	return &SmtpSender{
		Config: config,
		logger: logger.With(
			kitslog.Module("kitmail"),
			slog.String("sender", "in_memory")),
	}
}

func (s *SmtpSender) Send(e Email) error {
	err := e.Send(fmt.Sprintf("%s:%d", s.Config.Host, s.Config.Port), smtp.PlainAuth(
		s.Config.Identity,
		s.Config.Username,
		s.Config.Password,
		s.Config.Host))
	if err != nil {
		return fmt.Errorf("kitmail: failed to send email: %w", err)
	}

	return nil
}

func (s *SmtpSender) Name() string {
	return "smtp"
}
