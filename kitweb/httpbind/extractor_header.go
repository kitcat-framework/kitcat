package httpbind

import "net/http"

// HeaderExtractor allows to obtain a value from the header of the request.
type HeaderExtractor struct{}

// Extract header from the http request.
func (h HeaderExtractor) Extract(req *http.Request, valueOfTag string) ([]string, error) {
	header := req.Header.Values(valueOfTag)

	if header == nil {
		return nil, nil
	}

	return header, nil
}

// Tag return the tag name of this extractor.
func (h HeaderExtractor) Tag() string {
	return "header"
}
