package kittemplate

import (
	"github.com/expectedsh/dig"
	"github.com/expectedsh/kitcat"
	"io"
)

type (
	EngineOptions struct {
		Data   any
		Layout *string
	}

	EngineOptsApplier func(*EngineOptions)

	Engine interface {
		Execute(writer io.Writer, templateName string, options ...EngineOptsApplier) error
		kitcat.Nameable
	}

	Engines struct {
		dig.In
		Engines []Engine `group:"kittemplate.engine"`
	}

	ProvidableEngine struct {
		dig.Out
		Engine Engine `group:"kittemplate.engine"`
	}
)

func NewProvidableEngine(engine Engine) ProvidableEngine {
	return ProvidableEngine{Engine: engine}
}
