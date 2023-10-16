package kitweb

type ErrorDescription struct {
	Code    string
	Message string
	Meta    map[string]any
}

type Error struct {
	ErrorDescription
	error
}

func NewError(code, message string, err error) *Error {
	return &Error{
		ErrorDescription: ErrorDescription{
			Code:    code,
			Message: message,
		},
		error: err,
	}
}

// InternalError is a default implementation of DisplayableError
func InternalError(err error) *Error {
	return NewError("internal_error", "an internal error occurred", err)
}

// NotFoundError is a default implementation of DisplayableError
func NotFoundError(err error) *Error {
	return NewError("not_found", "resource not found", err)
}

// BadRequestError is a default implementation of DisplayableError
func BadRequestError(err error) *Error {
	return NewError("bad_request", "bad request", err)
}

// ValidationError is a dedicated error for validation errors where
// the error can be displayed to the end user with the field name
type ValidationError struct {
	Errors map[string]*Error
	Global *Error

	error
}
