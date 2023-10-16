package httpbind

import (
	"net/http"
)

// ContextExtractor extract value from the the context of the request.
type ContextExtractor struct{}

// Extract value from the context of the request.
func (c ContextExtractor) Extract(req *http.Request, valueOfTag string) (interface{}, error) {
	return req.Context().Value(valueOfTag), nil
}

// Tag return the tag name of this extractor.
func (c ContextExtractor) Tag() string {
	return "ctx"
}
