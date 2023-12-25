package kitweb

import (
	"context"
	"errors"
	"fmt"
	"github.com/expectedsh/kitcat/kitslog"
	"github.com/gorilla/mux"
	"log/slog"
	"net/http"
	"os"
	"reflect"
	"runtime/debug"
	"sync"
)

type Router struct {
	logger *slog.Logger

	handler   *mux.Router
	sync      sync.Mutex
	webModule *Module
}

func newRouter(m *Module) *Router {
	r := &Router{
		handler:   mux.NewRouter(),
		logger:    slog.With(kitslog.Module("router")),
		webModule: m,
	}

	r.initPublicFolder()

	return r
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

func (r *Router) RawRouter() *mux.Router {
	return r.handler
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
		response, err := func() (r Res, err error) {
			defer func() {
				err := recover()
				if err != nil {
					fmt.Println("--------------------------------------------")
					fmt.Println("WARNING RECOVER FROM HTTP HANDLER")
					fmt.Println()
					fmt.Println(err)
					fmt.Println(string(debug.Stack()))
					fmt.Println()
					fmt.Println("--------------------------------------------")

					err = errors.New("panic")
				}
			}()

			return h(request), nil
		}()

		if err != nil {
			return
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, ctxKeyEnginesValue, module.engines)

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

// addTrailingSlash add a trailing slash to the url if it doesn't have one
func addTrailingSlash(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" && r.URL.Path[len(r.URL.Path)-1] != '/' {
			http.Redirect(w, r, r.URL.Path+"/", http.StatusMovedPermanently)
			return
		}
		next.ServeHTTP(w, r)
	})
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
