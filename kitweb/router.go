package kitweb

import (
	"github.com/gorilla/mux"
	"github.com/kitcat-framework/kitcat"
	"github.com/kitcat-framework/kitcat/kitslog"
	"log/slog"
	"net/http"
	"os"
	"reflect"
	"slices"
)

type detailedMiddleware struct {
	order      int
	middleware mux.MiddlewareFunc
	name       string
}

type orderedMiddlewares []*detailedMiddleware

func (o orderedMiddlewares) Len() int {
	return len(o)
}

// SortByOrder sorts the middlewares by order, more the order is high, more the middleware will be executed firstly
func (o orderedMiddlewares) SortByOrder() []mux.MiddlewareFunc {
	slices.SortFunc(o, func(a, b *detailedMiddleware) int {
		if a.order > b.order {
			return 1
		} else if a.order < b.order {
			return -1
		}
		return 0
	})

	middlewares := make([]mux.MiddlewareFunc, len(o))
	for i, m := range o {
		middlewares[i] = m.middleware
	}

	return middlewares
}

type Router struct {
	logger *slog.Logger

	handler   *mux.Router
	webModule *KitWeb
	env       *kitcat.Environment

	middlewares orderedMiddlewares
}

func newRouter(name string, m *KitWeb, env *kitcat.Environment, routerModifier func(r *Router)) *Router {
	r := &Router{
		handler:   mux.NewRouter(),
		logger:    slog.With(kitslog.Module("router"), slog.String("name", name)),
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

// Use must be used before any route declaration
func (r *Router) Use(middlewares ...any) {
	for i, interceptor := range middlewares {
		order := i
		name := reflect.TypeOf(interceptor).String()
		if orderedMiddleware, ok := interceptor.(*DetailedMiddleware); ok {
			interceptor = orderedMiddleware.Middleware
			name = orderedMiddleware.Name

			if orderedMiddleware.Order != nil {
				order = *orderedMiddleware.Order
			}
		}

		middleware, err := r.toMiddlewareHandler(interceptor)
		if err != nil {
			r.logger.Error("invalid middleware",
				kitslog.Err(err),
				slog.String("middleware", reflect.TypeOf(interceptor).String()),
			)
			continue
		}

		r.middlewares = append(r.middlewares, &detailedMiddleware{
			order:      order,
			middleware: middleware,
			name:       name,
		})
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

	mws := make(orderedMiddlewares, len(r.middlewares))
	copy(mws, r.middlewares)

	for i, interceptor := range middlewares {
		order := i
		name := reflect.TypeOf(interceptor).String()
		if orderedMiddleware, ok := interceptor.(*DetailedMiddleware); ok {
			interceptor = orderedMiddleware.Middleware
			name = orderedMiddleware.Name

			if orderedMiddleware.Order != nil {
				order = *orderedMiddleware.Order
			}
		}

		middleware, err := r.toMiddlewareHandler(interceptor)
		if err != nil {
			r.logger.Error("invalid middleware",
				kitslog.Err(err),
				slog.String("interceptor", reflect.TypeOf(interceptor).String()),
			)
			continue
		}

		mws = append(mws, &detailedMiddleware{
			order:      order,
			middleware: middleware,
			name:       name,
		})
	}

	muxMiddlewares := mws.SortByOrder()
	// todo: add a way to debug middlewares
	route.Handler(adaptMiddlewares(httpHandler, muxMiddlewares...))
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
		r.handler.PathPrefix(path).Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			publicHandler.ServeHTTP(w, r)
		}))
	}
}
