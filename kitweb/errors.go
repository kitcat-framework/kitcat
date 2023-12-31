package kitweb

type ErrDesc struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Meta    map[string]any `json:"meta,omitempty"`
}

type Err struct {
	ErrDesc
	error
}

func Error(code, message string, err error) *Err {
	return &Err{
		ErrDesc: ErrDesc{
			Code:    code,
			Message: message,
			Meta:    make(map[string]any),
		},
		error: err,
	}
}

// InternalError is a default implementation of DisplayableError
func InternalError(err error) *Err {
	return Error("internal_error", "an internal error occurred", err)
}

// NotFoundError is a default implementation of DisplayableError
func NotFoundError(err error) *Err {
	return Error("not_found", "resource not found", err)
}

// BadRequestError is a default implementation of DisplayableError
func BadRequestError(err error) *Err {
	return Error("bad_request", "bad request", err)
}

// ValidationError is a dedicated error for validation errors where
// the error can be displayed to the end user with the field name
type ValidationError struct {
	Errors map[string]*Err `json:"errors"`
	Global *Err            `json:"global,omitempty"`

	error
}
