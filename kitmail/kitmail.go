package kitmail

import (
	"github.com/expectedsh/dig"
	jemail "github.com/jordan-wright/email"
)

type (
	Email struct {
		jemail.Email
	}

	Sender interface {
		Send(e Email) error
		Name() string
	}

	senders struct {
		dig.In
		Senders []Sender `group:"kitmail.sender"`
	}

	ProvidableSender struct {
		dig.Out
		Sender Sender `group:"kitmail.sender"`
	}
)

func NewProvidableSender(sender Sender) ProvidableSender {
	return ProvidableSender{Sender: sender}
}
