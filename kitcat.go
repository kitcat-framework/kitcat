package kitcat

import (
	"context"
	"github.com/expectedsh/dig"
)

type (
	Mod interface {
		OnStart(ctx context.Context, app *App) error
		OnStop(ctx context.Context, app *App) error
		Name() string
	}

	modules struct {
		dig.In
		Modules []Mod `group:"mod"`
	}

	ProvidableModule struct {
		dig.Out
		Mod Mod `group:"mod"`
	}
)

func NewProvidableModule(mod Mod) ProvidableModule {
	return ProvidableModule{Mod: mod}
}
