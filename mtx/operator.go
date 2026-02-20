package mtx

import (
	"context"
)

// ContextOperator implements transaction injection and extraction from context.Context.
// It uses a type-safe approach with comparable keys to store and retrieve transactions
// from context, avoiding global string keys and providing compile-time type safety.
//
// The type parameters:
//   - K: a comparable type used as the context key (typically a custom type or iota)
//   - T: the transaction type that satisfies the Tx interface
type ContextOperator[K comparable, T Tx] struct {
	key K
}

// NewContextOperator returns a pointer to a new ContextOperator instance.
//
// It accepts a key of a comparable type that will be used for both injecting
// and extracting transactions from context.Context.
//
// The key should be unique to avoid collisions.
func NewContextOperator[K comparable, T Tx](key K) *ContextOperator[K, T] {
	return &ContextOperator[K, T]{
		key: key,
	}
}

// Inject stores a transaction in the context and returns a new context containing it.
// It uses context.WithValue internally, associating the transaction with the operator's key.
// The original context remains unchanged.
//
// This method is typically used by Transactor when creating a new transaction
// to make it available to nested function calls.
//
// Parameters:
//   - ctx: the parent context
//   - tx: the transaction to store
func (o *ContextOperator[K, T]) Inject(ctx context.Context, tx T) context.Context {
	return context.WithValue(ctx, o.key, tx)
}

// Extract retrieves a transaction from the context if one exists.
// It performs a type assertion to convert the context value to the expected transaction type.
// The boolean return value indicates whether a transaction was found and successfully
// type-asserted.
//
// This method is used by Transactor to check if a transaction already exists
// in the context, enabling nested transaction support.
//
// Parameters:
//   - ctx: the context to extract the transaction from
//
// Returns:
//   - T: the extracted transaction (zero value if not found)
//   - bool: true if a transaction was found and is of the correct type
//
// Example:
//
//	tx, ok := operator.Extract(ctx)
//	if ok {
//	    // Use existing transaction
//	    result := tx.QueryRow(...)
//	} else {
//	    // No transaction in context, create new one
//	}
func (o *ContextOperator[K, T]) Extract(ctx context.Context) (T, bool) {
	c, ok := ctx.Value(o.key).(T)
	return c, ok
}
