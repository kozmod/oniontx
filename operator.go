package oniontx

import (
	"context"
	"fmt"
)

// CtxKey is key type for default ContextOperator.
type CtxKey string

// ContextOperator inject and extract TxCommitter from context.Context.
type ContextOperator[B any, C TxCommitter] struct {
	beginner *B
}

// NewContextOperator returns new ContextOperator.
func NewContextOperator[B any, C TxCommitter](b *B) *ContextOperator[B, C] {
	return &ContextOperator[B, C]{
		beginner: b,
	}
}

// Inject returns new context.Context contains TxCommitter as value.
func (p *ContextOperator[B, C]) Inject(ctx context.Context, c C) context.Context {
	key := p.Key()
	return context.WithValue(ctx, key, c)
}

// Extract returns TxCommitter extracted from context.Context.
func (p *ContextOperator[B, C]) Extract(ctx context.Context) (C, bool) {
	key := p.Key()
	c, ok := ctx.Value(key).(C)
	return c, ok
}

// Key returns key (CtxKey) for injecting or extracting TxCommitter from context.Context.
func (p *ContextOperator[B, C]) Key() CtxKey {
	return CtxKey(fmt.Sprintf("%p", p.beginner))
}
