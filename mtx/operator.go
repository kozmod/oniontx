package mtx

import (
	"context"
)

// ContextOperator inject and extract Tx from [context.Context].
//
// Default ContextOperator uses comparable key for [context.Context] value operation.
type ContextOperator[K comparable, T Tx] struct {
	key K
}

// NewContextOperator returns new ContextOperator.
//
// `key` uses as argument for extracting/injecting Tx.
func NewContextOperator[K comparable, T Tx](key K) *ContextOperator[K, T] {
	return &ContextOperator[K, T]{
		key: key,
	}
}

// Inject returns new [context.Context] contains Tx as value.
//
// Function wraps [context.WithValue].
func (o *ContextOperator[K, T]) Inject(ctx context.Context, tx T) context.Context {
	return context.WithValue(ctx, o.key, tx)
}

// Extract returns Tx extracted from [context.Context].
//
// Function calls `Value` with `key` as an argument, injected into ContextOperator.
func (o *ContextOperator[K, T]) Extract(ctx context.Context) (T, bool) {
	c, ok := ctx.Value(o.key).(T)
	return c, ok
}
