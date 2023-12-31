package kitweb

import (
	"github.com/google/uuid"
	"log/slog"
	"net/http"
	"time"
)

func MiddlewareRequestIDSetter(r *Ctx[struct{}], next http.HandlerFunc) Res {
	id := r.Req.Header.Get("X-Request-Id")

	if id == "" {
		id = uuid.New().String()
	}

	r.SetRequestContextValue("request_id", id)
	r.GetResponse().Header().Set("X-Request-ID", id)

	next.ServeHTTP(r.GetResponse(), r.Req)

	return nil
}

type MiddlewareLoggerParams struct {
	ID string `ctx:"request_id"`
}

func MiddlewareLogger() Middleware[MiddlewareLoggerParams] {
	return func(r *Ctx[MiddlewareLoggerParams], next http.HandlerFunc) Res {
		t := time.Now()

		next.ServeHTTP(r.res, r.Req)

		slogArgs := []any{
			slog.String("content_type", r.Req.Header.Get("Content-Type")),
			slog.String("request_id", r.Params().ID),
			slog.Duration("duration", time.Since(t)),
		}

		if wres, ok := r.res.(*wrappedResponseWriter); ok {
			slogArgs = append(slogArgs, slog.String("status", http.StatusText(wres.statusCode)))
		}

		r.Logger().Info("",
			slogArgs...,
		)

		return nil
	}
}
