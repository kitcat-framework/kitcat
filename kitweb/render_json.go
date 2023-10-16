package kitweb

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
)

type RenderJSONBuilder struct {
	data any

	*baseRenderBuilder[*RenderJSONBuilder]
}

func JSONRender() *RenderJSONBuilder {
	jsonBuilder := &RenderJSONBuilder{}
	jsonBuilder.baseRenderBuilder = newBaseRenderBuilder[*RenderJSONBuilder](jsonBuilder)

	return jsonBuilder.
		WithContentType("application/json; charset=utf-8").
		WithStatusCode(http.StatusOK)
}

func (r *RenderJSONBuilder) WithData(data any) *RenderJSONBuilder {
	r.data = data
	return r
}

func (r *RenderJSONBuilder) Write(_ context.Context, w http.ResponseWriter) error {
	response := make(map[string]any)

	var (
		ve  ValidationError
		err Error
	)
	if errors.As(r.error, &ve) {
		response["errors"] = ve.Errors

		if ve.Global != nil {
			response["global"] = ve.Global
		}
	} else if errors.As(r.error, &err) {
		response["error"] = err.Error()
	}

	if r.data != nil {
		response["data"] = r.data
	}

	r.baseRenderBuilder.write(w)

	return json.NewEncoder(w).Encode(r.data)
}
