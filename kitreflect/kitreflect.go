package kitreflect

import (
	"context"
	"reflect"
)

func GetFullTypeName(t reflect.Type) string {
	return t.String()
}

func EnsureInOutLength(t reflect.Type, in, out int) bool {
	return t.NumIn() == in && t.NumOut() == out
}

func EnsureMinParams(t reflect.Type, min int) bool {
	return t.NumIn() >= min
}

func EnsureOutIsError(t reflect.Type) bool {
	return t.Out(t.NumOut()-1).Name() == "error"
}

func IsContext(t reflect.Type) bool {
	return t.AssignableTo(reflect.TypeOf((*context.Context)(nil)).Elem())
}

func EnsureInIsContext(t reflect.Type) bool {
	return IsContext(t.In(0))
}
