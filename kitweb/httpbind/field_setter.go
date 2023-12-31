package httpbind

import (
	"encoding"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"time"
)

type setterContext struct {
	metadata               FieldMetadata
	path, decodingStrategy string
	value                  interface{} // value is either []string or interface{}
}

type fieldSetter struct {
	field        reflect.Value
	value        interface{}
	metadata     FieldMetadata
	errorContext FieldSetterContext
}

func newFieldSetter(field reflect.Value, setterCtx setterContext) *fieldSetter {
	fs := &fieldSetter{field: field, value: setterCtx.value, metadata: setterCtx.metadata}

	fs.errorContext = FieldSetterContext{
		Value:     fmt.Sprint(setterCtx.value),
		ValueType: reflect.TypeOf(setterCtx.value).String(),
		FieldType: field.Type().String(),
		Path:      setterCtx.path,
	}

	return fs
}
func (f fieldSetter) set() error {
	if f.field.Type().AssignableTo(reflect.TypeOf(f.value)) {
		f.field.Set(reflect.ValueOf(f.value))
		return nil
	}

	// because what we got from extractor are []string
	// but in some cases we can have:
	// - string usable with TextUnmarshaller
	// - []byte usable with BinaryUnmarshaller
	list, ok := f.value.([]string)
	if !ok {
		switch t := f.value.(type) {
		case string:
			list = []string{t}
		case []byte:
			list = []string{string(t)}
		default:
			return &FieldSetterError{
				Message:            "incompatible type",
				FieldSetterContext: f.errorContext,
			}
		}
	}

	if len(list) == 0 {
		return nil
	}

	isPtr := f.field.Kind() == reflect.Ptr

	if f.metadata.ArrayOrSlice && !f.metadata.ImplementUnmarshaller {
		return f.setForArrayOrSlice(isPtr, list)
	}

	return f.setForNormalType(list[0], isPtr)
}

func (f fieldSetter) setForArrayOrSlice(ptr bool, list []string) error {
	var (
		element reflect.Value
		array   bool
	)

	isTypePtr := false
	elemType := f.field.Type()
	array = f.field.Type().Kind() == reflect.Array

	if ptr {
		array = f.field.Type().Elem().Kind() == reflect.Array
		elemType = elemType.Elem()
		isTypePtr = f.field.Type().Elem().Elem().Kind() == reflect.Ptr
	} else {
		isTypePtr = f.field.Type().Elem().Kind() == reflect.Ptr
	}

	if array {
		element = reflect.New(elemType)
	} else {
		element = reflect.MakeSlice(elemType, 0, len(list))
	}

	addToElem := func(index int, i reflect.Value) {
		if array {
			element.Elem().Index(index).Set(i)
		} else {
			element = reflect.Append(element, i)
		}
	}

	for i, val := range list {
		switch {
		case IsCustomType(f.metadata.Type):
			native, err := f.makeCustomType(val, isTypePtr)
			if err != nil {
				err.FieldSetterContext.ValueIndex = i
				return err
			}

			addToElem(i, native)
		case IsImplementingUnmarshaler(f.metadata.Type):
			withUnmarshaller, err := f.makeWithUnmarshaller(val)
			if err != nil {
				err.FieldSetterContext.ValueIndex = i
				return err
			}

			addToElem(i, withUnmarshaller)
		case IsNative(f.metadata.Type):
			native, err := f.makeNative(val, isTypePtr)
			if err != nil {
				err.FieldSetterContext.ValueIndex = i
				return err
			}

			addToElem(i, native)
		default:
			f.errorContext.ValueIndex = i
			return &FieldSetterError{
				Message:            "type is not native, unmarshaller or custom type",
				FieldSetterContext: f.errorContext,
			}
		}
	}

	setToField := func(value reflect.Value) {
		if element.Kind() == reflect.Ptr && element.Elem().Kind() == reflect.Array {
			value.Set(element.Elem())
		} else {
			value.Set(element)
		}
	}

	if ptr {
		v := reflect.New(f.field.Type().Elem())
		setToField(v.Elem())
		f.field.Set(v)
	} else {
		setToField(f.field)
	}

	return nil
}

func (f fieldSetter) setForNormalType(str string, ptr bool) error {
	switch {
	case IsCustomType(f.metadata.Type):
		customType, err := f.makeCustomType(str, ptr)
		if err != nil {
			return err
		}

		f.field.Set(customType)
	case IsImplementingUnmarshaler(f.metadata.Type):
		withUnmarshaller, err := f.makeWithUnmarshaller(str)
		if err != nil {
			return err
		}

		f.field.Set(withUnmarshaller)
	case IsNative(f.metadata.Type):
		native, err := f.makeNative(str, ptr)
		if err != nil {
			return err
		}

		f.field.Set(native)
	default:
		return &FieldSetterError{
			Message:            "type is not native, unmarshaller or custom type",
			FieldSetterContext: f.errorContext,
		}
	}

	return nil
}

func (f fieldSetter) makeNative(str string, ptr bool) (reflect.Value, *FieldSetterError) {
	el := reflect.New(f.metadata.Type)

	switch f.metadata.Type.Kind() {
	case reflect.String:
		el.Elem().SetString(str)

		if ptr {
			return el, nil
		}
		return el.Elem(), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, err := strconv.ParseInt(str, 10, f.metadata.Type.Bits())
		if err != nil {
			return reflect.Value{}, &FieldSetterError{
				Message:            "invalid integer",
				FieldSetterContext: f.errorContext,
			}
		}

		el.Elem().SetInt(i)

		if ptr {
			return el, nil
		}
		return el.Elem(), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		i, err := strconv.ParseUint(str, 10, f.metadata.Type.Bits())
		if err != nil {
			return reflect.Value{}, &FieldSetterError{
				Message:            "invalid positive integer",
				FieldSetterContext: f.errorContext,
			}
		}

		el.Elem().SetUint(i)

		if ptr {
			return el, nil
		}
		return el.Elem(), nil
	case reflect.Bool:
		i, err := strconv.ParseBool(str)
		if err != nil {
			return reflect.Value{}, &FieldSetterError{
				Message:            "invalid boolean",
				FieldSetterContext: f.errorContext,
			}
		}

		el.Elem().SetBool(i)

		if ptr {
			return el, nil
		}
		return el.Elem(), nil
	case reflect.Float32, reflect.Float64:
		i, err := strconv.ParseFloat(str, f.metadata.Type.Bits())
		if err != nil {
			return reflect.Value{}, &FieldSetterError{
				Message:            "invalid floating number",
				FieldSetterContext: f.errorContext,
			}
		}

		el.Elem().SetFloat(i)

		if ptr {
			return el, nil
		}
		return el.Elem(), nil
	}

	return reflect.Value{}, &FieldSetterError{
		Message:            "invalid native type",
		FieldSetterContext: f.errorContext,
	}
}

func (f fieldSetter) makeWithUnmarshaller(str string) (reflect.Value, *FieldSetterError) {
	var el reflect.Value
	if f.metadata.Type.Kind() == reflect.Ptr {
		el = reflect.New(f.metadata.Type.Elem())
	} else {
		el = reflect.New(f.metadata.Type)
	}

	if el.Type().Implements(TextUnmarshaller) {
		t := el.Interface().(encoding.TextUnmarshaler)

		err := t.UnmarshalText([]byte(str))
		if err != nil {
			return reflect.Value{}, &FieldSetterError{
				Message:            "unable to unmarshal from text format",
				FieldSetterContext: f.errorContext,
			}
		}

		return el, nil
	}

	if el.Type().Implements(JSONUnmarshaler) {
		t := el.Interface().(json.Unmarshaler)
		err := t.UnmarshalJSON([]byte(str))
		if err != nil {
			return reflect.Value{}, &FieldSetterError{
				Message:            "unable to unmarshal from json format",
				FieldSetterContext: f.errorContext,
			}
		}

		return el, nil
	}

	if el.Type().Implements(BinaryUnmarshaler) {
		t := el.Interface().(encoding.BinaryUnmarshaler)
		err := t.UnmarshalBinary([]byte(str))
		if err != nil {
			return reflect.Value{}, &FieldSetterError{
				Message:            "unable to unmarshal from binary format",
				FieldSetterContext: f.errorContext,
			}
		}

		return el, nil
	}

	return reflect.Value{}, &FieldSetterError{
		Message:            "unable to unmarshal",
		FieldSetterContext: f.errorContext,
	}
}

func (f fieldSetter) makeCustomType(str string, ptr bool) (reflect.Value, *FieldSetterError) {
	el := reflect.New(f.metadata.Type)

	if f.metadata.Type.ConvertibleTo(reflect.TypeOf(time.Duration(0))) {
		duration, err := time.ParseDuration(str)
		if err != nil {
			return reflect.Value{}, &FieldSetterError{
				Message:            "unable to parse duration (format: 1ms, 1s, 3h3s)",
				FieldSetterContext: f.errorContext,
			}
		}

		el.Elem().Set(reflect.ValueOf(duration))

		if ptr {
			return el, nil
		}
		return el.Elem(), nil
	}

	return reflect.Value{}, &FieldSetterError{
		Message:            "invalid custom type",
		FieldSetterContext: f.errorContext,
	}
}
