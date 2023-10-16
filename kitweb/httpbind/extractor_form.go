package httpbind

import (
	"net/http"
)

// FormExtractor extract value from the chi router.
type FormExtractor struct{}

// Extract value from the chi router.
func (p FormExtractor) Extract(req *http.Request, valueOfTag string) ([]string, error) {
	str := req.FormValue(valueOfTag)
	if str == "" {
		return nil, nil
	}

	return []string{str}, nil
}

// Tag return the tag name of this extractor.
func (p FormExtractor) Tag() string {
	return "form"
}
