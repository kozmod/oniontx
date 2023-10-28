package oniontx

import (
	"context"
	"errors"

	"golang.org/x/xerrors"
)

var (
	ErrNilBeginner     = xerrors.New("tx beginner is nil")
	ErrBeginTx         = xerrors.New("begin tx")
	ErrCommitFailed    = xerrors.New("commit failed")
	ErrRollbackFailed  = xerrors.New("rollback failed")
	ErrRollbackSuccess = xerrors.New("rollback tx")
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
	var b B
	if b != beginner {
		b = beginner
	}
	return &Transactor[B, T, O]{
		beginner: b,
		operator: operator,
	}
}

// WithinTx execute all queries with TxCommitter.
// The function create new TxCommitter or reuse TxCommitter obtained from context.Context.
func (t *Transactor[B, T, O]) WithinTx(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	return t.WithinTxWithOpts(ctx, fn)
}

// WithinTxWithOpts execute all queries with TxCommitter and transaction Options.
// The function create new TxCommitter or reuse TxCommitter obtained from context.Context.
func (t *Transactor[B, T, O]) WithinTxWithOpts(ctx context.Context, fn func(ctx context.Context) error, opts ...Option[O]) (err error) {
	var nilDB B
	if t.beginner == nilDB {
		return xerrors.Errorf("transactor - cannot begin: %w", ErrNilBeginner)
	}

	tx, ok := t.operator.Extract(ctx)
	if !ok {
		tx, err = t.beginner.BeginTx(ctx, opts...)
		if err != nil {
			return xerrors.Errorf("transactor - cannot begin: %w", errors.Join(err, ErrBeginTx))
		}
	}

	defer func() {
		switch p := recover(); {
		case p != nil:
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				err = xerrors.Errorf("transactor - panic [%v]: %w", p, errors.Join(rbErr, ErrRollbackFailed))
				return
			}
			err = xerrors.Errorf("transactor - panic [%v]: %w", p, ErrRollbackSuccess)
		case err != nil:
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				err = xerrors.Errorf("transactor: %w", errors.Join(err, rbErr, ErrRollbackFailed))
				return
			}
			err = xerrors.Errorf("transactor: %w", errors.Join(err, ErrRollbackSuccess))
		default:
			if err = tx.Commit(ctx); err != nil {
				err = xerrors.Errorf("transactor: %w", errors.Join(err, ErrCommitFailed))
			}
		}
	}()

	ctx = t.operator.Inject(ctx, tx)
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
