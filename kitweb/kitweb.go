package kitweb

import (
	"github.com/expectedsh/dig"
	"github.com/expectedsh/kitcat"
	"github.com/expectedsh/kitcat/kitdi"
	"net/http"
)

type (
	ParamsValidator interface {
		Validate(a any) error
	}

	ParamsBinder interface {
		Bind(request *http.Request, params any) error
		GetParsableTags() []string
	}

	Handler interface {
		Routes(r *Router)
		kitcat.Nameable
	}

	handlers struct {
		dig.In
		Handlers []Handler `group:"kitweb.handler"`
	}
)

func HandlerAnnotation(handler any) *kitdi.Annotation {
	return kitdi.Annotate(handler, kitdi.Group("kitweb.handler"), kitdi.As(new(Handler)))
}
