package saga

import (
	"context"
	"fmt"
	"testing"

	"github.com/kozmod/oniontx"
	"github.com/kozmod/oniontx/test/integration/internal/entity"
	"github.com/kozmod/oniontx/test/integration/internal/stdlib"

	"github.com/stretchr/testify/assert"
)

func Test_Saga_stdlib_Facade(t *testing.T) {
	const (
		textRecord = "text_Saga_1"
	)
	var (
		db        = stdlib.ConnectDB(t)
		cleanupFn = func() {
			err := stdlib.ClearDB(db)
			assert.NoError(t, err)
		}
	)
	defer func() {
		err := db.Close()
		assert.NoError(t, err)
	}()

	cleanupFn()

	t.Run("success_exec_compensation_on_step2", func(t *testing.T) {
		t.Cleanup(cleanupFn)

		var (
			ctx        = context.Background()
			transactor = stdlib.NewTransactor(db)
			repoA      = stdlib.NewTextRepository(transactor, false)
			repoB      = stdlib.NewTextRepository(transactor, true)
		)
		err := oniontx.NewSaga([]oniontx.Step{
			{
				Name: "step_0",
				Action: func(ctx context.Context) error {
					err := transactor.WithinTx(ctx, func(ctx context.Context) error {
						return repoA.Insert(ctx, textRecord)
					})
					assert.NoError(t, err)
					return nil
				},
				Compensation: func(ctx context.Context, aroseErr error) error {
					assert.Error(t, aroseErr)
					assert.ErrorIs(t, aroseErr, entity.ErrExpected)

					err := repoA.Delete(ctx, textRecord)
					assert.NoError(t, err)
					return err
				},
			},
			{
				Name: "step_1",
				Action: func(ctx context.Context) error {
					records, err := stdlib.GetTextRecords(db)
					assert.NoError(t, err)
					assert.Len(t, records, 1)
					assert.ElementsMatch(t, []string{textRecord}, records)

					return nil
				},
			},
			{
				Name: "step_2",
				Action: func(ctx context.Context) error {
					err := transactor.WithinTx(ctx, func(ctx context.Context) error {
						err := repoA.Insert(ctx, textRecord)
						if err != nil {
							return fmt.Errorf("step_2 - repoA error: %w", err)
						}
						err = repoB.Insert(ctx, textRecord) // will fail (entity.ErrExpected)
						if err != nil {
							return fmt.Errorf("step_2 - repoB error: %w", err)
						}

						assert.Fail(t, "step_2 - repoB is expected to fail")
						return nil
					})

					assert.Error(t, err)
					assert.ErrorIs(t, err, entity.ErrExpected)
					return err
				},
			},
		}).Execute(ctx)

		assert.Error(t, err)

		t.Logf("test error output: \n{\n%v\n}", err)

		assert.ErrorIs(t, err, entity.ErrExpected)
		assert.ErrorIs(t, err, oniontx.ErrSagaActionFailed)
		assert.ErrorIs(t, err, oniontx.ErrRollbackSuccess)
		assert.ErrorIs(t, err, oniontx.ErrSagaCompensationSuccess)

		{
			records, err := stdlib.GetTextRecords(db)
			assert.NoError(t, err)
			assert.Len(t, records, 0)
		}
	})
}
