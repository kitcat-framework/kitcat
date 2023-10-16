package kitweb

import (
	"errors"
	"github.com/go-playground/validator/v10"
	"reflect"
)

// GetValidator allow to set the default paramsValidator builder
var GetValidator = getDefaultValidator

type GoPlaygroundParamsValidator struct {
	validate *validator.Validate
}

func (g GoPlaygroundParamsValidator) Validate(a any) error {
	err := g.validate.Struct(a)
	if err == nil {
		return nil
	}

	var validationErrors validator.ValidationErrors
	ok := errors.As(err, &validationErrors)
	if !ok {
		return err
	}

	validationError := ValidationError{error: err, Errors: make(map[string]*Error)}
	for _, ve := range validationErrors {
		validationError.Errors[ve.Field()] = NewError(ve.Tag(), ve.Error(), nil)
	}

	return validationError
}

func getDefaultValidator(tags []string) ParamsValidator {
	validate := validator.New(validator.WithRequiredStructEnabled())
	validate.RegisterTagNameFunc(func(field reflect.StructField) string {
		for _, extractor := range tags {
			if val := field.Tag.Get(extractor); val != "" {
				return val
			}
		}

		return field.Name
	})

	return &GoPlaygroundParamsValidator{validate: validate}
}
