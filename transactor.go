package oniontx

import (
	"context"
	"errors"

	"golang.org/x/xerrors"
)

var (
	ErrNilBeginner     = xerrors.New("tx beginner is nil")
	ErrCommitFailed    = xerrors.New("commit failed")
	ErrRollbackFailed  = xerrors.New("rollback failed")
	ErrRollbackSuccess = xerrors.New("rollback tx")
)

type (
	TxBeginner[C TxCommitter, O any] interface {
		comparable
		BeginTx(ctx context.Context, opts ...Option[O]) (C, error)
	}

	TxCommitter interface {
		Rollback(ctx context.Context) error
		Commit(ctx context.Context) error
	}

	Option[TxOpt any] interface {
		Apply(in TxOpt)
	}

	СtxOperator[C TxCommitter] interface {
		Inject(ctx context.Context, c C) context.Context
		Extract(ctx context.Context) (C, bool)
	}
)

// Transactor manage a transaction for single TxBeginner instance.
type Transactor[B TxBeginner[C, O], C TxCommitter, O any] struct {
	beginner B
	operator СtxOperator[C]
}

// NewTransactor creates new Transactor.
func NewTransactor[B TxBeginner[C, O], C TxCommitter, O any](
	beginner B,
	operator СtxOperator[C]) *Transactor[B, C, O] {
	var base B
	if base != beginner {
		base = beginner
	}
	return &Transactor[B, C, O]{
		beginner: base,
		operator: operator,
	}
}

// WithinTx execute all queries with TxCommitter.
// The function create new TxCommitter or reuse TxCommitter obtained from context.Context.
func (t *Transactor[B, C, O]) WithinTx(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	return t.WithinTxWithOpts(ctx, fn)
}

// WithinTxWithOpts execute all queries with TxCommitter and transaction Options.
// The function create new TxCommitter or reuse TxCommitter obtained from context.Context.
func (t *Transactor[B, C, O]) WithinTxWithOpts(ctx context.Context, fn func(ctx context.Context) error, opts ...Option[O]) (err error) {
	var nilDB B
	if t.beginner == nilDB {
		return xerrors.Errorf("transactor: cannot begin: %w", ErrNilBeginner)
	}

	tx, ok := t.operator.Extract(ctx)
	if !ok {
		tx, err = t.beginner.BeginTx(ctx, opts...)
		if err != nil {
			return xerrors.Errorf("transactor: cannot begin: %w", err)
		}
	}

	defer func() {
		switch p := recover(); {
		case p != nil:
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				err = xerrors.Errorf("transactor: panic: %v: %w", p, errors.Join(rbErr, ErrRollbackFailed))
				return
			}
			err = xerrors.Errorf("transactor: panic: %v: %w", p, ErrRollbackSuccess)
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

// TryGetTx returns pointer of TxBeginner from context.Context or return `false`.
func (t *Transactor[B, C, O]) TryGetTx(ctx context.Context) (C, bool) {
	tx, ok := t.operator.Extract(ctx)
	return tx, ok
}

// TxBeginner returns pointer of TxBeginner which using in Transactor.
func (t *Transactor[B, C, O]) TxBeginner() B {
	return t.beginner
}
