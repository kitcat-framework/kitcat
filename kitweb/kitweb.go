package kitweb

import (
	"context"
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

	Res interface {
		Write(ctx context.Context, w http.ResponseWriter) error
	}

	HandlerFunc[P any] func(r *Ctx[P]) Res

	Middleware[P any] func(r *Ctx[P], next http.HandlerFunc) Res

	// ExceptionHandlerFunc is a function that handle an exception, it can be used to show the
	// error while panicking from a handlerType or middlewaare, for 404 errors ...
	ExceptionHandlerFunc func(rw http.ResponseWriter, req *http.Request, err error)

	handlers struct {
		dig.In
		Handlers []Handler `group:"kitweb.handlerType"`
	}
)

func ProvideHandler(handler any) *kitdi.Annotation {
	return kitdi.Annotate(handler, kitdi.Group("kitweb.handlerType"), kitdi.As(new(Handler)))
}
