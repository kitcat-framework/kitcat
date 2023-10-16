package httpbind

import (
	"net/http"
)

// QueryExtractor allows to obtain a value from the query params of the request.
type QueryExtractor struct{}

// Extract query params from the http request.
func (q QueryExtractor) Extract(req *http.Request, valueOfTag string) ([]string, error) {
	return req.URL.Query()[valueOfTag], nil
}

// Tag return the tag name of this extractor.
func (q QueryExtractor) Tag() string {
	return "query"
}
