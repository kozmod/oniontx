package saga

import (
	"context"
	"fmt"
	"testing"

	"github.com/kozmod/oniontx"
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

	t.Run("success_exec_compensation_on_step3", func(t *testing.T) {
		t.Cleanup(cleanupFn)

		var (
			ctx        = context.Background()
			transactor = stdlib.NewTransactor(db)
			repoA      = stdlib.NewTextRepository(transactor, false)
			repoB      = stdlib.NewTextRepository(transactor, true)
		)
		err := oniontx.NewSaga([]oniontx.Step{
			{
				Name:       "step0",
				Transactor: transactor,
				Action: func(ctx context.Context) error {
					err := repoA.Insert(ctx, textRecord)
					assert.NoError(t, err)
					return err
				},
				Compensation: func(ctx context.Context) error {
					err := repoA.Delete(ctx, textRecord)
					return err
				},
			},
			{
				Name: "step1",
				Action: func(ctx context.Context) error {
					records, err := stdlib.GetTextRecords(db)
					assert.NoError(t, err)
					assert.Len(t, records, 1)
					assert.ElementsMatch(t, []string{textRecord}, records)

					return nil
				},
			},
			{
				Name:       "step2",
				Transactor: transactor,
				Action: func(ctx context.Context) error {
					err := repoA.Insert(ctx, textRecord)
					if err != nil {
						return fmt.Errorf("step3 - repoA error: %w", err)
					}
					err = repoB.Insert(ctx, textRecord)
					if err != nil {
						return fmt.Errorf("step3 - repoB error: %w", err)
					}
					return nil
				},
			},
		}).Execute(ctx)

		t.Logf("test error \n{\n%v\n}", err)

		assert.Error(t, err)
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
