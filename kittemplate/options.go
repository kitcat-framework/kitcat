package kittemplate

import "github.com/samber/lo"

func WithEngineOptLayout(layout string) EngineOption {
	return func(options *EngineOptions) {
		options.Layout = lo.ToPtr(layout)
	}
}

func WithEngineOptData(data any) EngineOption {
	return func(options *EngineOptions) {
		options.Data = data
	}
}
