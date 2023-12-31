package kitweb

import (
	"github.com/google/uuid"
	"log/slog"
	"net/http"
	"time"
)

type loggerResponseWriter struct {
	http.ResponseWriter

	statusCode int
	body       []byte
}

func (l *loggerResponseWriter) WriteHeader(statusCode int) {
	l.statusCode = statusCode
	l.ResponseWriter.WriteHeader(statusCode)
}

func (l *loggerResponseWriter) Write(body []byte) (int, error) {
	l.body = body
	return l.ResponseWriter.Write(body)
}

func MiddlewareRequestIDSetter() Middleware[struct{}] {
	return func(r *Ctx[struct{}], next http.HandlerFunc) Res {
		id := r.Req.Header.Get("X-Request-Id")

		if id == "" {
			id = uuid.New().String()
		}

		r.SetRequestContextValue("request_id", id)
		r.GetResponse().Header().Set("X-Request-ID", id)

		next.ServeHTTP(r.GetResponse(), r.Req)

		panic("bonjour")

		return nil
	}
}

type MiddlewareLoggerParams struct {
	ID string `ctx:"request_id"`
}

func MiddlewareLogger() Middleware[MiddlewareLoggerParams] {
	return func(r *Ctx[MiddlewareLoggerParams], next http.HandlerFunc) Res {
		t := time.Now()
		res := &loggerResponseWriter{ResponseWriter: r.GetResponse()}

		next.ServeHTTP(res, r.Req)

		r.Logger().Info("",
			slog.String("status", http.StatusText(res.statusCode)),
			slog.String("content_type", res.Header().Get("Content-Type")),
			slog.String("request_id", r.Params().ID),
			slog.Duration("duration", time.Since(t)),
		)

		return nil
	}
}
