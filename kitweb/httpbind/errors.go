package httpbind

import (
	"errors"
	"fmt"
)

var ErrInvalidParam = errors.New("invalid params: only ptr to a struct")

// ValidateParamsError is the error returned when a params is not either a pointer to a struct
type ValidateParamsError struct {
	error
}

// BindBodyError is the error returned when the body can't be binded.
type BindBodyError struct {
	error
	ContentType string
}

// ExtractError is the error returned when a value can't be extracted from the request.
type ExtractError struct {
	error
	Tag string
}

// FieldSetterError is the error returned when a value can't be set to the params.
type FieldSetterError struct {
	FieldSetterContext FieldSetterContext
	Message            string
}

// Error returns the error message.
func (f FieldSetterError) Error() string {
	return fmt.Sprintf("%s: %v", f.Message, f.FieldSetterContext)
}

type FieldSetterContext struct {
	Value            string `json:"value,omitempty"`
	FieldType        string `json:"field_type,omitempty"`
	ValueType        string `json:"value_type,omitempty"`
	Path             string `json:"path,omitempty"`
	ValueIndex       int    `json:"value_index,omitempty"`
	DecodingStrategy string `json:"decoding_strategy,omitempty"`
}

type Error struct {
	// Errors Can be one of : ValidateParamsError, BindBodyError, ExtractError, FieldSetterError
	Errors []error
}

func (b Error) Error() string {
	str := ""

	for _, err := range b.Errors {
		str += err.Error() + "\n"
	}

	return str
}
