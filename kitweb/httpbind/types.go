package httpbind

import "reflect"

// IsUnmarshallable check if the type t is either a native, custom type or implement an unmarshaler.
func IsUnmarshallable(t reflect.Type) bool {
	return IsNative(t) || IsCustomType(t) || IsImplementingUnmarshaler(t)
}

// IsCustomType check the type is a custom type.
func IsCustomType(t reflect.Type) bool {
	for _, c := range Custom {
		if t.AssignableTo(c) {
			return true
		}
	}
	return false
}

// IsNative check if the type t is a native type.
func IsNative(t reflect.Type) bool {
	_, ok := Native[t.Kind()]
	return ok
}

// IsImplementingUnmarshaler check if the type t implement one of the possible unmarshalers.
func IsImplementingUnmarshaler(t reflect.Type) bool {
	if t.Kind() != reflect.Ptr {
		t = reflect.New(t).Type()
	}
	for _, u := range Unmarshalers {
		if t.Implements(u) {
			return true
		}
	}

	return false
}
