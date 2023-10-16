package httpbind

import (
	"encoding"
	"encoding/json"
	"reflect"
)

// Unmarshalers is the list of possible unmarshaler kcd support.
var Unmarshalers = []reflect.Type{
	reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem(),
	reflect.TypeOf((*encoding.BinaryUnmarshaler)(nil)).Elem(),
	reflect.TypeOf((*json.Unmarshaler)(nil)).Elem(),
}

var (
	// TextUnmarshaller is the type of encoding.TextUnmarshaler
	TextUnmarshaller = Unmarshalers[0]

	// BinaryUnmarshaler is the type of encoding.BinaryUnmarshaler
	BinaryUnmarshaler = Unmarshalers[1]

	// JSONUnmarshaler is the type of json.Unmarshaler
	JSONUnmarshaler = Unmarshalers[2]
)
