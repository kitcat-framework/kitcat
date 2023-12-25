package httpbind

import (
	"mime/multipart"
	"reflect"
	"time"
)

type File struct {
	multipart.File
	Header *multipart.FileHeader
}

var d = time.Duration(1)

// Custom is the list of custom types supported.
var Custom = []reflect.Type{
	reflect.TypeOf(time.Duration(1)),
	reflect.TypeOf(&d),

	reflect.TypeOf(&File{}),
}
