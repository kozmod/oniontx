package saga

import (
	"errors"
	"testing"

	"github.com/kozmod/oniontx/internal/testtool/assert"
)

func TestExecutionTrack_apply(t *testing.T) {
	t.Run("act_update_track_atomically", func(t *testing.T) {
		expectedErr := errors.New("expected")
		track := NewExecutionTrack(nil)

		track.apply(newTrackCalledAct())
		track.apply(newTrackFailedAct(expectedErr))

		data := track.GetTrackData()
		assert.Equal(t, 1, data.Calls)
		assert.Equal(t, ExecutionStatusFail, data.Status)
		assert.Equal(t, 1, len(data.Errors))
		assert.ErrorIs(t, data.Errors[0], expectedErr)
	})

	t.Run("nil_failure_error_still_sets_failed_status", func(t *testing.T) {
		track := NewExecutionTrack(nil)

		track.apply(newTrackFailedAct(nil))

		data := track.GetTrackData()
		assert.Equal(t, ExecutionStatusFail, data.Status)
		assert.Equal(t, 0, len(data.Errors))
	})

	t.Run("unset_is_initial_state", func(t *testing.T) {
		tracker := newInMemoryTrack(0, NewStep("empty"))

		data := tracker.GetStepData()
		assert.Equal(t, ExecutionStatusUnset, data.Action.Status)
		assert.Equal(t, ExecutionStatusUnset, data.Compensation.Status)
	})
}

func TestExecutionRetryTrack_apply(t *testing.T) {
	t.Run("forwards_call_and_success_acts", func(t *testing.T) {
		track := NewExecutionTrack(nil)
		retryTrack := newExecutionRetryTrack(track, 2)

		retryTrack.apply(newTrackCalledAct())
		retryTrack.apply(newTrackSucceededAct())

		data := track.GetTrackData()
		assert.Equal(t, 1, data.Calls)
		assert.Equal(t, ExecutionStatusSuccess, data.Status)
	})

	t.Run("wraps_failure_once", func(t *testing.T) {
		expectedErr := errors.New("expected")
		track := NewExecutionTrack(nil)
		retryTrack := newExecutionRetryTrack(track, 2)

		retryTrack.apply(newTrackFailedAct(expectedErr))

		data := track.GetTrackData()
		assert.Equal(t, ExecutionStatusFail, data.Status)
		assert.Equal(t, 1, len(data.Errors))
		assert.ErrorIs(t, data.Errors[0], expectedErr)
		assert.Equal(t, "retry [2]: expected", data.Errors[0].Error())
	})

	t.Run("forwards_failure_with_nil_error", func(t *testing.T) {
		track := NewExecutionTrack(nil)
		retryTrack := newExecutionRetryTrack(track, 0)

		retryTrack.apply(newTrackFailedAct(nil))

		data := track.GetTrackData()
		assert.Equal(t, ExecutionStatusFail, data.Status)
		assert.Equal(t, 0, len(data.Errors))
	})
}
