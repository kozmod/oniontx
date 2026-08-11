// Package mtx provides a flexible transaction management system with support for
// nested transactions, panic recovery, and context-based transaction propagation.
// It allows working with different database/sql implementations through interfaces
package mtx

import (
	"context"
	"fmt"

	"github.com/kozmod/oniontx/internal/errors"
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

	// CtxOperator is responsible for transaction propagation through context.Context.
	// It provides methods to inject a transaction into context and extract it back.
	CtxOperator[T Tx] interface {
		Inject(ctx context.Context, tx T) context.Context
		Extract(ctx context.Context) (T, bool)
	}
)

// Transactor manages transactions for a single TxBeginner instance.
// It provides a high-level API for executing functions within a transaction context,
// with support for nested transactions, automatic rollback on error/panic,
// and proper transaction propagation through context.
//
// The type parameters B and T allow working with any transaction implementation
// that satisfies the TxBeginner and Tx interfaces respectively.
type Transactor[B TxBeginner[T], T Tx] struct {
	beginner           B
	operator           CtxOperator[T]
	rollbackCtxFactory func(ctx context.Context) context.Context
}

// NewTransactor returns new Transactor.
func NewTransactor[B TxBeginner[T], T Tx](
	beginner B,
	operator CtxOperator[T]) *Transactor[B, T] {
	return &Transactor[B, T]{
		beginner: beginner,
		operator: operator,
		rollbackCtxFactory: func(ctx context.Context) context.Context {
			return ctx
		},
	}
}

// WithRollbackCtxFactory returns a new Transactor that derives the context used
// for top-level rollback operations. The original Transactor is not modified.
//
// This is useful when the operation context may be canceled before rollback.
// For example, context.WithoutCancel allows a rollback to outlive request
// cancellation. If factory is nil or returns nil, rollback uses the original
// operation context.
func (t *Transactor[B, T]) WithRollbackCtxFactory(factory func(ctx context.Context) context.Context) *Transactor[B, T] {
	return &Transactor[B, T]{
		beginner:           t.beginner,
		operator:           t.operator,
		rollbackCtxFactory: factory,
	}
}

// WithinTx executes the provided function within a transaction context.
// It handles transaction creation, propagation, and automatic cleanup (commit/rollback).
//
// Key features:
//   - Nested transaction support: When called recursively, only the top-level
//     call creates and manages the actual transaction. Inner calls reuse the existing
//     transaction from the context.
//   - Automatic rollback: If the function returns an error or panics, the
//     transaction is automatically rolled back.
//   - Automatic commit: If the function completes without error, the transaction
//     is automatically committed (only at the top level).
//   - Panic recovery: Panics are recovered and converted to errors with
//     ErrPanicRecovered. Higher-level panics override lower-level ones.
//   - Context propagation: The transaction is injected into the context for
//     inner function calls.
//
// The function follows these rules:
//   - If a transaction exists in the context, it is reused (nested call)
//   - Otherwise, a new transaction is created (top-level call)
//   - Errors from the function or from commit/rollback are properly wrapped
//   - Panics are handled gracefully without crashing the application
//
// Example:
//
//	// Top-level transaction
//	err := transactor.WithinTx(ctx, func(ctx context.Context) error {
//	    // This operation runs in a transaction
//	    if err := someOperation(ctx); err != nil {
//	        return err // Will trigger rollback
//	    }
//
//	    // Nested call - reuses the same transaction
//	    err := transactor.WithinTx(ctx, func(ctx context.Context) error {
//	        return anotherOperation(ctx) // Same transaction
//	    })
//
//	    return err
//	}) // Auto-commits on success, rolls back on error
//
// Note:
//   - A processed error returns to the highest level for commit or rollback
//   - Panics are transformed to errors with the same message
//   - Higher level panics override lower level panics or errors
//
// Examples:
//   - [mtx.Test_Transactor_recursive_call]
//   - [test/integration/internal/stdlib/stdlib_test.go]
func (t *Transactor[B, T]) WithinTx(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	var (
		nilBeginner B
		nilOperator CtxOperator[T] = nil
	)

	if t == nil {
		return fmt.Errorf("transactor is nil")
	}

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
			return fmt.Errorf("transactor - cannot begin: %w", errors.Join(ErrBeginTx, err))
		}
	}

	defer func() {
		switch p := recover(); {
		case p != nil:
			if ok {
				err = fmt.Errorf(
					"transactor - panic: %w",
					errors.Join(ErrPanicRecovered, errors.WrapPanic(p)),
				)
				return
			}

			rollbackCtx := ctx
			if t.rollbackCtxFactory != nil {
				if newRollbackCtx := t.rollbackCtxFactory(ctx); newRollbackCtx != nil {
					rollbackCtx = newRollbackCtx
				}
			}

			if rbErr := tx.Rollback(rollbackCtx); rbErr != nil {
				err = fmt.Errorf(
					"transactor - panic: %w",
					errors.Join(ErrPanicRecovered, NewRollbackError(errors.WrapPanic(p), rbErr)),
				)
			} else {
				err = fmt.Errorf(
					"transactor - panic: %w",
					errors.Join(ErrPanicRecovered, errors.WrapPanic(p)),
				)
			}
		case err != nil:
			if ok {
				return
			}

			rollbackCtx := ctx
			if t.rollbackCtxFactory != nil {
				if newRollbackCtx := t.rollbackCtxFactory(ctx); newRollbackCtx != nil {
					rollbackCtx = newRollbackCtx
				}
			}

			if rbErr := tx.Rollback(rollbackCtx); rbErr != nil {
				err = fmt.Errorf("transactor - call: %w", NewRollbackError(err, rbErr))
			} else {
				err = fmt.Errorf("transactor - call: %w", err)
			}
		default:
			if ok {
				return
			}
			if err = tx.Commit(ctx); err != nil {
				err = fmt.Errorf("transactor: %w", errors.Join(ErrCommitFailed, err))
			}
		}
	}()

	if !ok {
		ctx = t.operator.Inject(ctx, tx)
	}

	err = fn(ctx)
	return err
}

// TryGetTx attempts to retrieve a transaction from the given context.
// It returns the transaction and true if found, or a zero value and false otherwise.
func (t *Transactor[B, T]) TryGetTx(ctx context.Context) (T, bool) {
	tx, ok := t.operator.Extract(ctx)
	return tx, ok
}

// TxBeginner returns the underlying TxBeginner used by this Transactor.
// This can be useful for creating transactions manually.
func (t *Transactor[B, T]) TxBeginner() B {
	return t.beginner
}
