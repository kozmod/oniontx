package saga

import (
	"context"
	"testing"

	"github.com/kozmod/oniontx/internal/testtool/assert"
)

func TestCtxFactory_Apply(t *testing.T) {
	t.Run("nil_factory", func(t *testing.T) {
		ctx := context.Background()
		var factory CtxFactory

		assert.Equal(t, ctx, factory.Apply(ctx))
	})

	t.Run("nil_result", func(t *testing.T) {
		ctx := context.Background()
		factory := CtxFactory(func(context.Context) context.Context {
			return nil
		})

		assert.Equal(t, ctx, factory.Apply(ctx))
	})

	t.Run("transformed_context", func(t *testing.T) {
		type contextKey struct{}

		ctx := context.Background()
		factory := CtxFactory(func(ctx context.Context) context.Context {
			return context.WithValue(ctx, contextKey{}, "value")
		})

		result := factory.Apply(ctx)

		assert.Equal(t, "value", result.Value(contextKey{}))
	})
}
