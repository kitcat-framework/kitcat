package kitweb

import (
	"context"
	"errors"
	"github.com/kitcat-framework/kitcat/kittemplate"
	"net/http"
	"reflect"
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

type RenderData struct {
	Data any
	Err  error
}

// SetRenderData set the RenderData field of dest with rd.
func SetRenderData(dest any, rd RenderData) {
	// safe do that :
	//reflect.ValueOf(dest).Elem().FieldByName("RenderData").Set(reflect.ValueOf(rd))

	val := reflect.ValueOf(dest)

	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if !val.FieldByName("RenderData").IsValid() {
		return
	}

	if val.FieldByName("RenderData").Type() != reflect.TypeOf(rd) {
		return
	}

	val.FieldByName("RenderData").Set(reflect.ValueOf(rd))
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
