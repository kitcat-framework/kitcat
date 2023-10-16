package kitslog

import "log/slog"

func Err(err error) slog.Attr {
	return slog.String("err", err.Error())
}

func Module(mod string) slog.Attr {
	return slog.String("module", mod)
}
