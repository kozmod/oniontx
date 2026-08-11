package saga

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/kozmod/oniontx/internal/testtool"
	"github.com/kozmod/oniontx/internal/testtool/assert"
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
		assert.False(t,
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

		assert.True(t,
			strings.Contains(resWithErrors.String(), "Errors: 2 [card declined, retry failed]"),
		)

		testtool.TestFn(t, func() {
			t.Log(
				resWithErrors,
			)
		})
	})
}

func Test_prepareResultSliceErrorMessage(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		res := prepareResultSliceErrorMessage([]string{})
		assert.Equal(t, "0", res)

		res = prepareResultSliceErrorMessage(nil)
		assert.Equal(t, "0", res)
	})

	t.Run("not_empty", func(t *testing.T) {
		var (
			someData = []string{"some_data"}
		)

		res := prepareResultSliceErrorMessage(someData)
		assert.Equal(t, fmt.Sprintf("%d: %s", len(someData), someData[0]), res)
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

	result, err := NewSaga(steps).Execute(context.Background())
	assert.NotNil(t, result)
	assert.NotNil(t, result.Status)
	assert.Equal(t, StageResultFail, result.Status)
	assert.Equal(t, 2, len(result.Steps))

	assert.Equal(t, ExecutionStatusSuccess, result.Steps[0].Action.Status)
	assert.Equal(t, ExecutionStatusFail, result.Steps[0].Compensation.Status)
	assert.Equal(t, 1, result.Steps[0].Compensation.Calls)
	assert.ErrorIs(t, result.Steps[0].Compensation.Errors[0], compensationErr)

	assert.Equal(t, ExecutionStatusFail, result.Steps[1].Action.Status)
	assert.Equal(t, 1, result.Steps[1].Action.Calls)
	assert.ErrorIs(t, result.Steps[1].Action.Errors[0], actionErr)

	assert.ErrorIs(t, err, ErrActionFailed)
	assert.ErrorIs(t, err, ErrCompensationFailed)
	assert.ErrorIs(t, err, actionErr)
	assert.ErrorIs(t, err, compensationErr)
}
