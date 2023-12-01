package kitcfg

import (
	"dario.cat/mergo"
	"fmt"
	"github.com/expectedsh/kitcat/kitexit"
	"os"
	"reflect"
)

func EnvPtr[T any](key string, defaultValue *T) *T {
	element := new(T)

	if os.Getenv(key) == "" {
		return defaultValue
	}

	err := fillValue(key, element)
	if err != nil {
		kitexit.Abnormal(fmt.Errorf("kitconfig: error while filling value for env var %s: %w", key, err))
		return nil
	}

	return element
}

func Env[T any](key string, defaultValue T) T {
	element := new(T)

	if os.Getenv(key) == "" {
		return defaultValue
	}

	err := fillValue(key, element)
	if err != nil {
		kitexit.Abnormal(fmt.Errorf("kitconfig: error while filling value for env var %s: %w", key, err))
		return defaultValue
	}

	return *element
}

func FromEnv[T any](optionalMerge ...*T) *T {
	element := new(T)

	err := Parse(element)
	if err != nil {
		kitexit.Abnormal(fmt.Errorf("kitconfig: parsing type %s: %w",
			reflect.TypeOf(element).Elem().Name(),
			err,
		))
		return nil
	}

	if len(optionalMerge) > 0 {
		_ = mergo.Merge(element, optionalMerge[0], mergo.WithOverride)
	}

	return element
}

func fillValue[T any](key string, element *T) error {
	return set(
		reflect.ValueOf(element),
		reflect.StructField{
			Name: key,
			Tag:  "",
			Type: reflect.TypeOf(element),
		},
		os.Getenv(key),
		defaultTypeParsers(),
	)
}
