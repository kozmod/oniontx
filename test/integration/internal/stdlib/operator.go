package stdlib

import (
	"context"
	"sync"

	"github.com/kozmod/oniontx/mtx"
)

// ContextOperator supports replacing a transaction in an existing context for
// the savepoint tests. Context values hold a pointer to the transaction interface.
type ContextOperator[K comparable, T mtx.Tx] struct {
	mx  sync.RWMutex
	key K
}

// NewContextOperator creates the mutable context operator used by savepoint tests.
func NewContextOperator[K comparable, T mtx.Tx](key K) *ContextOperator[K, T] {
	return &ContextOperator[K, T]{
		key: key,
	}
}

// Inject stores tx or replaces the existing transaction for the savepoint tests.
// Replacement is visible through contexts sharing the same stored pointer.
// The mutex protects pointer access, not the ordering of transaction operations.
func (o *ContextOperator[K, T]) Inject(ctx context.Context, tx T) context.Context {
	o.mx.Lock()
	defer o.mx.Unlock()
	c, ok := ctx.Value(o.key).(*T)
	if ok && c != nil {
		*c = tx
		return ctx
	}

	return context.WithValue(ctx, o.key, &tx)
}

// Extract reads the current transaction, including wrappers installed by the
// savepoint tests.
func (o *ContextOperator[K, T]) Extract(ctx context.Context) (T, bool) {
	o.mx.RLock()
	defer o.mx.RUnlock()

	var def T
	c := ctx.Value(o.key)
	if c == nil {
		return def, false
	}
	val, ok := c.(*T)
	return *val, ok
}
