package mockery

//go:generate mockery --inpackage --all  --outpkg=mockery --dir=. --outpkg=profile --output=.
//go:generate sh ./scripts.sh update_mocks .
//go:generate git add .

import (
	"context"
	"database/sql"
	"fmt"
)

type (
	repository interface {
		Insert(ctx context.Context, val string) error
	}

	transactor interface {
		WithinTx(ctx context.Context, fn func(ctx context.Context) error) (err error)
		WithinTxWithOpts(ctx context.Context, fn func(ctx context.Context) error, opts ...dbOptSetter) (err error)
	}
)

type UseCase struct {
	textRepoA repository
	textRepoB repository

	transactor transactor
}

func (u *UseCase) CreateTextRecords(ctx context.Context, text string) error {
	return u.transactor.WithinTx(ctx, func(ctx context.Context) error {
		err := u.textRepoA.Insert(ctx, text)
		if err != nil {
			return fmt.Errorf("text repo A: %w", err)
		}

		err = u.textRepoB.Insert(ctx, text)
		if err != nil {
			return fmt.Errorf("text repo B: %w", err)
		}
		return nil
	})
}

type dbOptSetter func(opt *sql.TxOptions)

func (o dbOptSetter) Apply(opt *sql.TxOptions) {
	o(opt)
}

//goland:noinspection GoExportedFuncWithUnexportedType
func WithIsolationLevel(level int) dbOptSetter {
	return func(opt *sql.TxOptions) {
		opt.Isolation = sql.IsolationLevel(level)
	}
}

//goland:noinspection GoExportedFuncWithUnexportedType
func WithReadOnly(readonly bool) dbOptSetter {
	return func(opt *sql.TxOptions) {
		opt.ReadOnly = readonly
	}
}
