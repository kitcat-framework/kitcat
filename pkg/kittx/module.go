package kittx

import (
	"context"
	"gorm.io/gorm"
)

type Module struct {
	db *gorm.DB
}

func New(db *gorm.DB) *Module {
	return &Module{
		db: db,
	}
}

func (m Module) Name() string {
	return "kittx"
}

type ctxKey string

const (
	ctxKeyTx ctxKey = "kitttx"
)

func Tx[R any](m *Module, function func(ctx context.Context) (R, bool)) R {
	ret := new(R)

	_ = m.db.Transaction(func(tx *gorm.DB) error {
		ctx := context.WithValue(context.Background(), ctxKeyTx, tx)
		r, b := function(ctx)

		ret = &r

		if !b {
			return gorm.ErrInvalidTransaction
		} else {
			return nil
		}
	})

	return *ret
}

func MayTx(ctx context.Context, db *gorm.DB) *gorm.DB {
	if tx, ok := ctx.Value(ctxKeyTx).(*gorm.DB); ok {
		return tx
	}

	return db
}
