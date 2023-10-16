package kittemplate

import "github.com/samber/lo"

func WithEngineOptLayout(layout string) EngineOptsApplier {
	return func(options *EngineOptions) {
		options.Layout = lo.ToPtr(layout)
	}
}

func WithEngineOptData(data any) EngineOptsApplier {
	return func(options *EngineOptions) {
		options.Data = data
	}
}
