package kitstorage

import (
	"context"
	"github.com/expectedsh/dig"
	"github.com/expectedsh/kitcat"
	"github.com/expectedsh/kitcat/kitdi"
	"io"
)

type (
	FileSystem interface {
		Put(ctx context.Context, path string, reader io.Reader, opts ...PutOptionFunc) error
		Get(ctx context.Context, path string) (io.Reader, error)
		Exists(ctx context.Context, path string) (bool, error)
		Delete(ctx context.Context, path string) error
		GetURL(ctx context.Context, path string) (string, error)
		ListFiles(ctx context.Context, path string, recursive bool) ([]string, error)

		kitcat.Nameable
	}

	PutOptions struct {
		options map[string]any

		public bool
	}

	PutOptionFunc func(*PutOptions)

	GetURLOptions struct {
		options map[string]any
	}

	GetURLOptionFunc func(*GetURLOptions)

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

var PutOptionPublic = func(o *PutOptions) {
	o.public = true
}
