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

	res, err := NewSaga(steps).Execute(context.Background())
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
	require.ErrorIsNot(t, err, actionErr)
	require.ErrorIsNot(t, err, compensationErr)
}

func Test_SagaExecute_actionFailureWithoutCompensation(t *testing.T) {
	steps := []Step{
		NewStep("failed step").
			WithAction(NewOperation(func(context.Context, Track) error {
				return testtool.ErrExpTestA
			})),
	}

	res, err := NewSaga(steps).Execute(context.Background())
	require.NotNil(t, res)
	require.Equal(t, StageResultFail, res.Status)
	require.ErrorIs(t, err, ErrActionFailed)
	require.ErrorIsNot(t, err, ErrCompensationFailed)
}
