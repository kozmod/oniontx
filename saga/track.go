package saga

import (
	"fmt"
	"slices"
	"strings"
)

// ExecutionStatus represents the current state of an action or compensation execution.
type ExecutionStatus string

const (
	// ExecutionStatusSuccess indicates the operation completed successfully.
	ExecutionStatusSuccess ExecutionStatus = "Success"
	// ExecutionStatusFail indicates the operation failed.
	ExecutionStatusFail ExecutionStatus = "Fail"
	// ExecutionStatusUncalled indicates the operation has not been invoked.
	ExecutionStatusUncalled ExecutionStatus = "Uncalled"
	// ExecutionStatusUnset indicates the operation is not configured (e.g., nil function).
	ExecutionStatusUnset ExecutionStatus = "Unset"
)

type (
	// Tracker provides access to step data for a specific saga step.
	Tracker interface {
		GetStepData() StepData
	}

	// Track represents an executable operation within a saga step.
	Track interface {
		Apply(Act)

		GetStepData() StepData
		GetTrackData() TrackData
	}
)

// StepData contains the complete execution history for a single saga step.
type StepData struct {
	// StepPosition is the index of this step in the saga.
	StepPosition uint32
	StepName     string

	// Action is the execution data for the main action.
	Action TrackData
	// Compensation is the execution data for the compensation operation.
	Compensation TrackData

	// CompensationRequired reports whether this step can compensate its own action failure.
	CompensationRequired bool
}

// String returns a human-readable representation of the StepData.
func (s StepData) String() string {
	return fmt.Sprintf("Step %d: %s | Action: %s | Compensation: %s",
		s.StepPosition,
		s.StepName,
		s.Action.String(),
		s.Compensation.String(),
	)
}

// TrackData contains execution metrics for a single operation snapshot.
type TrackData struct {
	Calls  uint32
	Errors []error
	Status ExecutionStatus
}

// String returns a compact representation of TrackData.
func (ed *TrackData) String() string {
	var builder strings.Builder
	switch ed {
	case nil:
		_, _ = fmt.Fprintf(&builder, "{Status: %s, Calls: %d", "nil", -1)
	default:
		_, _ = fmt.Fprintf(&builder, "{Status: %s, Calls: %d", ed.Status, ed.Calls)
		if len(ed.Errors) > 0 {
			fmt.Fprintf(&builder, ", Errors: %d", len(ed.Errors))
			// @TODO: add errors output
			//if len(ed.Errors) == 1 {
			//	builder.WriteString(fmt.Sprintf(" [%v]", ed.Errors[0]))
			//}
		}

	}

	builder.WriteString("}")
	return builder.String()
}

// ExecutionTrack holds execution details for a single operation.
type ExecutionTrack struct {
	calls  uint32
	errors []error
	status ExecutionStatus

	tracker Tracker
}

// NewExecutionTrack creates a new ExecutionTrack.
func NewExecutionTrack(tracker Tracker) *ExecutionTrack {
	return newExecutionTrack(tracker, ExecutionStatusUncalled)
}

func newExecutionTrack(tracker Tracker, initialStatus ExecutionStatus) *ExecutionTrack {
	return &ExecutionTrack{
		status:  initialStatus,
		tracker: tracker,
	}
}

// Calls returns the number of times this operation has been invoked.
func (ed *ExecutionTrack) Calls() uint32 {
	return ed.calls
}

// Errors returns the collected execution errors.
func (ed *ExecutionTrack) Errors() []error {
	return ed.errors
}

// Apply applies an act to the execution track.
// Failure acts set the status to failed and append a non-nil error.
func (ed *ExecutionTrack) Apply(act Act) {
	if ed == nil {
		return
	}

	switch act.Type {
	case ActCalled:
		ed.calls++
	case ActSucceeded:
		ed.status = ExecutionStatusSuccess
	case ActFailed:
		ed.status = ExecutionStatusFail
		if act.Err != nil {
			ed.errors = append(ed.errors, act.Err)
		}
	}
}

// GetStepData returns the StepData from the associated tracker.
func (ed *ExecutionTrack) GetStepData() StepData {
	return ed.tracker.GetStepData()
}

// GetTrackData returns a copy of the current TrackData.
func (ed *ExecutionTrack) GetTrackData() TrackData {
	return TrackData{
		Calls:  ed.calls,
		Errors: slices.Clone(ed.errors),
		Status: ed.status,
	}
}

type ExecutionRetryTrack struct {
	Track
	retryNumber uint32
}

func newExecutionRetryTrack(track Track, retry uint32) *ExecutionRetryTrack {
	return &ExecutionRetryTrack{
		Track:       track,
		retryNumber: retry,
	}
}

func (ed *ExecutionRetryTrack) Apply(act Act) {
	if ed == nil || ed.Track == nil {
		return
	}

	if act.Type == ActFailed && act.Err != nil {
		act.Err = fmt.Errorf("retry [%d]: %w", ed.retryNumber, act.Err)
	}
	ed.Track.Apply(act)
}

// simpleTracker manages the execution state for a single saga step.
type simpleTracker struct {
	stepName     string
	stepPosition uint32

	action       Track
	compensation Track

	compensationFunc     OperationFunc
	compensationRequired bool
}

// newInMemoryTrack creates a new simpleTracker for a given step.
func newInMemoryTrack(position uint32, step Step) *simpleTracker {
	tracker := &simpleTracker{
		stepName:             step.name,
		stepPosition:         position,
		compensationFunc:     step.compensation.fn,
		compensationRequired: step.compensationRequired,
	}

	actionStatus := ExecutionStatusUncalled
	compensationStatus := ExecutionStatusUncalled
	if step.compensation.fn == nil {
		compensationStatus = ExecutionStatusUnset
	}
	if step.action.fn == nil {
		actionStatus = ExecutionStatusUnset
	}

	tracker.action = newExecutionTrack(tracker, actionStatus)
	tracker.compensation = newExecutionTrack(tracker, compensationStatus)

	return tracker
}

// GetStepData returns a snapshot of the current step state.
func (t *simpleTracker) GetStepData() StepData {
	return StepData{
		StepName:             t.stepName,
		StepPosition:         t.stepPosition,
		Action:               t.action.GetTrackData(),
		Compensation:         t.compensation.GetTrackData(),
		CompensationRequired: t.compensationRequired,
	}
}
