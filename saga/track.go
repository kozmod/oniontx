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
		Call()
		SetStatus(ExecutionStatus)
		SetParentError(error)
		AddError(error)

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
	// Compensation is the xecution data for the compensation operation.
	Compensation TrackData

	// CompensationRequired define when compensation should be triggered on failure.
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

// TrackData contains immutable execution metrics for a single operation.
type TrackData struct {
	Calls  uint32
	Errors []error
	Status ExecutionStatus
}

// String returns a compact representation of TrackData.
func (ed *TrackData) String() string {
	var builder strings.Builder
	switch {
	case ed == nil:
		builder.WriteString(fmt.Sprintf("{Status: %s, Calls: %d", "nil", -1))
	default:
		builder.WriteString(fmt.Sprintf("{Status: %s, Calls: %d", ed.Status, ed.Calls))
		if len(ed.Errors) > 0 {
			builder.WriteString(fmt.Sprintf(", Errors: %d", len(ed.Errors)))
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
	calls       uint32
	parentError error
	errors      []error
	status      ExecutionStatus

	tracker Tracker
}

// NewExecutionTrack creates a new ExecutionTrack.
func NewExecutionTrack(tracker Tracker) *ExecutionTrack {
	return &ExecutionTrack{
		status:  ExecutionStatusUncalled,
		tracker: tracker,
	}
}

// Calls returns the number of times this operation has been invoked.
func (ed *ExecutionTrack) Calls() uint32 {
	return ed.calls
}

// Errors returns the list of errors that occurred during execution.
func (ed *ExecutionTrack) Errors() []error {
	return ed.errors
}

// Call increments the call counter for this execution track.
func (ed *ExecutionTrack) Call() {
	ed.calls++
}

// SetStatus updates the execution status of this track.
func (ed *ExecutionTrack) SetStatus(status ExecutionStatus) {
	ed.status = status
}

// SetParentError sets a parent error that will be wrapped with any subsequent errors added.
func (ed *ExecutionTrack) SetParentError(err error) {
	ed.parentError = err
}

// AddError appends an error to the track's error list.
// If a parent error is set, it will be wrapped with the new error.
// Nil errors are silently ignored.
func (ed *ExecutionTrack) AddError(err error) {
	if err == nil || ed == nil {
		return
	}
	if ed.parentError != nil {
		err = fmt.Errorf("%w: %w", ed.parentError, err)
	}
	ed.errors = append(ed.errors, err)
}

// GetStepData returns the StepData from the associated tracker.
func (ed *ExecutionTrack) GetStepData() StepData {
	return ed.tracker.GetStepData()
}

// GetTrackData creates a deep copy of TrackData.
func (ed *ExecutionTrack) GetTrackData() TrackData {
	return TrackData{
		Calls:  ed.calls,
		Errors: slices.Clone(ed.errors),
		Status: ed.status,
	}
}

// inMemoryTracker manages the execution state for a single saga step.
type inMemoryTracker struct {
	stepName     string
	stepPosition uint32

	action       Track
	compensation Track

	compensationRequired bool
	parentErr            error

	compensationFunc CompensationFunc
}

// newInMemoryTrack creates a new inMemoryTracker for a given step.
func newInMemoryTrack(position uint32, step Step, trackFactory func(Tracker) Track) *inMemoryTracker {

	tracker := &inMemoryTracker{
		stepName:             step.Name,
		stepPosition:         position,
		compensationRequired: step.CompensationRequired,
		compensationFunc:     step.Compensation,
	}

	tracker.action = trackFactory(tracker)
	tracker.compensation = trackFactory(tracker)

	if step.Compensation == nil {
		tracker.compensation.SetStatus(ExecutionStatusUnset)
	}

	if step.Action == nil {
		tracker.action.SetStatus(ExecutionStatusUnset)
	}

	return tracker
}

// GetStepData returns a snapshot of the current step state.
func (t *inMemoryTracker) GetStepData() StepData {
	return StepData{
		StepName:             t.stepName,
		StepPosition:         t.stepPosition,
		Action:               t.action.GetTrackData(),
		Compensation:         t.compensation.GetTrackData(),
		CompensationRequired: t.compensationRequired,
	}
}
