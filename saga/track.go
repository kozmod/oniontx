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
	// Its state can be read by operations but is updated only by the Saga engine.
	Track interface {
		GetStepData() StepData
		GetTrackData() TrackData
	}

	mutableTrack interface {
		Track
		apply(trackAct)
	}
)

func applyTrackAct(track Track, act trackAct) {
	if mutable, ok := track.(mutableTrack); ok {
		mutable.apply(act)
	}
}

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

// executionTrack holds execution details for a single operation.
type executionTrack struct {
	calls  uint32
	errors []error
	status ExecutionStatus

	tracker Tracker
}

func newExecutionTrack(tracker Tracker, initialStatus ExecutionStatus) *executionTrack {
	return &executionTrack{
		status:  initialStatus,
		tracker: tracker,
	}
}

// apply updates the execution track from an internal state transition.
func (ed *executionTrack) apply(act trackAct) {
	if ed == nil {
		return
	}

	switch act.typeID {
	case trackActCalled:
		ed.calls++
	case trackActSucceeded:
		ed.status = ExecutionStatusSuccess
	case trackActFailed:
		ed.status = ExecutionStatusFail
		if act.err != nil {
			ed.errors = append(ed.errors, act.err)
		}
	}
}

// GetStepData returns the StepData from the associated tracker.
func (ed *executionTrack) GetStepData() StepData {
	return ed.tracker.GetStepData()
}

// GetTrackData returns a copy of the current TrackData.
func (ed *executionTrack) GetTrackData() TrackData {
	return TrackData{
		Calls:  ed.calls,
		Errors: slices.Clone(ed.errors),
		Status: ed.status,
	}
}

type executionRetryTrack struct {
	Track
	mutable     mutableTrack
	retryNumber uint32
}

func newExecutionRetryTrack(track Track, retry uint32) *executionRetryTrack {
	mutable, _ := track.(mutableTrack)
	return &executionRetryTrack{
		Track:       track,
		mutable:     mutable,
		retryNumber: retry,
	}
}

func (ed *executionRetryTrack) apply(act trackAct) {
	if ed == nil || ed.mutable == nil {
		return
	}

	if act.typeID == trackActFailed && act.err != nil {
		act.err = fmt.Errorf("retry [%d]: %w", ed.retryNumber, act.err)
	}
	ed.mutable.apply(act)
}

// simpleTracker manages the execution state for a single saga step.
type simpleTracker struct {
	stepName     string
	stepPosition uint32

	action       mutableTrack
	compensation mutableTrack

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
