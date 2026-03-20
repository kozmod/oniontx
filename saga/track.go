package saga

import (
	"fmt"
	"strings"
	"sync"
)

type ExecutionStatus string

const (
	ExecutionStatusSuccess  ExecutionStatus = "Success"
	ExecutionStatusFail     ExecutionStatus = "Fail"
	ExecutionStatusUncalled ExecutionStatus = "Uncalled"
)

type StepData struct {
	StepPosition uint32
	StepName     string

	Action       ExecutionData
	Compensation ExecutionData
}

func (s StepData) String() string {
	return fmt.Sprintf("Step %d: %s | Action: %s | Compensation: %s",
		s.StepPosition,
		s.StepName,
		s.Action.String(),
		s.Compensation.String())
}

type ExecutionData struct {
	Calls  uint32
	Errors []error
	Status ExecutionStatus
}

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

func (t ExecutionData) Clone() ExecutionData {
	errors := make([]error, len(t.Errors))
	copy(errors, t.Errors)
	return ExecutionData{
		Calls:  t.Calls,
		Errors: errors,
		Status: t.Status,
	}
}

type executionTrack struct {
	StepName   string
	StepNumber uint32

	mx *sync.RWMutex

	action       *ExecutionData
	compensation *ExecutionData
	current      *ExecutionData

	compensationFn CompensationFunc
	parentErr      error
}

func newExecutionTrack(stepName string, stepNumber uint32, comp CompensationFunc) *executionTrack {
	return &executionTrack{
		StepName:   stepName,
		StepNumber: stepNumber,
		mx:         new(sync.RWMutex),
		action: &ExecutionData{
			Status: ExecutionStatusUncalled,
		},
		compensation: &ExecutionData{
			Status: ExecutionStatusUncalled,
		},
		compensationFn: comp,
	}
}

func (t *executionTrack) actionTrack() *executionTrack {
	t.mx.Lock()
	defer t.mx.Unlock()
	t.current = t.action
	return t
}

func (t *executionTrack) compensationTrack() *executionTrack {
	t.mx.Lock()
	defer t.mx.Unlock()
	t.current = t.compensation
	return t
}

func (t *executionTrack) call() {
	t.mx.Lock()
	defer t.mx.Unlock()
	t.current.Calls = t.current.Calls + 1
}

func (t *executionTrack) setParentError(err error) {
	t.mx.Lock()
	defer t.mx.Unlock()
	t.parentErr = err
}

func (t *executionTrack) SetStatus(status ExecutionStatus) {
	t.mx.Lock()
	defer t.mx.Unlock()
	t.current.Status = status
}

func (t *executionTrack) getStatus() ExecutionStatus {
	t.mx.RLock()
	defer t.mx.RUnlock()
	return t.current.Status
}

func (t *executionTrack) SetFailedOnError(err error) {
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

func (t *executionTrack) AddError(err error) {
	t.mx.Lock()
	defer t.mx.Unlock()
	if err == nil {
		return
	}
	if t.parentErr != nil {
		err = fmt.Errorf("%w: %w", t.parentErr, err)
	}

	t.current.Errors = append(t.current.Errors, err)
}

func (t *executionTrack) GetData() StepData {
	t.mx.RLock()
	defer t.mx.RUnlock()

	return StepData{
		StepName:     t.StepName,
		StepPosition: t.StepNumber,
		Action:       t.action.Clone(),
		Compensation: t.compensation.Clone(),
	}
}
