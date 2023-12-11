package kitdi

import (
	"errors"
	"github.com/expectedsh/dig"
	"reflect"
)

type Annotation struct {
	Group string
	Name  string
	As    []any

	Target any
}

type AnnotateOption func(options *Annotation)

func Group(group string) AnnotateOption {
	return func(options *Annotation) {
		options.Group = group
	}
}

func Name(name string) AnnotateOption {
	return func(options *Annotation) {
		options.Name = name
	}
}

func As(as ...any) AnnotateOption {
	return func(options *Annotation) {
		options.As = as
	}
}

func Annotate(target any, opts ...AnnotateOption) *Annotation {
	options := new(Annotation)

	if reflect.TypeOf(target).Kind() != reflect.Func {
		options.Target = Supply(target)
	} else {
		options.Target = target
	}

	for _, option := range opts {
		option(options)
	}

	return options
}

func (a *Annotation) Apply(c *dig.Container, opts ...dig.ProvideOption) error {
	if a.Group != "" {
		opts = append(opts, dig.Group(a.Group))
	}

	if a.Name != "" {
		opts = append(opts, dig.Name(a.Name))
	}

	if len(a.As) > 0 {
		opts = append(opts, dig.As(a.As...))
	}

	target := a.Target

	if sup, ok := target.(*Supplier); ok {
		return sup.Apply(c, opts...)
	}

	return c.Provide(a.Target, opts...)
}

type Supplier struct {
	Target any
}

func Supply(target any) *Supplier {
	return &Supplier{Target: target}
}

func (s *Supplier) Apply(c *dig.Container, opts ...dig.ProvideOption) error {
	switch s.Target.(type) {
	case nil:
		return errors.New("nil value passed to Supply")
	case error:
		return errors.New("error value passed to Supply")
	}

	typ := reflect.TypeOf(s.Target)
	returnTypes := []reflect.Type{typ}
	returnValues := []reflect.Value{reflect.ValueOf(s.Target)}

	ft := reflect.FuncOf([]reflect.Type{}, returnTypes, false)
	fv := reflect.MakeFunc(ft, func([]reflect.Value) []reflect.Value {
		return returnValues
	})

	return c.Provide(fv.Interface(), opts...)
}

type ProvidableInvoker struct {
	Target any
}

func ProvidableInvoke(target any) *ProvidableInvoker {
	return &ProvidableInvoker{Target: target}
}

func (p *ProvidableInvoker) Apply(c *dig.Container, _ ...dig.ProvideOption) error {
	return c.Invoke(p.Target)
}

type Applier interface {
	Apply(c *dig.Container, opts ...dig.ProvideOption) error
}

// Invokable is used to mark a function as invokable.
// Useful to avoir the need to call dig.Invoke on a function.
type Invokable struct{}
