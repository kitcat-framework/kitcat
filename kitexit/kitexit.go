package kitexit

import (
	"github.com/kitcat-framework/kitcat/kitslog"
	"log/slog"
	"os"
)

func Abnormal(err error) {
	slog.Error("abnormal exit", kitslog.Err(err))
	os.Exit(1)
}
