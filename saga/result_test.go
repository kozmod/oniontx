package saga

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/kozmod/oniontx/internal/testtool"
	"github.com/kozmod/oniontx/internal/testtool/require"
)

func Test_Result_String(t *testing.T) {
	res := Result{
		Status: StageResultFail,
		Steps: []StepData{
			{
				StepPosition: 0,
				StepName:     "payment",
				Action: TrackData{
					Calls: 2,
					Errors: []error{
						fmt.Errorf("card declined"),
						fmt.Errorf("retry failed"),
					},
					Status: ExecutionStatusFail,
				},
				Compensation: TrackData{
					Calls:  1,
					Status: ExecutionStatusSuccess,
				},
			},
		},
	}

	t.Run("without_track_data_errors", func(t *testing.T) {
		require.False(t,
			strings.Contains(res.String(), "Errors: 2 ["),
		)

		testtool.TestFn(t, func() {
			t.Log(
				res,
			)
		})
	})

	t.Run("with_track_data_errors", func(t *testing.T) {
		resWithErrors := res.WithErrorsInTrackDataString()

		require.True(t,
			strings.Contains(resWithErrors.String(), "Errors: 2 [card declined, retry failed]"),
		)

		testtool.TestFn(t, func() {
			t.Log(
				resWithErrors,
			)
		})
	})
}

func Test_SagaExecute_preservesExecutionErrors(t *testing.T) {
	var (
		actionErr       = fmt.Errorf("action error")
		compensationErr = fmt.Errorf("compensation error")
		steps           = []Step{
			NewStep("completed step").
				WithAction(NewOperation(func(context.Context, Track) error {
					return nil
				})).
				WithCompensation(NewOperation(func(context.Context, Track) error {
					return compensationErr
				})),
			NewStep("failed step").
				WithAction(NewOperation(func(context.Context, Track) error {
					return actionErr
				})),
		}
	)

	res, err := NewSaga(steps).
		Execute(context.Background())
	require.NotNil(t, res)
	require.NotNil(t, res.Status)
	require.Equal(t, StageResultFail, res.Status)
	require.Equal(t, 2, len(res.Steps))

	require.Equal(t, ExecutionStatusSuccess, res.Steps[0].Action.Status)
	require.Equal(t, ExecutionStatusFail, res.Steps[0].Compensation.Status)
	require.Equal(t, 1, res.Steps[0].Compensation.Calls)
	require.ErrorIs(t, res.Steps[0].Compensation.Errors[0], compensationErr)

	require.Equal(t, ExecutionStatusFail, res.Steps[1].Action.Status)
	require.Equal(t, 1, res.Steps[1].Action.Calls)
	require.ErrorIs(t, res.Steps[1].Action.Errors[0], actionErr)

	require.Len(t, res.Errors(), 2)

	require.True(t, slices.ContainsFunc(res.Errors(), func(err error) bool {
		return errors.Is(err, actionErr)
	}))
	require.True(t, slices.ContainsFunc(res.Errors(), func(err error) bool {
		return errors.Is(err, compensationErr)
	}))

	require.ErrorIs(t, err, ErrActionFailed)
	require.ErrorIs(t, err, ErrCompensationFailed)
	require.ErrorIs(t, err, actionErr)
	require.ErrorIs(t, err, compensationErr)
}

func Test_SagaExecute_actionFailureWithoutCompensation(t *testing.T) {
	steps := []Step{
		NewStep("failed step").
			WithAction(NewOperation(func(context.Context, Track) error {
				return testtool.ErrExpTestA
			})),
	}

	res, err := NewSaga(steps).
		Execute(context.Background())
	require.NotNil(t, res)
	require.Equal(t, StageResultFail, res.Status)
	require.ErrorIs(t, err, ErrActionFailed)
	require.ErrorIsNot(t, err, ErrCompensationFailed)
}

func Test_Saga_Execute_Cancellation_Inside_Operation(t *testing.T) {
	for _, compensationCase := range []struct {
		name      string
		condition bool
	}{
		{
			name:      "action",
			condition: false,
		},
		{
			name:      "compensation",
			condition: true,
		},
	} {
		t.Run(compensationCase.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			canceledOperation := NewOperation(func(ctx context.Context, _ Track) error {
				cancel()
				return fmt.Errorf("operation interrupted: %w", ctx.Err())
			})
			step := NewStep("step").
				WithAction(canceledOperation)
			switch {
			case compensationCase.condition:
				step = NewStep("step").
					WithAction(NewOperation(func(context.Context, Track) error {
						return testtool.ErrExpTestA
					})).
					WithCompensation(canceledOperation).
					WithCompensationOnActionFailure()
			}
			result, err := NewSaga([]Step{step}).
				Execute(ctx)
			require.Equal(t, StageResultFail, result.Status)
			require.ErrorIs(t, err, context.Canceled)
			require.ErrorIs(t, err, ErrActionFailed)

			switch {
			case compensationCase.condition:
				require.ErrorIs(t, err, testtool.ErrExpTestA)
				require.ErrorIs(t, err, ErrCompensationFailed)
				require.Equal(t, ExecutionStatusFail, result.Steps[0].Compensation.Status)
			default:
				require.ErrorIsNot(t, err, ErrCompensationFailed)
				require.Equal(t, ExecutionStatusFail, result.Steps[0].Action.Status)
			}
		})
	}
}

func Test_Saga_Execute_Successful_Retry_Errors_Stay_In_History(t *testing.T) {
	for _, failNextCase := range []struct {
		name     string
		failNext bool
	}{
		{
			name:     "success",
			failNext: false,
		},
		{
			name:     "compensation",
			failNext: true,
		},
	} {
		t.Run(failNextCase.name, func(t *testing.T) {
			calls := 0
			steps := []Step{
				NewStep("retried").
					WithAction(NewOperation(func(context.Context, Track) error {
						calls++
						if calls == 1 {
							return testtool.ErrExpTestA
						}
						return nil
					}).
						WithRetry(NewBaseRetryPolicy(1, 0))).
					WithCompensation(NewOperation(nil)),
			}
			switch {
			case failNextCase.failNext:
				steps = append(steps,
					NewStep("failed").
						WithAction(NewOperation(func(context.Context, Track) error {
							return testtool.ErrExpTestB
						})),
				)
			}
			result, err := NewSaga(steps).
				Execute(context.Background())
			require.Equal(t, 2, calls)
			require.True(t, slices.ContainsFunc(result.Errors(), func(err error) bool {
				return errors.Is(err, testtool.ErrExpTestA)
			}))
			require.ErrorIsNot(t, err, testtool.ErrExpTestA)

			switch {
			case failNextCase.failNext:
				require.Equal(t, StageResultCompensated, result.Status)
				require.ErrorIs(t, err, testtool.ErrExpTestB)
				require.ErrorIs(t, err, ErrActionFailed)
			default:
				require.Equal(t, StageResultSuccess, result.Status)
				require.NoError(t, err)
				require.Equal(t, ExecutionStatusUncalled, result.Steps[0].Compensation.Status)
			}
		})
	}
}

func Test_Saga_Execute_Successful_Compensation_Retry(t *testing.T) {
	calls := 0
	result, err := NewSaga([]Step{
		NewStep("step").
			WithAction(
				NewOperation(func(context.Context, Track) error {
					return testtool.ErrExpTestA
				})).
			WithCompensation(
				NewOperation(func(context.Context, Track) error {
					calls++
					if calls == 1 {
						return testtool.ErrExpTestB
					}
					return nil
				}).
					WithRetry(NewBaseRetryPolicy(1, 0))).
			WithCompensationOnActionFailure(),
	}).Execute(context.Background())
	require.NotNil(t, result)
	require.Equal(t, StageResultCompensated, result.Status)
	require.ErrorIs(t, err, testtool.ErrExpTestA)
	require.ErrorIsNot(t, err, testtool.ErrExpTestB)
	require.ErrorIsNot(t, err, ErrCompensationFailed)
	require.True(t, slices.ContainsFunc(result.Errors(), func(err error) bool {
		return errors.Is(err, testtool.ErrExpTestB)
	}))
}

func Test_Result_String_Includes_Execution_Interruption(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result, err := NewSaga(
		[]Step{
			NewStep("skipped").
				WithAction(NewOperation(nil)),
		}).Execute(ctx)
	require.ErrorIs(t, err, context.Canceled)
	require.True(t, strings.Contains(result.String(), "Execution error: "))
	require.True(t, strings.Contains(result.String(), ErrExecuteActionsContextDone.Error()))
	require.True(t, strings.Contains(result.String(), context.Canceled.Error()))
	require.Len(t, result.Steps[0].Action.Errors, 0)
}

func Test_Saga_Successful_Action_Does_Not_Require_Missing_Compensation(t *testing.T) {
	for _, canceledCase := range []struct {
		name     string
		canceled bool
	}{
		{
			name:     "later_action_failure",
			canceled: false,
		},
		{
			name:     "later_cancellation",
			canceled: true,
		},
	} {
		t.Run(canceledCase.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			result, err := NewSaga([]Step{
				NewStep("completed").
					WithAction(NewOperation(func(context.Context, Track) error {
						switch {
						case canceledCase.canceled:
							cancel()
						}
						return nil
					})).
					WithCompensationOnActionFailure(),
				NewStep("next").
					WithAction(NewOperation(func(context.Context, Track) error {
						switch {
						case canceledCase.canceled:
							t.Fatal("action must not run after cancellation")
						}
						return testtool.ErrExpTestA
					})),
			}).Execute(ctx)
			require.Equal(t, StageResultFail, result.Status)
			require.ErrorIsNot(t, err, ErrCompensationFailed)
			require.ErrorIsNot(t, err, ErrCompensationNotConfigured)
			require.Equal(t, ExecutionStatusSuccess, result.Steps[0].Action.Status)
			require.Equal(t, ExecutionStatusUnset, result.Steps[0].Compensation.Status)
			require.Len(t, result.Steps[0].Compensation.Errors, 0)

			switch {
			case canceledCase.canceled:
				require.ErrorIs(t, err, context.Canceled)
				require.ErrorIs(t, err, ErrExecuteActionsContextDone)
				require.ErrorIsNot(t, err, ErrActionFailed)
			default:
				require.ErrorIs(t, err, testtool.ErrExpTestA)
				require.ErrorIs(t, err, ErrActionFailed)
			}
		})
	}
}
