package kitweb

import (
	"github.com/expectedsh/kitcat"
	"github.com/expectedsh/kitcat/kitslog"
	"github.com/gorilla/mux"
	"log/slog"
	"net/http"
	"os"
	"reflect"
)

type Router struct {
	logger *slog.Logger

	handler   *mux.Router
	webModule *Module
	env       *kitcat.Environment
}

func newRouter(name string, m *Module, env *kitcat.Environment, routerModifier func(r *Router)) *Router {
	r := &Router{
		handler:   mux.NewRouter(),
		logger:    slog.With(kitslog.Module("globalRouter")),
		webModule: m,
		env:       env,
	}

	if routerModifier != nil {
		routerModifier(r)
	}

	return r
}

func (r *Router) RawRouter() *mux.Router {
	return r.handler
}

func (r *Router) Use(middlewares ...any) {
	for _, interceptor := range middlewares {
		middleware, err := r.toMiddlewareHandler(interceptor)
		if err != nil {
			r.logger.Error("invalid middleware",
				kitslog.Err(err),
				slog.String("middleware", reflect.TypeOf(interceptor).String()),
			)
			continue
		}

		r.handler.Use(middleware)
	}
}

func (r *Router) Get(path string, handler any, middlewares ...any) {
	r.Route(http.MethodGet, path, handler, middlewares...)
}

func (r *Router) Post(path string, handler any, middlewares ...any) {
	r.Route(http.MethodPost, path, handler, middlewares...)
}

func (r *Router) Put(path string, handler any, middlewares ...any) {
	r.Route(http.MethodPut, path, handler, middlewares...)
}

func (r *Router) Patch(path string, handler any, middlewares ...any) {
	r.Route(http.MethodPatch, path, handler, middlewares...)
}

func (r *Router) Delete(path string, handler any, middlewares ...any) {
	r.Route(http.MethodDelete, path, handler, middlewares...)
}

func (r *Router) Head(path string, handler any, middlewares ...any) {
	r.Route(http.MethodHead, path, handler, middlewares...)
}

func (r *Router) Options(path string, handler any, middlewares ...any) {
	r.Route(http.MethodOptions, path, handler, middlewares...)
}

func (r *Router) Trace(path string, handler any, middlewares ...any) {
	r.Route(http.MethodTrace, path, handler, middlewares...)
}

func adaptMiddlewares(handler http.Handler, adapters ...mux.MiddlewareFunc) http.Handler {
	// The loop is reversed so the adapters/middleware gets executed in the same
	// order as provided in the array.
	for i := len(adapters); i > 0; i-- {
		handler = adapters[i-1](handler)
	}
	return handler
}

func (r *Router) Route(method, path string, handler any, middlewares ...any) {
	httpHandler, err := r.toHTTPHandler(handler)
	if err != nil {
		r.logger.Error("invalid handlerType",
			kitslog.Err(err),
			slog.String("method", method),
			slog.String("path", path),
			slog.String("handlerType", reflect.TypeOf(handler).String()),
		)
		return
	}

	route := r.handler.
		Methods(method).
		Path(path)

	mws := make([]mux.MiddlewareFunc, 0)
	for _, interceptor := range middlewares {
		middleware, err := r.toMiddlewareHandler(interceptor)
		if err != nil {
			r.logger.Error("invalid middleware",
				kitslog.Err(err),
				slog.String("interceptor", reflect.TypeOf(interceptor).String()),
			)
			continue
		}

		middlewares = append(middlewares, middleware)
	}

	route.Handler(adaptMiddlewares(httpHandler, mws...))
}

func (r *Router) initPublicFolder() {
	folder := r.webModule.config.PublicFolder
	path := r.webModule.config.PublicPath

	_, err := os.Stat(folder)
	if os.IsNotExist(err) {
		// try to create the folder
		_ = os.Mkdir(folder, 0755)
	}

	if folder != "" {
		publicHandler := http.StripPrefix(path, http.FileServer(http.Dir(folder)))
		r.handler.PathPrefix(path).Handler(publicHandler)
	}
}
