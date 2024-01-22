package oniontx

import (
	"context"
)

// ContextOperator inject and extract Tx from context.Context.
//
// Default ContextOperator uses comparable key for context.Context value operation.
type ContextOperator[K comparable, T Tx] struct {
	key K
}

// NewContextOperator returns new ContextOperator.
func NewContextOperator[K comparable, T Tx](key K) *ContextOperator[K, T] {
	return &ContextOperator[K, T]{
		key: key,
	}
}

// Inject returns new context.Context contains Tx as value.
func (p *ContextOperator[K, T]) Inject(ctx context.Context, tx T) context.Context {
	return context.WithValue(ctx, p.key, tx)
}

// Extract returns Tx extracted from context.Context.
func (p *ContextOperator[K, T]) Extract(ctx context.Context) (T, bool) {
	c, ok := ctx.Value(p.key).(T)
	return c, ok
}
