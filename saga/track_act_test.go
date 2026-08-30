package saga

import (
	"errors"
	"testing"

	"github.com/kozmod/oniontx/internal/testtool/require"
)

func Test_ExecutionTrackApply(t *testing.T) {
	t.Run("act_update_track_atomically", func(t *testing.T) {
		expectedErr := errors.New("expected")
		track := newExecutionTrack(nil, ExecutionStatusUncalled)

		track.apply(newTrackCalledAct())
		track.apply(newTrackFailedAct(expectedErr))

		data := track.GetTrackData()
		require.Equal(t, 1, data.Calls)
		require.Equal(t, ExecutionStatusFail, data.Status)
		require.Equal(t, 1, len(data.Errors))
		require.ErrorIs(t, data.Errors[0], expectedErr)
	})

	t.Run("nil_failure_error_still_sets_failed_status", func(t *testing.T) {
		track := newExecutionTrack(nil, ExecutionStatusUncalled)

		track.apply(newTrackFailedAct(nil))

		data := track.GetTrackData()
		require.Equal(t, ExecutionStatusFail, data.Status)
		require.Equal(t, 0, len(data.Errors))
	})

	t.Run("unset_is_initial_state", func(t *testing.T) {
		tracker := newInMemoryTrack(0, NewStep("empty"))

		data := tracker.GetStepData()
		require.Equal(t, ExecutionStatusUnset, data.Action.Status)
		require.Equal(t, ExecutionStatusUnset, data.Compensation.Status)
	})
}

func Test_ExecutionRetryTrack_apply(t *testing.T) {
	t.Run("forwards_call_and_success_acts", func(t *testing.T) {
		track := newExecutionTrack(nil, ExecutionStatusUncalled)
		retryTrack := newExecutionRetryTrack(track, 2)

		retryTrack.apply(newTrackCalledAct())
		retryTrack.apply(newTrackSucceededAct())

		data := track.GetTrackData()
		require.Equal(t, 1, data.Calls)
		require.Equal(t, ExecutionStatusSuccess, data.Status)
	})

	t.Run("wraps_failure_once", func(t *testing.T) {
		expectedErr := errors.New("expected")
		track := newExecutionTrack(nil, ExecutionStatusUncalled)
		retryTrack := newExecutionRetryTrack(track, 2)

		retryTrack.apply(newTrackFailedAct(expectedErr))

		data := track.GetTrackData()
		require.Equal(t, ExecutionStatusFail, data.Status)
		require.Equal(t, 1, len(data.Errors))
		require.ErrorIs(t, data.Errors[0], expectedErr)
		require.Equal(t, "retry [2]: expected", data.Errors[0].Error())
	})

	t.Run("forwards_failure_with_nil_error", func(t *testing.T) {
		track := newExecutionTrack(nil, ExecutionStatusUncalled)
		retryTrack := newExecutionRetryTrack(track, 0)

		retryTrack.apply(newTrackFailedAct(nil))

		data := track.GetTrackData()
		require.Equal(t, ExecutionStatusFail, data.Status)
		require.Equal(t, 0, len(data.Errors))
	})
}
