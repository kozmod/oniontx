package mtx

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
	ErrPanicRecovered  = fmt.Errorf("panic recovered")
)

type (
	// TxBeginner is responsible for creating new Tx.
	TxBeginner[T Tx] interface {
		comparable
		BeginTx(ctx context.Context) (T, error)
	}

	// Tx represent transaction contract.
	Tx interface {
		Rollback(ctx context.Context) error
		Commit(ctx context.Context) error
	}

	// СtxOperator is responsible for interaction with context.Context to store or extract Tx.
	СtxOperator[T Tx] interface {
		Inject(ctx context.Context, tx T) context.Context
		Extract(ctx context.Context) (T, bool)
	}
)

// Transactor manage a transaction for single TxBeginner instance.
type Transactor[B TxBeginner[T], T Tx] struct {
	beginner B
	operator СtxOperator[T]
}

// NewTransactor returns new Transactor.
func NewTransactor[B TxBeginner[T], T Tx](
	beginner B,
	operator СtxOperator[T]) *Transactor[B, T] {
	return &Transactor[B, T]{
		beginner: beginner,
		operator: operator,
	}
}

// WithinTx execute all queries with Tx and transaction Options.
// The function create new Tx or reuse Tx obtained from [context.Context].
/*
When WithinTx call recursively, the highest level function responsible for creating transaction and applying commit or rollback of a transaction.

		tr := NewTransactor[...](...)

		// The highest level.
		// A transaction creates and finishes (commit/rollback) on this level.
		err := tr.WithinTx(ctx, func(ctx context.Context) error {

			// A middle level.
			err := tr.WithinTx(ctx, func(ctx context.Context) error {
				return nil
			})

			// A middle level.
			err = tr.WithinTx(ctx, func(ctx context.Context) error {

				// The lowest level.
				err = tr.WithinTx(ctx, func(ctx context.Context) error {
					return nil
				})
				return nil
			})

			return err
		})

NOTE:

+ a processed error returns to the highest level function for commit or rollback.

+ panics are transformed to errors with the same message.

+ higher level panics override lower level panic (that was transformed to an error) or an error.

Examples:

1 - [mtx.Test_Transactor_recursive_call]
2 - [test/integration/internal/stdlib/stdlib_test.go]
*/
func (t *Transactor[B, T]) WithinTx(ctx context.Context, fn func(ctx context.Context) error) (err error) {
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
		tx, err = t.beginner.BeginTx(ctx)
		if err != nil {
			return fmt.Errorf("transactor - cannot begin: %w", errors.Join(err, ErrBeginTx))
		}
	}

	defer func() {
		switch p := recover(); {
		case p != nil:
			if ok {
				err = errors.Join(fmt.Errorf("transactor - panic [%v]", p), ErrPanicRecovered)
				return
			}
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				err = fmt.Errorf(
					"transactor - panic [%v]: %w", p,
					errors.Join(rbErr, ErrRollbackFailed, ErrPanicRecovered),
				)
			} else {
				err = fmt.Errorf(
					"transactor - panic [%v]: %w", p,
					errors.Join(ErrRollbackSuccess, ErrPanicRecovered),
				)
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

	err = fn(ctx)
	return err
}

// TryGetTx returns [Tx] and "true" from [context.Context] or return `false`.
func (t *Transactor[B, T]) TryGetTx(ctx context.Context) (T, bool) {
	tx, ok := t.operator.Extract(ctx)
	return tx, ok
}

// TxBeginner returns [TxBeginner] which using in Transactor.
func (t *Transactor[B, T]) TxBeginner() B {
	return t.beginner
}
