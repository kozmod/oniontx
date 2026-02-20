// Package mtx provides a flexible transaction management system with support for
// nested transactions, panic recovery, and context-based transaction propagation.
// It allows working with different database/sql implementations through interfaces
package mtx

import (
	"context"
	"errors"
	"fmt"
)

var (
	// ErrNilTxBeginner indicates that the provided TxBeginner is nil.
	// This error is returned when trying to use a Transactor with an uninitialized beginner.
	ErrNilTxBeginner = fmt.Errorf("tx beginner is nil")

	// ErrNilTxOperator indicates that the provided CtxOperator is nil.
	// This error is returned when trying to use a Transactor with an uninitialized operator.
	ErrNilTxOperator = fmt.Errorf("tx operator is nil")

	// ErrBeginTx indicates that starting a new transaction has failed.
	// This error wraps the underlying error from the database driver.
	ErrBeginTx = fmt.Errorf("begin tx")

	// ErrCommitFailed indicates that committing a transaction has failed.
	// This error wraps the underlying error from the database driver during commit
	ErrCommitFailed = fmt.Errorf("commit failed")

	// ErrRollbackFailed indicates that rolling back a transaction has failed.
	// This error wraps the underlying error from the database driver during rollback.
	ErrRollbackFailed = fmt.Errorf("rollback failed")

	// ErrRollbackSuccess indicates that a transaction was successfully rolled back.
	// Despite being an error type, it signals a successful rollback operation.
	ErrRollbackSuccess = fmt.Errorf("rollback tx")

	// ErrPanicRecovered indicates that a panic was recovered and converted to an error.
	// It wraps the original panic value to provide context about what caused the panic.
	ErrPanicRecovered = fmt.Errorf("panic recovered")
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

	// СtxOperator is responsible for transaction propagation through context.Context.
	// It provides methods to inject a transaction into context and extract it back.
	СtxOperator[T Tx] interface {
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
