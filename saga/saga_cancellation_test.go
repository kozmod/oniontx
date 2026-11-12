package saga

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/kozmod/oniontx/internal/testtool/require"
)

func Test_Saga_Cancellation_Before_Action(t *testing.T) {
	for _, deadline := range []bool{false, true} {
		name := "canceled"
		if deadline {
			name = "deadline"
		}
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			expected := context.Canceled
			if deadline {
				cancel()
				ctx, cancel = context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
				expected = context.DeadlineExceeded
			}
			cancel()
			unexpected := NewOperation(func(context.Context, Track) error {
				t.Fatal("operation must not run")
				return nil
			})
			result, err := NewSaga([]Step{
				NewStep("skipped").WithAction(unexpected).WithCompensation(unexpected).WithCompensationOnActionFailure(),
				NewStep("future").WithAction(unexpected).WithCompensation(unexpected),
			}).Execute(ctx)
			require.Equal(t, StageResultFail, result.Status)
			require.ErrorIs(t, err, expected)
			require.ErrorIs(t, err, ErrExecuteActionsContextDone)
			require.ErrorIsNot(t, err, ErrActionFailed)
			require.ErrorIsNot(t, err, ErrCompensationFailed)
			require.Len(t, result.Errors(), 1)
			require.ErrorIs(t, result.Errors()[0], expected)
			for _, step := range result.Steps {
				require.Equal(t, ExecutionStatusUncalled, step.Action.Status)
				require.Equal(t, 0, step.Action.Calls)
				require.Len(t, step.Action.Errors, 0)
				require.Equal(t, ExecutionStatusUncalled, step.Compensation.Status)
				require.Equal(t, 0, step.Compensation.Calls)
				require.Len(t, step.Compensation.Errors, 0)
			}
		})
	}
}

func Test_Saga_Compensation_Context_Stops_Rollback(t *testing.T) {
	compensationCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	actionErr := errors.New("action failed")
	var calls []string
	compensation := func(name string) Operation {
		return NewOperation(func(ctx context.Context, _ Track) error {
			require.Equal(t, compensationCtx, ctx)
			calls = append(calls, name)
			cancel()
			return nil
		})
	}
	result, err := NewSaga([]Step{
		NewStep("first").WithAction(NewOperation(nil)).WithCompensation(compensation("first")),
		NewStep("second").WithAction(NewOperation(nil)).WithCompensation(compensation("second")),
		NewStep("failed").WithAction(NewOperation(func(context.Context, Track) error { return actionErr })),
		NewStep("future").WithAction(NewOperation(nil)).WithCompensation(compensation("future")).WithCompensationOnActionFailure(),
	}).WithCompensationContext(func(context.Context) context.Context { return compensationCtx }).Execute(context.Background())
	require.Equal(t, StageResultFail, result.Status)
	require.True(t, slices.Equal([]string{"second"}, calls))
	require.ErrorIs(t, err, ErrActionFailed)
	require.ErrorIs(t, err, ErrCompensationFailed)
	require.ErrorIs(t, err, ErrExecuteCompensationContextDone)
	require.ErrorIs(t, err, context.Canceled)
	require.ErrorIsNot(t, err, ErrExecuteActionsContextDone)
	require.Equal(t, ExecutionStatusUncalled, result.Steps[0].Compensation.Status)
	require.Equal(t, ExecutionStatusSuccess, result.Steps[1].Compensation.Status)
	require.Equal(t, ExecutionStatusUncalled, result.Steps[3].Compensation.Status)
}

func Test_Saga_Cancellation_Ignores_Unset_Actions(t *testing.T) {
	for _, withAction := range []bool{false, true} {
		name := "only_unset"
		if withAction {
			name = "unset_after_completed_action"
		}
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var steps []Step
			if withAction {
				steps = append(steps, NewStep("completed").WithAction(NewOperation(func(context.Context, Track) error {
					cancel()
					return nil
				})).WithCompensation(NewOperation(func(context.Context, Track) error {
					t.Fatal("completed workflow must not be compensated")
					return nil
				})))
			} else {
				cancel()
			}
			steps = append(steps, NewStep("unset"), NewStep("another unset"))
			result, err := NewSaga(steps).Execute(ctx)
			require.NoError(t, err)
			require.Equal(t, StageResultSuccess, result.Status)
			require.Len(t, result.Errors(), 0)
			require.Equal(t, ExecutionStatusUnset, result.Steps[len(steps)-1].Action.Status)
			if withAction {
				require.Equal(t, ExecutionStatusSuccess, result.Steps[0].Action.Status)
				require.Equal(t, ExecutionStatusUncalled, result.Steps[0].Compensation.Status)
			}
		})
	}
}
