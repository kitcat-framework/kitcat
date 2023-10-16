package kitweb

import (
	"context"
	"errors"
	"github.com/expectedsh/kitcat/kittemplate"
	"net/http"
)

type RenderViewBuilder struct {
	engine string
	data   any
	name   string
	layout *string

	*baseRenderBuilder[*RenderViewBuilder]
}

func ViewRender(name string) *RenderViewBuilder {
	renderViewBuilder := &RenderViewBuilder{
		engine: "gohtml",
		name:   name,
	}

	renderViewBuilder.baseRenderBuilder = newBaseRenderBuilder[*RenderViewBuilder](renderViewBuilder)

	return renderViewBuilder.
		WithContentType("text/html; charset=utf-8").
		WithStatusCode(http.StatusOK)
}

type ctxKeyEngines struct{}

var ctxKeyEnginesValue = ctxKeyEngines{}

func (r *RenderViewBuilder) WithData(data any) *RenderViewBuilder {
	r.data = data
	return r
}

func (r *RenderViewBuilder) WithEngine(engine string) *RenderViewBuilder {
	r.engine = engine
	return r
}

func (r *RenderViewBuilder) WithLayout(layout string) *RenderViewBuilder {
	r.layout = &layout
	return r
}

// RenderData is the data passed to the template engine
//
// Data is any because anyway even jetbrain auto completion can't help you with generics type...
type RenderData struct {
	Data any

	Err error
}

func (r RenderData) ValidationError() (*ValidationError, bool) {
	var validationError *ValidationError
	ok := errors.As(r.Err, &validationError)
	return validationError, ok
}

func (r RenderData) Error() (*Error, bool) {
	var err *Error
	ok := errors.As(r.Err, &err)
	return err, ok
}

func (r *RenderViewBuilder) Write(ctx context.Context, w http.ResponseWriter) error {
	engine := ctx.Value(ctxKeyEnginesValue).(map[string]kittemplate.Engine)[r.engine]

	opts := make([]kittemplate.EngineOptsApplier, 0)
	if r.layout != nil {
		opts = append(opts, kittemplate.WithEngineOptLayout(*r.layout))
	}

	opts = append(opts, kittemplate.WithEngineOptData(RenderData{
		Data: r.data,
		Err:  r.error,
	}))

	r.baseRenderBuilder.write(w)

	return engine.Execute(w, r.name, opts...)
}
