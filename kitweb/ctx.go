package kitweb

import (
	"context"
	"log/slog"
	"net/http"
)

// Ctx is the context passed to the handlerType
//
// - P is the type of the parameters that will be parsed from the request (default: httbind package),
// you can use: form, json,xml,query,header,path,file with default,explode tags
//
// - P must be a struct (without pointer)
//
// - P can have tags to specify how the binding should be done (default: httbind package)
//
// - P can have tags to specify how the validation should be done (default: go-playground/validator package)
//
// Res should be avoided, it's mainly used for middlewares.
type Ctx[P any] struct {
	Req *http.Request
	res http.ResponseWriter

	params *P

	bindingError error
	logger       *slog.Logger
	validate     ParamsValidator
	binder       ParamsBinder
}

type internalCtx interface {
	init(request *http.Request, response http.ResponseWriter, binder ParamsBinder, validate ParamsValidator)
	Logger() *slog.Logger
	SetRequestContextValue(key, value any)

	GetResponse() http.ResponseWriter
}

func (r *Ctx[P]) init(request *http.Request, response http.ResponseWriter, binder ParamsBinder, validate ParamsValidator) {
	r.Req = request
	r.res = response
	r.binder = binder
	r.validate = validate
}

func (r *Ctx[P]) Params() *P {
	if r.params == nil {
		r.params, r.bindingError = r.parseParams()
	}
	return r.params
}

// Validate returns the error that occurred during the validation of the request parameters
func (r *Ctx[P]) Validate() error {
	return r.validate.Validate(r.Params())
}

// BindingErrors returns the error that occurred during the binding of the request parameters
// This might not be needed to be check but it's here if you need it
func (r *Ctx[P]) BindingErrors() error {
	return r.bindingError
}

func (r *Ctx[P]) SetRequestContextValue(key, value any) {
	r.Req = r.Req.WithContext(context.WithValue(r.Req.Context(), key, value))
}

func (r *Ctx[P]) GetRequestContextValue(key any) any {
	return r.Req.Context().Value(key)
}

func (r *Ctx[P]) GetResponse() http.ResponseWriter {
	return r.res
}

func (r *Ctx[P]) Logger() *slog.Logger {
	if r.logger == nil {
		r.logger = slog.With(
			slog.String("method", r.Req.Method),
			slog.String("path", r.Req.URL.Path))
	}
	return r.logger
}

func (r *Ctx[P]) parseParams() (*P, error) {
	params := new(P)
	err := r.binder.Bind(r.Req, params)
	return params, err
}
