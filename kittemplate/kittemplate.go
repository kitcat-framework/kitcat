package kittemplate

import (
	"github.com/expectedsh/dig"
	"github.com/expectedsh/kitcat"
	"github.com/expectedsh/kitcat/kitdi"
	"io"
)

type (
	EngineOptions struct {
		Data   any
		Layout *string
	}

	EngineOption func(*EngineOptions)

	Engine interface {
		Execute(writer io.Writer, templateName string, options ...EngineOption) error
		kitcat.Nameable
	}

	Engines struct {
		dig.In
		Engines []Engine `group:"kittemplate.engine"`
	}
)

func ProvideEngine(a any) *kitdi.Annotation {
	return kitdi.Annotate(a, kitdi.Group("kittemplate.engine"), kitdi.As((*Engine)(nil)))
}
