package kitmail

import (
	jemail "github.com/jordan-wright/email"
	"github.com/kitcat-framework/kitcat/kitdi"
	"go.uber.org/dig"
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
)

func ProvideSender(sender any) *kitdi.Annotation {
	return kitdi.Annotate(sender, kitdi.Group("kitmail.sender"), kitdi.As(new(Sender)))
}
