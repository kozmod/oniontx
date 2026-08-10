package saga

import (
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
