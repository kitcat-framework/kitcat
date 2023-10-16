package httpbind

import (
	"github.com/gorilla/mux"
	"net/http"
)

// PathExtractor extract value from the chi router.
type PathExtractor struct{}

// Extract value from the chi router.
func (p PathExtractor) Extract(req *http.Request, valueOfTag string) ([]string, error) {
	vars := mux.Vars(req)
	str := vars[valueOfTag]
	if str == "" {
		return nil, nil
	}

	return []string{str}, nil
}

// Tag return the tag name of this extractor.
func (p PathExtractor) Tag() string {
	return "path"
}
