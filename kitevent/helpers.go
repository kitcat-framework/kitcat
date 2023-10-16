package kitevent

import (
	"context"
	"github.com/expectedsh/kitcat"
	"github.com/expectedsh/kitcat/kitreflect"
	"reflect"
)

func IsHandler(handler kitcat.Nameable) bool {
	handleFunc := reflect.ValueOf(handler).MethodByName("Handle")
	if handleFunc.Kind() != reflect.Func {
		return false
	}

	if !kitreflect.EnsureInOutLength(handleFunc.Type(), 2, 1) {
		return false
	}

	if !kitreflect.EnsureInIsContext(handleFunc.Type()) {
		return false
	}

	if !kitreflect.EnsureOutIsError(handleFunc.Type()) {
		return false
	}

	if !handleFunc.Type().In(1).AssignableTo(reflect.TypeOf((*Event)(nil)).Elem()) {
		return false
	}

	return true
}

func CallHandler(ctx context.Context, handler kitcat.Nameable, event Event) error {
	handleFunc := reflect.ValueOf(handler).MethodByName("Handle")
	ret := handleFunc.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(event)})

	if len(ret) > 0 && !ret[0].IsNil() {
		return ret[0].Interface().(error)
	}

	return nil
}
