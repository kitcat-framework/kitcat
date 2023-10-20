package kitweb

import (
	"errors"
	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
	"reflect"
)

// GetValidator allow to set the default paramsValidator builder
var GetValidator = getDefaultValidator

type GoPlaygroundParamsValidator struct {
	validate *validator.Validate
	trans    ut.Translator
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
		validationError.Errors[ve.Field()] = NewError(ve.Tag(), ve.Translate(g.trans), nil)
	}

	return validationError
}

func getDefaultValidator(tags []string) ParamsValidator {
	validate := validator.New(validator.WithRequiredStructEnabled())
	validate.RegisterTagNameFunc(func(field reflect.StructField) string {
		tags = append(tags, "json", "xml")
		for _, extractor := range tags {
			if val := field.Tag.Get(extractor); val != "" {
				return val
			}
		}

		return field.Name
	})

	en := en.New()
	uni := ut.New(en, en)
	trans, _ := uni.GetTranslator("en")
	en_translations.RegisterDefaultTranslations(validate, trans)

	return &GoPlaygroundParamsValidator{validate: validate, trans: trans}
}
