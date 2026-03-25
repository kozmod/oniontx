package saga

import (
	"fmt"
	"strings"
	"sync"
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

// StepData contains the complete execution history for a single saga step.
// It includes information about both the main action and its compensation.
type StepData struct {
	StepPosition uint32
	StepName     string

	Action               ExecutionData
	Compensation         ExecutionData
	CompensationRequired bool
}

// String returns a human-readable representation of the StepData.
func (s StepData) String() string {
	return fmt.Sprintf("Step %d: %s | Action: %s | Compensation: %s",
		s.StepPosition,
		s.StepName,
		s.Action.String(),
		s.Compensation.String())
}

// ExecutionData holds execution details for a single operation (action or compensation).
type ExecutionData struct {
	Calls  uint32
	Errors []error
	Status ExecutionStatus
}

// String returns a compact representation of ExecutionData.
func (t ExecutionData) String() string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("{Status: %s, Calls: %d", t.Status, t.Calls))
	if len(t.Errors) > 0 {
		builder.WriteString(fmt.Sprintf(", Errors: %d", len(t.Errors)))
		// @TODO: add errors output
		//if len(t.Errors) == 1 {
		//	builder.WriteString(fmt.Sprintf(" [%v]", t.Errors[0]))
		//}
	}

	builder.WriteString("}")
	return builder.String()
}

// Clone creates a deep copy of ExecutionData.
func (t ExecutionData) Clone() ExecutionData {
	errors := make([]error, len(t.Errors))
	copy(errors, t.Errors)
	return ExecutionData{
		Calls:  t.Calls,
		Errors: errors,
		Status: t.Status,
	}
}

// inMemoryTrack manages the execution state for a single saga step.
// It implements the Track interface and provides thread-safe state management.
type inMemoryTrack struct {
	StepName     string
	StepPosition uint32

	mx *sync.RWMutex

	action       *ExecutionData
	compensation *ExecutionData
	current      *ExecutionData

	CompensationRequired bool
	compensationFn       CompensationFunc
	parentErr            error
}

// newInMemoryTrack creates a new inMemoryTrack for a given step.
func newInMemoryTrack(position uint32, step Step) *inMemoryTrack {
	track := inMemoryTrack{
		StepName:     step.Name,
		StepPosition: position,
		mx:           new(sync.RWMutex),
		action: &ExecutionData{
			Status: ExecutionStatusUncalled,
		},
		compensation: &ExecutionData{
			Status: ExecutionStatusUncalled,
		},
		CompensationRequired: step.CompensationRequired,
		compensationFn:       step.Compensation,
	}

	if step.Compensation == nil {
		track.compensation.Status = ExecutionStatusUnset
	}

	if step.Action == nil {
		track.action.Status = ExecutionStatusUnset
	}

	return &track
}

// actionTrack switches the current execution context to the action.
// Returns the track for method chaining.
func (t *inMemoryTrack) actionTrack() *inMemoryTrack {
	t.mx.Lock()
	defer t.mx.Unlock()
	t.current = t.action
	return t
}

// compensationTrack switches the current execution context to the compensation.
// Returns the track for method chaining.
func (t *inMemoryTrack) compensationTrack() *inMemoryTrack {
	t.mx.Lock()
	defer t.mx.Unlock()
	t.current = t.compensation
	return t
}

// call increments the call counter for the current execution context.
// Should be called before each invocation of the operation.
func (t *inMemoryTrack) call() {
	t.mx.Lock()
	defer t.mx.Unlock()
	t.current.Calls = t.current.Calls + 1
}

// setParentError sets a parent error that will be wrapped with subsequent errors.
// Used to provide context about which retry attempt or operation triggered an error.
func (t *inMemoryTrack) setParentError(err error) {
	t.mx.Lock()
	defer t.mx.Unlock()
	t.parentErr = err
}

// SetStatus sets the status of the current execution context.
func (t *inMemoryTrack) SetStatus(status ExecutionStatus) {
	t.mx.Lock()
	defer t.mx.Unlock()
	t.current.Status = status
}

// getStatus returns the status of the current execution context.
func (t *inMemoryTrack) getStatus() ExecutionStatus {
	t.mx.RLock()
	defer t.mx.RUnlock()
	return t.current.Status
}

// SetFailedOnError marks the current execution as failed and records the error.
// If a parent error exists, it will be wrapped with the new error.
func (t *inMemoryTrack) SetFailedOnError(err error) {
	t.mx.Lock()
	defer t.mx.Unlock()
	if err == nil {
		return
	}
	if t.parentErr != nil {
		err = fmt.Errorf("%w: %w", t.parentErr, err)
	}
	t.current.Status = ExecutionStatusFail
	t.current.Errors = append(t.current.Errors, err)
}

// GetData returns a snapshot of the current execution state for this step.
func (t *inMemoryTrack) GetData() StepData {
	t.mx.RLock()
	defer t.mx.RUnlock()
	return StepData{
		StepName:             t.StepName,
		StepPosition:         t.StepPosition,
		Action:               t.action.Clone(),
		Compensation:         t.compensation.Clone(),
		CompensationRequired: t.CompensationRequired,
	}
}
