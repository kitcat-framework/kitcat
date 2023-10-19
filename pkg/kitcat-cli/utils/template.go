package utils

import (
	"bytes"
	"text/template"
)

func Template(str string, data any) (*bytes.Buffer, error) {
	tmpl := template.New("template")

	tmpl, err := tmpl.Parse(str)
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	if err := tmpl.Execute(buf, data); err != nil {
		return nil, err
	}

	return buf, nil
}
