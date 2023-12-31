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
		ContentType("text/html; charset=utf-8").
		StatusCode(http.StatusOK)
}

func (r *RenderViewBuilder) Data(data any) *RenderViewBuilder {
	r.data = data
	return r
}

func (r *RenderViewBuilder) Engine(engine string) *RenderViewBuilder {
	r.engine = engine
	return r
}

func (r *RenderViewBuilder) Layout(layout string) *RenderViewBuilder {
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

func (r RenderData) Error() (*Err, bool) {
	var err *Err
	ok := errors.As(r.Err, &err)
	return err, ok
}

func (r *RenderViewBuilder) Write(ctx context.Context, w http.ResponseWriter) error {
	engine := ctx.Value(ContextKeyEngines).(map[string]kittemplate.Engine)[r.engine]

	opts := make([]kittemplate.EngineOption, 0)
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
