package savepoint

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kozmod/oniontx/test/integration/internal/entity"
	"github.com/kozmod/oniontx/test/integration/internal/stdlib"
)

func Test_Savepoint(t *testing.T) {
	var (
		db        = stdlib.ConnectDB(t)
		cleanupFn = func() {
			err := stdlib.ClearDB(db)
			require.NoError(t, err)
		}
	)
	defer func() {
		err := db.Close()
		require.NoError(t, err)
	}()

	cleanupFn()

	t.Run("success_insert_use_savepoint", func(t *testing.T) {
		t.Cleanup(cleanupFn)

		var (
			ctx         = context.Background()
			transactor  = stdlib.NewTransactor(db)
			repositoryA = stdlib.NewTextRepository(transactor, false)
			repositoryB = stdlib.NewTextRepository(transactor, true)
		)

		err := transactor.WithinTx(ctx, func(ctx context.Context) error {
			const (
				savepoint = "A"
			)
			err := repositoryA.Insert(ctx, "A")
			if err != nil {
				return fmt.Errorf("text repo A: %w", err)
			}

			err = repositoryA.Savepoint(ctx, savepoint)
			if err != nil {
				return fmt.Errorf("repo A - create savepoint [%s]: %w", savepoint, err)
			}

			// This successful write must be undone when the later operation fails.
			err = repositoryA.Insert(ctx, "B")
			if err != nil {
				return fmt.Errorf("text repo A - insert after savepoint: %w", err)
			}

			err = repositoryB.Insert(ctx, "C")
			if err != nil {
				return fmt.Errorf("text repo B: %w", err)
			}
			return nil
		})
		require.Error(t, err)
		require.ErrorIs(t, err, entity.ErrExpected)

		{
			records, err := stdlib.GetTextRecords(db)
			require.NoError(t, err)
			require.Len(t, records, 1)
			require.ElementsMatch(t, []string{"A"}, records)
		}
	})
}
