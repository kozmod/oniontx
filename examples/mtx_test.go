package examples

import (
	"context"
	"fmt"
	"testing"

	"github.com/kozmod/oniontx/mtx"
)

type mtxExampleKey struct{}

type mtxExampleDB struct{}

func (mtxExampleDB) BeginTx(context.Context) (*mtxExampleTx, error) {
	return &mtxExampleTx{}, nil
}

type mtxExampleTx struct{}

func (*mtxExampleTx) Commit(context.Context) error {
	return nil
}

func (*mtxExampleTx) Rollback(context.Context) error {
	return nil
}

// Test_Mtx_example demonstrates transaction setup, nested calls, and rollback
// contexts with the public mtx API.
func Test_Mtx_example(t *testing.T) {
	t.Skipf("mtx.Transactor example")

	var (
		ctx        = context.Background()
		db         = mtxExampleDB{}
		operator   = mtx.NewContextOperator[mtxExampleKey, *mtxExampleTx](mtxExampleKey{})
		transactor = mtx.NewTransactor[mtxExampleDB, *mtxExampleTx](db, operator)

		repoAInsert = func(ctx context.Context) error {
			// Repository code retrieves the transaction from context.
			tx, ok := transactor.TryGetTx(ctx)
			if !ok || tx == nil {
				return fmt.Errorf("transaction is required")
			}
			return nil
		}
		repoBInsert = func(ctx context.Context) error {
			tx, ok := transactor.TryGetTx(ctx)
			if !ok || tx == nil {
				return fmt.Errorf("transaction is required")
			}
			return nil
		}
	)

	t.Run("single transaction for multiple repositories", func(t *testing.T) {
		err := transactor.WithinTx(ctx, func(ctx context.Context) error {
			if err := repoAInsert(ctx); err != nil {
				return fmt.Errorf("insert to repository_A: %w", err)
			}
			if err := repoBInsert(ctx); err != nil {
				return fmt.Errorf("insert to repository_B: %w", err)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("nested calls reuse the outer transaction", func(t *testing.T) {
		err := transactor.WithinTx(ctx, func(ctx context.Context) error {
			outerTx, ok := transactor.TryGetTx(ctx)
			if !ok {
				return fmt.Errorf("outer transaction is missing")
			}

			return transactor.WithinTx(ctx, func(ctx context.Context) error {
				innerTx, ok := transactor.TryGetTx(ctx)
				if !ok || innerTx != outerTx {
					return fmt.Errorf("nested call must reuse the outer transaction")
				}
				return repoAInsert(ctx)
			})
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("rollback can outlive request cancellation", func(t *testing.T) {
		rollbackTransactor := transactor.WithRollbackCtxFactory(context.WithoutCancel)

		requestCtx, cancel := context.WithCancel(ctx)
		cancel()

		err := rollbackTransactor.WithinTx(requestCtx, func(context.Context) error {
			return fmt.Errorf("operation failed")
		})
		if err == nil {
			t.Fatal("error expected")
		}
	})
}
