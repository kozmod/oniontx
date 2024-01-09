package oniontx

import (
	"context"
)

// ContextOperator inject and extract Tx from context.Context.
type ContextOperator[B any, T Tx] struct {
	beginner *B
}

// NewContextOperator returns new ContextOperator.
func NewContextOperator[B any, T Tx](b *B) *ContextOperator[B, T] {
	return &ContextOperator[B, T]{
		beginner: b,
	}
}

// Inject returns new context.Context contains Tx as value.
func (p *ContextOperator[B, T]) Inject(ctx context.Context, tx T) context.Context {
	return context.WithValue(ctx, p.beginner, tx)
}

// Extract returns Tx extracted from context.Context.
func (p *ContextOperator[B, T]) Extract(ctx context.Context) (T, bool) {
	c, ok := ctx.Value(p.beginner).(T)
	return c, ok
}
