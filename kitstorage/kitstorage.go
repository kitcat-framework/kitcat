package kitstorage

import (
	"context"
	"github.com/kitcat-framework/kitcat"
	"github.com/kitcat-framework/kitcat/kitdi"
	"go.uber.org/dig"
	"io"
	"time"
)

type (
	FileSystem interface {
		Put(ctx context.Context, path string, reader io.Reader, opts ...PutOptionFunc) error
		Get(ctx context.Context, path string) (io.Reader, error)
		Exists(ctx context.Context, path string) (bool, error)
		Delete(ctx context.Context, path string) error
		GetURL(ctx context.Context, path string, opts ...GetURLOptionFunc) (string, error)
		ListFiles(ctx context.Context, path string, recursive bool) ([]string, error)

		kitcat.Nameable
	}

	PutOptions struct {
		options map[string]any

		Public bool
	}

	PutOptionFunc func(*PutOptions)

	GetURLOptionFunc func(*GetURLOptions)

	GetURLOptions struct {
		PreSign    bool
		Expiration *time.Duration
	}

	fileSystems struct {
		dig.In

		FileSystems []FileSystem `group:"kitstorage.filesystem"`
	}
)

func ProvideFileSystem(a any) *kitdi.Annotation {
	return kitdi.Annotate(a, kitdi.As(new(FileSystem)), kitdi.Group("kitstorage.filesystem"))
}

func NewPutOptions() *PutOptions {
	return &PutOptions{
		options: make(map[string]any),
	}
}

func NewGetURLOptions() *GetURLOptions {
	return &GetURLOptions{
		PreSign: false,
	}
}

var PutOptionPublic = func(o *PutOptions) {
	o.Public = true
}

var GetURLOptionPreSign = func(o *GetURLOptions) {
	o.PreSign = true
}

func GetURLOptionExpiration(d time.Duration) GetURLOptionFunc {
	return func(o *GetURLOptions) {
		o.Expiration = &d
	}
}
