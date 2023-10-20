package kitweb

import (
	"context"
	"fmt"
	"github.com/expectedsh/kitcat/kitslog"
	"github.com/gorilla/mux"
	"log/slog"
	"net/http"
	"reflect"
	"sync"
)

type Router struct {
	logger *slog.Logger

	handler   *mux.Router
	sync      sync.Mutex
	webModule *Module
}

func newRouter(m *Module) *Router {
	return &Router{
		handler:   mux.NewRouter(),
		logger:    slog.With(kitslog.Module("router")),
		webModule: m,
	}
}

func (r *Router) Route(method, path string, handler any) {
	httpHandler, err := r.getHTTPHandler(handler)
	if err != nil {
		r.logger.Error("invalid handler",
			kitslog.Err(err),
			slog.String("method", method),
			slog.String("path", path),
			slog.String("handler", reflect.TypeOf(handler).String()),
		)
		return
	}

	r.handler.
		Methods(method).
		Path(path).
		Handler(httpHandler)
}

func (r *Router) Use(handler ...any) {
	//r.middlewares = append(r.middlewares, handler...)
}

func (r *Router) Get(path string, handler any) {
	r.sync.Lock()
	defer r.sync.Unlock()

	r.Route(http.MethodGet, path, handler)
}

func (r *Router) Post(path string, handler any) {
	r.sync.Lock()
	defer r.sync.Unlock()

	r.Route(http.MethodPost, path, handler)
}

func (r *Router) Put(path string, handler any) {
	r.sync.Lock()
	defer r.sync.Unlock()

	r.Route(http.MethodPut, path, handler)
}

func (r *Router) Patch(path string, handler any) {
	r.sync.Lock()
	defer r.sync.Unlock()

	r.Route(http.MethodPatch, path, handler)
}

func (r *Router) Delete(path string, handler any) {
	r.sync.Lock()
	defer r.sync.Unlock()

	r.Route(http.MethodDelete, path, handler)
}

type HTTPHandler interface {
	ServeHTTP(provider *Module) http.HandlerFunc
}

type HandlerFunc[P any] func(r *Req[P]) Res

func (h HandlerFunc[P]) ServeHTTP(module *Module) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		request := newRequest[P](r, module.paramsBinder, module.paramsValidator)
		response := h(request)

		ctx := r.Context()
		context.WithValue(ctx, ctxKeyEnginesValue, module.engines)

		if err := response.Write(ctx, w); err != nil {
			// todo: based on the http accept content type header, we should return a json response or a html response
			// with the error message
			request.Logger().Error("error while writing response", kitslog.Err(err))
			return
		}
	}
}

func (r *Router) getHTTPHandler(handler any) (http.Handler, error) {
	if value, ok := handler.(func(http.ResponseWriter, *http.Request)); ok {
		handler = http.HandlerFunc(value)
		return handler.(http.Handler), nil
	} else if value, ok := handler.(HTTPHandler); ok {
		return value.ServeHTTP(r.webModule), nil
	}

	return nil, fmt.Errorf("invalid handler type: %s", reflect.TypeOf(handler).String())
}
