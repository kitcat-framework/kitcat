package httpbind

import (
	"net/http"
)

// FileExtractor extract file from the request.
type FileExtractor struct{}

// Extract value from the request.
func (p FileExtractor) Extract(req *http.Request, valueOfTag string) (any, error) {
	//req.ParseMultipartForm(32 << 20) // limit your max input length! -> config that ?
	file, header, err := req.FormFile(valueOfTag)
	if err != nil {
		panic(err)
	}

	return &File{
		Header: header,
		File:   file,
	}, nil
}

// Tag return the tag name of this extractor.
func (p FileExtractor) Tag() string {
	return "file"
}
