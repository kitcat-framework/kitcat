package kitweb

import (
	"context"
	"net/http"
)

type Res interface {
	Write(ctx context.Context, w http.ResponseWriter) error
}
