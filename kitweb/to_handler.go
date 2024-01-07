package kitweb

import (
	"context"
	"errors"
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"github.com/gorilla/mux"
	"github.com/kitcat-framework/kitcat"
	"github.com/kitcat-framework/kitcat/kitreflect"
	"log/slog"
	"net/http"
	"reflect"
	"runtime/debug"
	"strings"
)

var errWrite = fmt.Errorf("response write error")

type panicked struct {
	error       any
	StackTrace  string
	handlerType handlerType
}

func (e panicked) Error() string {
	return fmt.Sprintf("%s\n%s", e.error, e.StackTrace)
}

func (e panicked) print(env *kitcat.Environment, logger *slog.Logger) {
	if env.Equal(kitcat.EnvironmentProduction) {
		logger.Error("panic",
			slog.Any("panic_value", e.error),
			slog.String("handler_type", e.handlerType.String()),
			slog.String("stack_trace", e.StackTrace))
		return
	}

	fmt.Println()

	errorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#ffffff")).
		Background(lipgloss.Color("#dc2626")).
		Bold(true).
		Render(" KITWEB: PANIC OCCURRED ")

	handlerTypeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#451a03")).
		Background(lipgloss.Color("#fbbf24")).
		Bold(true).
		Render(fmt.Sprintf(" HANDLER TYPE: %s ", strings.ToUpper(e.handlerType.String())))

	grayLine := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6b7280")).
		Render("---------------------------------------------------")

	fmt.Printf("%s%s\n", errorStyle, handlerTypeStyle)
	fmt.Println(grayLine)
	fmt.Println(e.StackTrace)
	fmt.Println(grayLine)
	fmt.Println()
}

type wrappedResponseWriter struct {
	http.ResponseWriter
	alreadyWritten bool
	statusCode     int
}

func (w *wrappedResponseWriter) WriteHeader(statusCode int) {
	w.alreadyWritten = true
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *wrappedResponseWriter) Write(data []byte) (int, error) {
	w.alreadyWritten = true
	return w.ResponseWriter.Write(data)
}

type handlerType int

const (
	handlerKitwebHandler handlerType = iota
	handlerStdHttpHandler
	handlerStdMiddleware
	handlerKitwebMiddleware
)

func (h handlerType) isCustom() bool {
	return h == handlerKitwebHandler || h == handlerKitwebMiddleware
}

func (h handlerType) isMiddleware() bool {
	return h == handlerStdMiddleware || h == handlerKitwebMiddleware
}

func (h handlerType) isHandler() bool {
	return h == handlerStdHttpHandler || h == handlerKitwebHandler
}

func (h handlerType) String() string {
	switch h {
	case handlerKitwebHandler:
		return "custom_handler"
	case handlerStdHttpHandler:
		return "http_handler"
	case handlerStdMiddleware:
		return "mux_middleware"
	case handlerKitwebMiddleware:
		return "custom_middleware"
	default:
		return "unknown_handlerType"
	}
}

func (r *Router) toHTTPHandler(h any) (http.Handler, error) {
	handlerType := handlerStdHttpHandler
	var handler any

	if value, ok := h.(func(http.ResponseWriter, *http.Request)); ok {
		handler = http.HandlerFunc(value)
	} else if value, ok := h.(http.Handler); ok {
		handler = value
	} else {
		functionType := reflect.TypeOf(h)
		err := r.isHandlerFunc(functionType)
		if err != nil {
			return nil, fmt.Errorf(""+
				"function handlerType should be a http.HandlerFunc, http.Handler or <func(r *Ctx[T]) Res> : %w",
				err,
			)
		}

		handlerType = handlerKitwebHandler
		handler = functionType
	}

	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		var res *wrappedResponseWriter
		if value, ok := rw.(*wrappedResponseWriter); ok {
			res = value
		} else {
			res = &wrappedResponseWriter{ResponseWriter: rw}
		}

		callHandlerParams := callHandlerParams{
			handlerType: handlerType,

			req: req,
			rw:  res,
		}

		switch handlerType {
		case handlerKitwebHandler:
			kitwebContext := reflect.New(handler.(reflect.Type).In(0).Elem())
			internalContext := kitwebContext.Interface().(internalCtx)

			internalContext.init(req, res, r.webModule.paramsBinder, r.webModule.paramsValidator)

			callHandlerParams.customHandler = h
			callHandlerParams.customHandlerArgs = []reflect.Value{kitwebContext}
		case handlerStdHttpHandler:
			callHandlerParams.stdHandler = handler.(http.Handler)
		default:
			panic("should not happen: invalid handlerType")
		}

		response, err := r.callHandler(callHandlerParams)

		if res.alreadyWritten {
			return
		}

		r.postCallHandler(postCallHandlerParams{
			rw:          rw,
			req:         req,
			panicked:    err,
			response:    response,
			handlerType: handlerType,
		})
	}), nil
}

func (r *Router) toMiddlewareHandler(h any) (mux.MiddlewareFunc, error) {
	handlerType := handlerStdMiddleware
	var handler any

	if value, ok := h.(func(http.Handler) http.Handler); ok {
		handler = mux.MiddlewareFunc(value)
	} else if value, ok := h.(mux.MiddlewareFunc); ok {
		handler = value
	} else {
		functionType := reflect.TypeOf(h)
		err := r.isMiddlewareHandlerFunc(functionType)
		if err != nil {
			return nil, fmt.Errorf(""+
				"function handlerType should be a mux.MiddlewareFunc or <func(r *Ctx[T], next http.HandlerFunc) Res> : %w",
				err,
			)
		}

		handlerType = handlerKitwebMiddleware
		handler = functionType
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			var res *wrappedResponseWriter
			if value, ok := rw.(*wrappedResponseWriter); ok {
				res = value
			} else {
				res = &wrappedResponseWriter{ResponseWriter: rw}
			}

			callHandlerParams := callHandlerParams{
				handlerType: handlerType,

				req: req,
				rw:  res,
			}

			switch handlerType {
			case handlerKitwebMiddleware:
				kitwebContext := reflect.New(handler.(reflect.Type).In(0).Elem())
				internalContext := kitwebContext.Interface().(internalCtx)

				internalContext.init(req, res, r.webModule.paramsBinder, r.webModule.paramsValidator)

				callHandlerParams.customHandler = h
				callHandlerParams.customHandlerArgs = []reflect.Value{kitwebContext, reflect.ValueOf(next)}
			case handlerStdMiddleware:
				callHandlerParams.stdMiddlewareHandler = handler.(mux.MiddlewareFunc)
				callHandlerParams.stdHandler = next
			default:
				panic("should not happen: invalid handlerType")
			}

			response, err := r.callHandler(callHandlerParams)

			if res.alreadyWritten {
				return
			}

			r.postCallHandler(postCallHandlerParams{
				rw:          rw,
				req:         req,
				panicked:    err,
				response:    response,
				handlerType: handlerType,
			})
		})
	}, nil
}

type postCallHandlerParams struct {
	rw  http.ResponseWriter
	req *http.Request

	// will be a panicked error or nil
	panicked error

	// can be null in the case of middleware, panic, no content from developer handlerType or std handlerType/middleware
	response Res

	handlerType handlerType
}

// postCallHandler will handle the response of a handlerType
// it will call the panicHandler if a panic occurred
// it will call the noContentHandler if the handlerType returned nothing
// it will write the response if the handlerType returned a Res
func (r *Router) postCallHandler(p postCallHandlerParams) {
	// set contextual values that can be used by the Res or ExceptionHandlerFunc
	ctx := context.WithValue(p.req.Context(), ContextKeyEnv, r.webModule.env)
	ctx = context.WithValue(ctx, ContextKeyEngines, r.webModule.engines)

	p.req = p.req.WithContext(ctx)

	if p.panicked == nil && p.response == nil && p.handlerType.isHandler() {
		// no content because a handlerType must return something (a middleware can return nothing)
		r.webModule.config.NoContentHandler(p.rw, p.req, nil)
		return
	}

	if p.panicked != nil {
		// a panic occurred
		r.webModule.config.PanicHandler(p.rw, p.req, p.panicked)
		return
	}

	if p.response == nil && (p.handlerType.isMiddleware() || !p.handlerType.isCustom()) {
		// a middleware can return nothing because it might have call the next handlerType
		return
	}

	if err := p.response.Write(ctx, p.rw); err != nil {
		r.webModule.config.PanicHandler(p.rw, p.req, errors.Join(err, errWrite))
		return
	}

	return
}

type callHandlerParams struct {
	handlerType handlerType

	customHandler     any
	customHandlerArgs []reflect.Value

	stdHandler           http.Handler
	stdMiddlewareHandler mux.MiddlewareFunc

	rw  http.ResponseWriter
	req *http.Request
}

func (r *Router) callHandler(p callHandlerParams) (res Res, err error) {
	defer func() {
		recoverErr := recover()
		if recoverErr != nil {
			p := panicked{recoverErr, string(debug.Stack()), p.handlerType}

			p.print(r.env, r.logger)

			err = p
		}
	}()

	var response any

	if p.handlerType.isCustom() {
		rets := reflect.ValueOf(p.customHandler).Call(p.customHandlerArgs)
		response = rets[0].Interface()
	} else if p.handlerType == handlerStdHttpHandler {
		p.stdHandler.ServeHTTP(p.rw, p.req)
	} else {
		p.stdMiddlewareHandler(p.stdHandler).ServeHTTP(p.rw, p.req)
	}

	if response == nil {
		return nil, nil
	}

	return response.(Res), nil

}

func (r *Router) isHandlerFunc(functionType reflect.Type) error {
	if !kitreflect.EnsureInOutLength(functionType, 1, 1) {
		return fmt.Errorf("invalid handlerType length: %s", functionType.String())
	}

	err := isCtxParam(functionType)
	if err != nil {
		return err
	}

	if !functionType.Out(0).AssignableTo(reflect.TypeOf((*Res)(nil)).Elem()) {
		return fmt.Errorf("invalid handlerType return type: %s", functionType.String())
	}
	return nil
}

func (r *Router) isMiddlewareHandlerFunc(functionType reflect.Type) error {
	if functionType.Kind() != reflect.Func {
		return fmt.Errorf("invalid handler type type, should be a function: %s", functionType.String())
	}

	if !kitreflect.EnsureInOutLength(functionType, 2, 1) {
		return fmt.Errorf("invalid handler type length: %s", functionType.String())
	}

	err := isCtxParam(functionType)
	if err != nil {
		return err
	}

	err = isHttpFuncParam(functionType)
	if err != nil {
		return err
	}

	if !functionType.Out(0).AssignableTo(reflect.TypeOf((*Res)(nil)).Elem()) {
		return fmt.Errorf("invalid handler type return type: %s", functionType.String())
	}

	return nil
}

// isCtxParam will check if the first parameter of the function is a *kitweb.Ctx
func isCtxParam(functionType reflect.Type) error {
	if functionType.In(0).Kind() != reflect.Ptr {
		return fmt.Errorf("invalid handler type 1st arg: %s", functionType.String())
	}

	if functionType.In(0).Elem().Kind() != reflect.Struct {
		return fmt.Errorf("invalid handler type 1st arg type: %s", functionType.String())
	}

	if !strings.HasPrefix(functionType.In(0).String(), "*kitweb.Ctx") {
		return fmt.Errorf("invalid handler type 1st arg type: %s", functionType.String())
	}
	return nil
}

// isHttpFuncParam will check if the second parameter of the function is a http.HandlerFunc
func isHttpFuncParam(functionType reflect.Type) error {
	if !strings.HasPrefix(functionType.In(1).String(), "http.HandlerFunc") {
		return fmt.Errorf("invalid handler type 2nd arg type: %s", functionType.String())
	}

	return nil
}
