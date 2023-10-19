package utils

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/labstack/gommon/color"
)

func Err(err error) error {
	if err == nil {
		return err
	}
	errs := strings.Split(err.Error(), "\n")
	buff := bytes.NewBufferString("")
	errPrefix := color.New().Red("ERR!") + " "
	for i, e := range errs {
		if i != 0 {
			buff.WriteByte('\n')
		}
		buff.WriteString(errPrefix)
		buff.WriteString(e)
	}
	return fmt.Errorf(buff.String())
}
