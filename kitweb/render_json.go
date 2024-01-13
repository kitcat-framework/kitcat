package kitweb

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/kitcat-framework/kitcat"
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
		ContentType("application/json; charset=utf-8").
		StatusCode(http.StatusOK)
}

func (r *RenderJSONBuilder) Data(data any) *RenderJSONBuilder {
	r.data = data
	return r
}

func (r *RenderJSONBuilder) Write(ctx context.Context, w http.ResponseWriter) error {
	response := make(map[string]any)

	var (
		ve  ValidationError
		err *Err
	)

	env := ctx.Value(ContextKeyEnv).(*kitcat.Environment)

	if errors.As(r.error, &ve) {
		response["errors"] = ve.Errors

		if ve.Global != nil {
			response["global"] = ve.Global
		}
	} else if errors.As(r.error, &err) {
		response["error"] = err.Message
		response["code"] = err.Code

		if err.Meta != nil {
			response["meta"] = err.Meta
		}

		if !env.Equal(kitcat.EnvironmentProduction) && err.error != nil {
			response["origin_error"] = err.Error()
		}
	} else if r.error != nil {
		response["error"] = InternalError(r.error)
	}

	if r.data != nil {
		response["data"] = r.data
	}

	r.baseRenderBuilder.write(w)

	return json.NewEncoder(w).Encode(response)
}
