package saga

import (
	"context"
	"fmt"
	"testing"

	"github.com/kozmod/oniontx/internal/testtool"
	"github.com/kozmod/oniontx/saga"
	"github.com/kozmod/oniontx/test/integration/internal/entity"
	"github.com/kozmod/oniontx/test/integration/internal/stdlib"

	"github.com/stretchr/testify/require"
)

func Test_Saga_stdlib_Facade(t *testing.T) {
	const (
		textRecord = "text_Saga_1"
	)
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

	t.Run("success_exec_compensation_on_step2", func(t *testing.T) {
		t.Cleanup(cleanupFn)

		var (
			ctx        = context.Background()
			transactor = stdlib.NewTransactor(db)
			repoA      = stdlib.NewTextRepository(transactor, false)
			repoB      = stdlib.NewTextRepository(transactor, true)
		)
		res, err := saga.NewSaga([]saga.Step{
			saga.NewStep("step_0").
				WithAction(saga.NewOperation(func(ctx context.Context, _ saga.Track) error {
					err := transactor.WithinTx(ctx, func(ctx context.Context) error {
						return repoA.Insert(ctx, textRecord)
					})
					require.NoError(t, err)
					return nil
				})).
				WithCompensation(saga.NewOperation(func(ctx context.Context, track saga.Track) error {
					//data := track.GetStepData()
					//require.Len(t, data.Action.Errors, 1)
					//require.ErrorIs(t, data.Action.Errors[0], entity.ErrExpected)

					err := repoA.Delete(ctx, textRecord)
					require.NoError(t, err)
					return err
				})),
			saga.NewStep("step_1").
				WithAction(saga.NewOperation(func(ctx context.Context, _ saga.Track) error {
					records, err := stdlib.GetTextRecords(db)
					require.NoError(t, err)
					require.Len(t, records, 1)
					require.ElementsMatch(t, []string{textRecord}, records)

					return nil
				})),
			saga.NewStep("step_2").
				WithAction(saga.NewOperation(func(ctx context.Context, _ saga.Track) error {
					err := transactor.WithinTx(ctx, func(ctx context.Context) error {
						err := repoA.Insert(ctx, textRecord)
						if err != nil {
							return fmt.Errorf("step_2 - repoA error: %w", err)
						}
						err = repoB.Insert(ctx, textRecord) // will fail (entity.ErrExpected)
						if err != nil {
							return fmt.Errorf("step_2 - repoB error: %w", err)
						}

						require.Fail(t, "step_2 - repoB is expected to fail")
						return nil
					})

					require.Error(t, err)
					require.ErrorIs(t, err, entity.ErrExpected)
					return err
				})),
		}).Execute(ctx)

		require.Error(t, err)
		require.ErrorIs(t, err, saga.ErrActionFailed)

		require.Equal(t, saga.StageResultCompensated, res.Status)
		require.Len(t, res.Steps, 3)

		require.Equal(t, saga.ExecutionStatusSuccess, res.Steps[0].Action.Status)
		require.Equal(t, saga.ExecutionStatusSuccess, res.Steps[0].Compensation.Status)

		require.Equal(t, saga.ExecutionStatusSuccess, res.Steps[1].Action.Status)
		require.Equal(t, saga.ExecutionStatusUnset, res.Steps[1].Compensation.Status)

		require.Equal(t, saga.ExecutionStatusFail, res.Steps[2].Action.Status)
		require.Equal(t, saga.ExecutionStatusUnset, res.Steps[2].Compensation.Status)

		require.Len(t, res.Steps[2].Action.Errors, 1)
		require.ErrorIs(t, res.Steps[2].Action.Errors[0], saga.ErrActionFailed)

		testtool.TestFn(t, func() {
			printResult(t, res, err)
		})

		{
			records, err := stdlib.GetTextRecords(db)
			require.NoError(t, err)
			require.Len(t, records, 0)
		}
	})
}
