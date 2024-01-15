package oniontx

import (
	"context"
	"errors"
	"fmt"
)

var (
	ErrNilTxBeginner   = fmt.Errorf("tx beginner is nil")
	ErrNilTxOperator   = fmt.Errorf("tx operator is nil")
	ErrBeginTx         = fmt.Errorf("begin tx")
	ErrCommitFailed    = fmt.Errorf("commit failed")
	ErrRollbackFailed  = fmt.Errorf("rollback failed")
	ErrRollbackSuccess = fmt.Errorf("rollback tx")
)

type (
	TxBeginner[T Tx, O any] interface {
		comparable
		BeginTx(ctx context.Context, opts ...Option[O]) (T, error)
	}

	Tx interface {
		Rollback(ctx context.Context) error
		Commit(ctx context.Context) error
	}

	Option[TxOpt any] interface {
		Apply(in TxOpt)
	}

	СtxOperator[T Tx] interface {
		Inject(ctx context.Context, tx T) context.Context
		Extract(ctx context.Context) (T, bool)
	}
)

// Transactor manage a transaction for single TxBeginner instance.
type Transactor[B TxBeginner[T, O], T Tx, O any] struct {
	beginner B
	operator СtxOperator[T]
}

// NewTransactor returns new Transactor.
func NewTransactor[B TxBeginner[T, O], T Tx, O any](
	beginner B,
	operator СtxOperator[T]) *Transactor[B, T, O] {
	return &Transactor[B, T, O]{
		beginner: beginner,
		operator: operator,
	}
}

// WithinTx execute all queries with Tx.
// The function create new Tx or reuse Tx obtained from context.Context.
func (t *Transactor[B, T, O]) WithinTx(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	return t.WithinTxWithOpts(ctx, fn)
}

// WithinTxWithOpts execute all queries with Tx and transaction Options.
// The function create new Tx or reuse Tx obtained from context.Context.
func (t *Transactor[B, T, O]) WithinTxWithOpts(ctx context.Context, fn func(ctx context.Context) error, opts ...Option[O]) (err error) {
	var (
		nilBeginner B
		nilOperator СtxOperator[T] = nil
	)
	if t.beginner == nilBeginner {
		return fmt.Errorf("transactor - can't begin: %w", ErrNilTxBeginner)
	}

	if t.operator == nilOperator {
		return fmt.Errorf("transactor - can't try extract transaction: %w", ErrNilTxOperator)
	}

	tx, ok := t.operator.Extract(ctx)
	if !ok {
		tx, err = t.beginner.BeginTx(ctx, opts...)
		if err != nil {
			return fmt.Errorf("transactor - cannot begin: %w", errors.Join(err, ErrBeginTx))
		}
	}

	defer func() {
		switch p := recover(); {
		case p != nil:
			if ok {
				err = fmt.Errorf("transactor - panic [%v]", p)
				return
			}
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				err = fmt.Errorf("transactor - panic [%v]: %w", p, errors.Join(rbErr, ErrRollbackFailed))
			} else {
				err = fmt.Errorf("transactor - panic [%v]: %w", p, ErrRollbackSuccess)
			}
		case err != nil:
			if ok {
				return
			}
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				err = fmt.Errorf("transactor - call: %w", errors.Join(err, rbErr, ErrRollbackFailed))
			} else {
				err = fmt.Errorf("transactor - call: %w", errors.Join(err, ErrRollbackSuccess))
			}
		default:
			if ok {
				return
			}
			if err = tx.Commit(ctx); err != nil {
				err = fmt.Errorf("transactor: %w", errors.Join(err, ErrCommitFailed))
			}
		}
	}()

	if !ok {
		ctx = t.operator.Inject(ctx, tx)
	}
	return fn(ctx)
}

// TryGetTx returns pointer of Tx and "true" from context.Context or return `false`.
func (t *Transactor[B, T, O]) TryGetTx(ctx context.Context) (T, bool) {
	tx, ok := t.operator.Extract(ctx)
	return tx, ok
}

// TxBeginner returns pointer of TxBeginner which using in Transactor.
func (t *Transactor[B, T, O]) TxBeginner() B {
	return t.beginner
}
