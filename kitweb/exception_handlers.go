package kitweb

import (
	_ "embed"
	"errors"
	"github.com/expectedsh/kitcat"
	"html/template"
	"net/http"
	"strings"
)

//go:embed unexpected_error.gohtml
var unexpectedErrorTemplate string

type unexpectedErrorData struct {
	URL  string
	Path string
	Verb string

	StackTrace string
	Error      error
}

func panicHandler(rw http.ResponseWriter, req *http.Request, err error) {
	contentType := req.Header.Get("Content-Type")
	var stack panicked
	env := req.Context().Value(ContextKeyEnv).(*kitcat.Environment)

	if strings.Contains(contentType, "application/json") {
		e := Error("unexpected_error", "an unexpected error occurred", err)
		if !env.Equal(kitcat.EnvironmentProduction) && errors.As(err, &stack) {
			e.Meta["stack_trace"] = strings.Split(stack.StackTrace, "\n")
			e.Meta["origin_error"] = stack.error
		}

		_ = JSONRender().Err(e).StatusCode(http.StatusInternalServerError).Write(req.Context(), rw)
	} else {
		data := unexpectedErrorData{
			URL:   req.URL.String(),
			Verb:  req.Method,
			Error: err,
		}

		if !env.Equal(kitcat.EnvironmentProduction) && errors.As(err, &stack) {
			data.StackTrace = stack.StackTrace
		}

		t := template.Must(template.New("unexpected_error").Parse(unexpectedErrorTemplate))
		rw.WriteHeader(http.StatusInternalServerError)
		_ = t.Execute(rw, data)
	}
}

func noContentHandler(rw http.ResponseWriter, req *http.Request, _ error) {
	contentType := req.Header.Get("Content-Type")

	if strings.Contains(contentType, "application/json") {
		_ = JSONRender().StatusCode(http.StatusNoContent).Write(req.Context(), rw)
	} else {
		rw.WriteHeader(http.StatusNoContent)
	}
}

func notFoundHandler(rw http.ResponseWriter, req *http.Request, _ error) {
	contentType := req.Header.Get("Content-Type")

	if strings.Contains(contentType, "application/json") {
		_ = JSONRender().Err(NotFoundError(errors.New("not found"))).StatusCode(http.StatusNotFound).Write(req.Context(),
			rw)
	} else {
		rw.WriteHeader(http.StatusNotFound)
	}
}

func methodNotAllowedHandler(rw http.ResponseWriter, req *http.Request, _ error) {
	contentType := req.Header.Get("Content-Type")

	if strings.Contains(contentType, "application/json") {
		_ = JSONRender().Err(Error("method_not_allowed", "method not allowed", errors.New("method not allowed"))).
			StatusCode(http.StatusMethodNotAllowed).Write(req.Context(), rw)
	} else {
		rw.WriteHeader(http.StatusMethodNotAllowed)
	}
}
