package kitweb

import (
	"net/http"
)

type (
	ParamsValidator interface {
		Validate(a any) error
	}

	ParamsBinder interface {
		Bind(request *http.Request, params any) error
		GetParsableTags() []string
	}
)
