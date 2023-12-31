package kitweb

import (
	"errors"
	"net/http"
)

type baseRenderBuilder[T Res] struct {
	statusCode int
	headers    http.Header
	error      error

	ret T
}

func (r *baseRenderBuilder[T]) StatusCode(statusCode int) T {
	r.statusCode = statusCode
	return r.ret
}

func (r *baseRenderBuilder[T]) Headers(headers http.Header) T {
	if r.headers == nil {
		r.headers = http.Header{}
	}

	for key, value := range headers {
		r.headers[key] = value
	}
	return r.ret
}

// Err set the error of the response
func (r *baseRenderBuilder[T]) Err(error error) T {
	r.error = error
	return r.ret
}

func (r *baseRenderBuilder[T]) Header(key, value string) T {
	if r.headers == nil {
		r.headers = http.Header{}
	}
	r.headers.Set(key, value)
	return r.ret
}

func (r *baseRenderBuilder[T]) ContentType(contentType string) T {
	return r.Header("Content-Type", contentType)
}

func (r *baseRenderBuilder[T]) write(w http.ResponseWriter) {
	if r.headers != nil {
		for key, value := range r.headers {
			w.Header()[key] = value
		}
	}

	var validationError ValidationError
	if errors.As(r.error, &validationError) && r.statusCode == http.StatusOK {
		r.statusCode = http.StatusBadRequest
	} else if r.error != nil && r.statusCode == http.StatusOK {
		r.statusCode = http.StatusInternalServerError
	}

	w.WriteHeader(r.statusCode)
}

func newBaseRenderBuilder[T Res](ret T) *baseRenderBuilder[T] {
	return &baseRenderBuilder[T]{ret: ret}
}
