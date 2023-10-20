package kitweb

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/expectedsh/kitcat"
	"github.com/expectedsh/kitcat/kitconfig"
	"github.com/expectedsh/kitcat/kitslog"
	"github.com/expectedsh/kitcat/kittemplate"
	"github.com/expectedsh/kitcat/kitweb/httpbind"
	"log/slog"
	"net"
	"net/http"
	"time"
)

type Config struct {
	// Addr optionally specifies the TCP address for the server to listen on,
	// Example: ":8080"
	// If empty, ":http" (port 80) is used.
	Addr string `env:"KITWEB_ADDR" envDefault:":8080"`

	// Below is almost a copy of the http.Server struct

	// DisableGeneralOptionsHandler, if true, passes "OPTIONS *" requests to the CustomHandler,
	// otherwise responds with 200 OK and Content-Length: 0.
	DisableGeneralOptionsHandler bool

	// TLSConfig optionally provides a TLS configuration for use
	// by ServeTLS and ListenAndServeTLS. Note that this value is
	// cloned by ServeTLS and ListenAndServeTLS, so it's not
	// possible to modify the configuration with methods like
	// tls.GoHTMLEngineConfig.SetSessionTicketKeys. To use
	// SetSessionTicketKeys, use Server.Serve with a TLS Listener
	// instead.
	TLSConfig *tls.Config

	// ReadTimeout is the maximum duration for reading the entire
	// request, including the body. A zero or negative value means
	// there will be no timeout.
	//
	// Because ReadTimeout does not let Handlers make per-request
	// decisions on each request body's acceptable deadline or
	// upload rate, most users will prefer to use
	// ReadHeaderTimeout. It is valid to use them both.
	ReadTimeout time.Duration

	// ReadHeaderTimeout is the amount of time allowed to read
	// request headers. The connection's read deadline is reset
	// after reading the headers and the CustomHandler can decide what
	// is considered too slow for the body. If ReadHeaderTimeout
	// is zero, the value of ReadTimeout is used. If both are
	// zero, there is no timeout.
	ReadHeaderTimeout time.Duration

	// WriteTimeout is the maximum duration before timing out
	// writes of the response. It is reset whenever a new
	// request's header is read. Like ReadTimeout, it does not
	// let Handlers make decisions on a per-request basis.
	// A zero or negative value means there will be no timeout.
	WriteTimeout time.Duration

	// IdleTimeout is the maximum amount of time to wait for the
	// next request when keep-alives are enabled. If IdleTimeout
	// is zero, the value of ReadTimeout is used. If both are
	// zero, there is no timeout.
	IdleTimeout time.Duration

	// MaxHeaderBytes controls the maximum number of bytes the
	// server will read parsing the request header's keys and
	// values, including the request line. It does not limit the
	// size of the request body.
	// If zero, DefaultMaxHeaderBytes is used.
	MaxHeaderBytes int

	AdditionalValueExtractors []httpbind.ValueParamExtractor
	AdditionalStringExtractor []httpbind.StringsParamExtractor

	TemplateEngineName string `env:"KITWEB_TEMPLATE_ENGINE_NAME" envDefault:"gohtml"`
}

type Module struct {
	config *Config
	router *Router

	logger *slog.Logger

	paramsBinder    ParamsBinder
	paramsValidator ParamsValidator

	httpServer *http.Server

	engines map[string]kittemplate.Engine
}

// New returns a new Web
func New(config *Config) func(a *kitcat.App) {
	return func(a *kitcat.App) {
		w := &Module{
			config: config,
			logger: slog.With(kitslog.Module("kitweb")),
		}

		valueExtractors := append(httpbind.ValuesParamExtractors, config.AdditionalValueExtractors...)
		stringExtractors := append(httpbind.StringsParamExtractors, config.AdditionalStringExtractor...)

		w.router = newRouter(w)

		w.paramsBinder = httpbind.NewBinder(stringExtractors, valueExtractors)
		w.paramsValidator = GetValidator(w.paramsBinder.GetParsableTags())

		a.Provides(
			w,
			kitcat.ModuleAnnotation(w),
			w.config,
			w.router,
		)
	}
}

func (w *Module) OnStart(_ context.Context, app *kitcat.App) error {
	app.Invoke(w.registerHandlers)
	w.setTemplateEngine(app)
	w.httpServer = w.buildHTPServerFromConfig()

	addr := w.config.Addr
	if addr == "" {
		addr = ":http"
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("kitweb: error while starting http server: %w", err)
	}

	w.logger.Info("starting http server", slog.String("addr", listener.Addr().String()))

	go w.httpServer.Serve(listener)

	return nil
}

func (w *Module) OnStop(ctx context.Context, _ *kitcat.App) error {
	w.logger.Info("stopping http server")

	return w.httpServer.Shutdown(ctx)
}

func (w *Module) Name() string {
	return "kitweb"
}

func (w *Module) buildHTPServerFromConfig() *http.Server {
	srv := &http.Server{Handler: w.router.handler}

	if w.config.TLSConfig != nil {
		srv.TLSConfig = w.config.TLSConfig
	}

	if w.config.ReadTimeout != 0 {
		srv.ReadTimeout = w.config.ReadTimeout
	}

	if w.config.ReadHeaderTimeout != 0 {
		srv.ReadHeaderTimeout = w.config.ReadHeaderTimeout
	}

	if w.config.WriteTimeout != 0 {
		srv.WriteTimeout = w.config.WriteTimeout
	}

	if w.config.IdleTimeout != 0 {
		srv.IdleTimeout = w.config.IdleTimeout
	}

	if w.config.MaxHeaderBytes != 0 {
		srv.MaxHeaderBytes = w.config.MaxHeaderBytes
	}

	return srv
}

func (w *Module) setTemplateEngine(app *kitcat.App) {
	app.Invoke(func(engines kittemplate.Engines) {
		// if no template engine is provided, we provide the default one
		if engines.Engines == nil {
			app.Provides(kittemplate.NewGoHTMLTemplateEngine(
				kitconfig.FromEnv[kittemplate.GoHTMLEngineConfig]()),
			)
		}
	})

	app.Invoke(func(engines kittemplate.Engines) error {
		for _, engine := range engines.Engines {
			w.engines[engine.Name()] = engine
		}

		return nil
	})
}

func (w *Module) registerHandlers(handlers handlers) {
	if len(handlers.Handlers) == 0 {
		return
	}

	w.logger.Info("registering handlers", slog.Int("count", len(handlers.Handlers)))
	for _, h := range handlers.Handlers {
		w.logger.Info("registering handler", slog.String("handler", h.Name()))
		h.Routes(w.router)
	}
}
