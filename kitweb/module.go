package kitweb

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/expectedsh/kitcat"
	"github.com/expectedsh/kitcat/kitdi"
	"github.com/expectedsh/kitcat/kitslog"
	"github.com/expectedsh/kitcat/kittemplate"
	"github.com/expectedsh/kitcat/kitweb/httpbind"
	"github.com/spf13/viper"
	"log/slog"
	"net"
	"net/http"
	"time"
)

type Config struct {
	// Addr optionally specifies the TCP address for the server to listen on,
	// Example: ":8080"
	// If empty, ":http" (port 80) is used.
	Addr string `cfg:"addr"`

	// Below is almost a copy of the http.Server struct

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
	// Because ReadTimeout does not let handlers make per-request
	// decisions on each request body's acceptable deadline or
	// upload rate, most users will prefer to use
	// ReadHeaderTimeout. It is valid to use them both.
	ReadTimeout time.Duration `cfg:"read_timeout"`

	// ReadHeaderTimeout is the amount of time allowed to read
	// request headers. The connection's read deadline is reset
	// after reading the headers and the CustomHandler can decide what
	// is considered too slow for the body. If ReadHeaderTimeout
	// is zero, the value of ReadTimeout is used. If both are
	// zero, there is no timeout.
	ReadHeaderTimeout time.Duration `cfg:"read_header_timeout"`

	// WriteTimeout is the maximum duration before timing out
	// writes of the response. It is reset whenever a init
	// request's header is read. Like ReadTimeout, it does not
	// let handlers make decisions on a per-request basis.
	// A zero or negative value means there will be no timeout.
	WriteTimeout time.Duration `cfg:"write_timeout"`

	// IdleTimeout is the maximum amount of time to wait for the
	// next request when keep-alives are enabled. If IdleTimeout
	// is zero, the value of ReadTimeout is used. If both are
	// zero, there is no timeout.
	IdleTimeout time.Duration `cfg:"idle_timeout"`

	// MaxHeaderBytes controls the maximum number of bytes the
	// server will read parsing the request header's keys and
	// values, including the request line. It does not limit the
	// size of the request body.
	// If zero, DefaultMaxHeaderBytes is used.
	MaxHeaderBytes int `cfg:"max_header_bytes"`

	AdditionalValueExtractors []httpbind.ValueParamExtractor
	AdditionalStringExtractor []httpbind.StringsParamExtractor

	TemplateEngineName string `cfg:"template_engine_name"`

	// PublicFolder is the folder where the static files are located
	PublicFolder string `cfg:"public_folder"`
	PublicPath   string `cfg:"public_path"`

	panicHandler      ExceptionHandlerFunc
	noContentHandler  ExceptionHandlerFunc
	notFoundHandler   ExceptionHandlerFunc
	notAllowedHandler ExceptionHandlerFunc
}

func (c *Config) InitConfig(prefix string) kitcat.ConfigUnmarshal {
	prefix = prefix + ".kitweb"

	viper.SetDefault(prefix+".addr", ":8080")
	viper.SetDefault(prefix+".read_timeout", 0)
	viper.SetDefault(prefix+".read_header_timeout", 0)
	viper.SetDefault(prefix+".write_timeout", 0)
	viper.SetDefault(prefix+".idle_timeout", 0)
	viper.SetDefault(prefix+".max_header_bytes", 0)
	viper.SetDefault(prefix+".template_engine_name", "gohtml")
	viper.SetDefault(prefix+".public_folder", "public")
	viper.SetDefault(prefix+".public_path", "/public/")

	c.panicHandler = panicHandler
	c.noContentHandler = noContentHandler
	c.notFoundHandler = notFoundHandler
	c.notAllowedHandler = methodNotAllowedHandler

	return kitcat.ConfigUnmarshalHandler(prefix, c, "unable to unmarshal kitweb config: %w")
}

func init() {
	kitcat.RegisterConfig(new(Config))
}

type Module struct {
	config       *Config
	globalRouter *Router

	logger *slog.Logger

	paramsBinder    ParamsBinder
	paramsValidator ParamsValidator

	httpServer *http.Server

	engines map[string]kittemplate.Engine

	env *kitcat.Environment
}

// New create the kitweb module
func New(_ kitdi.Invokable, config *Config, a *kitcat.App, env *kitcat.Environment) {
	w := &Module{
		config:  config,
		logger:  slog.With(kitslog.Module("kitweb")),
		engines: map[string]kittemplate.Engine{},
		env:     env,
	}

	valueExtractors := append(httpbind.ValuesParamExtractors, config.AdditionalValueExtractors...)
	stringExtractors := append(httpbind.StringsParamExtractors, config.AdditionalStringExtractor...)

	w.globalRouter = newRouter("global", w, env, func(r *Router) {
		r.initPublicFolder()
	})

	w.paramsBinder = httpbind.NewBinder(stringExtractors, valueExtractors)
	w.paramsValidator = GetValidator(w.paramsBinder.GetParsableTags())

	a.Provides(
		w,
		kitcat.ModuleAnnotation(w),
		w.globalRouter,
	)

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
	srv := &http.Server{Handler: w.globalRouter.handler}

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
		if len(engines.Engines) == 0 {
			app.Provides(kittemplate.ProvideEngine(kittemplate.NewGoHTMLTemplateEngine))
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

	w.globalRouter.handler.NotFoundHandler = http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		w.config.notFoundHandler(rw, req, nil)
	})

	w.globalRouter.handler.MethodNotAllowedHandler = http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		w.config.notAllowedHandler(rw, req, nil)
	})

	for _, h := range handlers.Handlers {
		w.logger.Info("registering handlerType", slog.String("handlerType", h.Name()))

		newRouter(h.Name(), w, w.env, func(r *Router) {
			r.handler = w.globalRouter.handler.PathPrefix("/").Subrouter()
		})

		h.Routes(w.globalRouter)
	}
}
