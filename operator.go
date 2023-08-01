package oniontx

import (
	"context"
	"fmt"
)

type CtxKeyType string

// ContextOperator inject and extract TxCommitter rom context.Context.
type ContextOperator[B TxBeginner[C, O], C TxCommitter, O any] struct {
	beginner *B
}

// NewContextOperator creates new ContextOperator.
func NewContextOperator[B TxBeginner[C, O], C TxCommitter, O any](b *B) *ContextOperator[B, C, O] {
	return &ContextOperator[B, C, O]{
		beginner: b,
	}
}

// Inject injects TxCommitter.
func (p *ContextOperator[B, C, O]) Inject(ctx context.Context, c C) context.Context {
	key := p.Key()
	return context.WithValue(ctx, key, c)
}

// Extract injects TxCommitter.
func (p *ContextOperator[B, C, O]) Extract(ctx context.Context) (C, bool) {
	key := p.Key()
	c, ok := ctx.Value(key).(C)
	return c, ok
}

// Key get key (CtxKeyType) for injecting or extracting TxCommitter from context.Context.
func (p *ContextOperator[B, C, O]) Key() CtxKeyType {
	return CtxKeyType(fmt.Sprintf("%p", p.beginner))
}
