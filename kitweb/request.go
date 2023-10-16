package kitweb

import (
	"log/slog"
	"net/http"
)

type Req[P any] struct {
	*http.Request
	params *P

	bindingError error
	logger       *slog.Logger
	validate     ParamsValidator
	binder       ParamsBinder
}

func newRequest[P any](request *http.Request, binder ParamsBinder, validate ParamsValidator) *Req[P] {
	return &Req[P]{
		Request:  request,
		binder:   binder,
		validate: validate,
	}
}

func (r *Req[P]) Params() *P {
	if r.params == nil {
		r.params, r.bindingError = r.parseParams()
	}
	return r.params
}

func (r *Req[P]) Validate() error {
	return r.validate.Validate(r.Params())
}

func (r *Req[P]) BindingErrors() error {
	return r.bindingError
}

func (r *Req[P]) Logger() *slog.Logger {
	if r.logger == nil {
		r.logger = slog.With(
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path))
	}
	return r.logger
}

func (r *Req[P]) parseParams() (*P, error) {
	params := new(P)
	err := r.binder.Bind(r.Request, params)
	return params, err
}
