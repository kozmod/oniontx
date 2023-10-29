package oniontx

import (
	"context"
	"fmt"
)

// CtxKey is key type for default ContextOperator.
type CtxKey string

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
	key := p.Key()
	return context.WithValue(ctx, key, tx)
}

// Extract returns Tx extracted from context.Context.
func (p *ContextOperator[B, T]) Extract(ctx context.Context) (T, bool) {
	key := p.Key()
	c, ok := ctx.Value(key).(T)
	return c, ok
}

// Key returns key (CtxKey) for injecting or extracting Tx from context.Context.
func (p *ContextOperator[B, T]) Key() CtxKey {
	return CtxKey(fmt.Sprintf("%p", p.beginner))
}
