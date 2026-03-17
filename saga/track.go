package saga

import (
	"fmt"
	"strings"
	"sync"
)

type ExecutionStatus string

const (
	ExecutionStatusUnknown ExecutionStatus = "Unknown"
	ExecutionStatusSuccess ExecutionStatus = "Success"
	ExecutionStatusFail    ExecutionStatus = "Fail"
)

type ExecutionTrack struct {
	Status ExecutionStatus
	Calls  uint32
	Errors []error
}

func (t ExecutionTrack) String() string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("{Status: %s, Calls: %d", t.Status, t.Calls))
	if len(t.Errors) > 0 {
		builder.WriteString(fmt.Sprintf(", Errors: %d", len(t.Errors)))
		if len(t.Errors) == 1 {
			builder.WriteString(fmt.Sprintf(" [%v]", t.Errors[0]))
		}
	}

	builder.WriteString("}")
	return builder.String()
}

type StepTrack struct {
	StepName     string
	StepPosition uint32

	Action       ExecutionTrack
	Compensation ExecutionTrack
}

func (s StepTrack) String() string {
	return fmt.Sprintf("Step %d: %s | Action: %s | Compensation: %s",
		s.StepPosition,
		s.StepName,
		s.Action.String(),
		s.Compensation.String())
}

type track struct {
	StepName   string
	StepNumber uint32

	mx *sync.RWMutex

	action       *ExecutionTrack
	compensation *ExecutionTrack
	current      *ExecutionTrack

	compensationFn CompensationFunc
}

func newTrack(stepName string, stepNumber uint32, comp CompensationFunc) *track {
	return &track{
		StepName:   stepName,
		StepNumber: stepNumber,
		mx:         new(sync.RWMutex),
		action: &ExecutionTrack{
			Status: ExecutionStatusUnknown,
		},
		compensation: &ExecutionTrack{
			Status: ExecutionStatusUnknown,
		},
		compensationFn: comp,
	}
}

func (t *track) actionTrack() *track {
	t.mx.Lock()
	defer t.mx.Unlock()
	t.current = t.action
	return t
}

func (t *track) compensationTrack() *track {
	t.mx.Lock()
	defer t.mx.Unlock()
	t.current = t.compensation
	return t
}

func (t *track) call() {
	t.mx.Lock()
	defer t.mx.Unlock()
	t.current.Calls = t.current.Calls + 1
}

func (t *track) setSuccess() {
	t.mx.Lock()
	defer t.mx.Unlock()
	t.current.Status = ExecutionStatusSuccess
}

func (t *track) setFailed() {
	t.mx.Lock()
	defer t.mx.Unlock()
	t.current.Status = ExecutionStatusFail
}

func (t *track) setFailedOnError(err error) {
	t.mx.Lock()
	defer t.mx.Unlock()
	if err != nil {
		t.current.Status = ExecutionStatusFail
		t.current.Errors = append(t.current.Errors, err)
	}
}

func (t *track) addError(err error) {
	t.mx.Lock()
	defer t.mx.Unlock()
	if err != nil {
		t.current.Errors = append(t.current.Errors, err)
	}
}

func (t *track) GetTrack() StepTrack {
	t.mx.RLock()
	defer t.mx.RUnlock()
	return StepTrack{
		StepName:     t.StepName,
		StepPosition: t.StepNumber,
		Action:       *t.action,
		Compensation: *t.compensation,
	}
}
