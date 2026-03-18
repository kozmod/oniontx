package saga

import (
	"context"
	"errors"
	"fmt"
)

var (
	// ErrActionFailed indicates that an action execution has failed.
	// This error is typically returned when a business operation or step
	// in a workflow cannot be completed successfully
	ErrActionFailed = fmt.Errorf("action failed")

	// ErrCompensationFailed indicates that a compensation operation has failed.
	// This error is returned when trying to undo a previously
	// completed action, and the compensation logic itself encounters an error.
	ErrCompensationFailed = fmt.Errorf("compensation failed")

	// ErrPanicRecovered is returned when a panic is recovered and converted to an error.
	// It wraps the original panic value to provide more context about what caused
	// the panic. This allows panics to be handled gracefully without crashing
	// the application.
	ErrPanicRecovered = fmt.Errorf("panic recovered")

	// ErrExecuteActionsContextDone indicates that the context was cancelled or
	// timed out during the execution of saga actions. This error is returned
	// when the saga is interrupted before completing all steps, typically due to
	// client cancellation or deadline exceeded.
	ErrExecuteActionsContextDone = fmt.Errorf("execute actions context done")

	// ErrExecuteCompensationContextDone indicates that the context was cancelled or
	// timed out during the execution of saga action compensation. This error is returned
	// when the saga is interrupted before completing all steps, typically due to
	// client cancellation or deadline exceeded.
	ErrExecuteCompensationContextDone = fmt.Errorf("execute compensation context done")

	// ErrRetryContextDone indicates that the context was cancelled or timed out
	// during retry attempts. This error is returned when a retry operation is
	// interrupted by context cancellation, meaning the operation was not completed
	// and no more retries will be attempted.
	ErrRetryContextDone = fmt.Errorf("retry context done")
)

type Track interface {
	call()
	setStatus(ExecutionStatus)
	addError(error)
	setFailedOnError(err error)
	GetData() StepData
}

// Saga coordinates a distributed transaction using the `Saga` pattern.
type Saga struct {
	steps []Step
}

// NewSaga creates a new [Saga] instance.
func NewSaga(steps []Step) *Saga {
	return &Saga{
		steps: steps,
	}
}

// Execute runs all Saga steps.
//
// If any step fails, compensating actions are triggered for all successfully completed steps.
func (s *Saga) Execute(ctx context.Context) (Result, error) {
	var (
		tracks         []*executionTrack
		completedTrack []*executionTrack
		err            error
	)

stop:
	for i, step := range s.steps {
		var (
			tr = newExecutionTrack(
				step.Name,
				uint32(i),
				step.Compensation,
			)
		)

		tr.actionTrack()
		tracks = append(tracks, tr)
		select {
		case <-ctx.Done():
			tr.addError(errors.Join(ErrExecuteActionsContextDone, ctx.Err()))
			break stop
			//return Errors.Join(ErrExecuteActionsContextDone, ctx.Err())
		default:
			if step.Action == nil {
				continue
			}

			if step.CompensationOnFail {
				completedTrack = append(completedTrack, tr)
			}

			tr.call()
			err = step.Action(ctx, tr)

			switch status := tr.getStatus(); {
			case err == nil && status != ExecutionStatusFail:
				tr.setStatus(ExecutionStatusSuccess)
			case err != nil || status == ExecutionStatusFail:
				tr.setFailedOnError(err)
				err = fmt.Errorf("action failed [%d#%s]: %w", i, step.Name, errors.Join(ErrActionFailed, err))
				// Run compensation when error arise.
				s.compensate(ctx, completedTrack)
				break stop
			}

			if !step.CompensationOnFail {
				completedTrack = append(completedTrack, tr)
			}
		}
	}

	result, err := prepareResult(tracks)
	return result, err
}

// compensate triggers compensating actions for all steps in reverse order.
func (s *Saga) compensate(ctx context.Context, tracks []*executionTrack) {
stop:
	for i, tr := range tracks {
		if tr.compensationFn == nil {
			continue
		}
		tr.compensationTrack()
		select {
		case <-ctx.Done():
			tr.setFailedOnError(
				fmt.Errorf("compensation failed [%d#%s]: %w", i, tr.StepName,
					errors.Join(ErrExecuteCompensationContextDone, ctx.Err()),
				),
			)
			break stop
		default:
			tr.call()
			err := tr.compensationFn(ctx, tr)
			switch status := tr.getStatus(); {
			case err == nil && status != ExecutionStatusFail:
				tr.setStatus(ExecutionStatusSuccess)
			case err != nil:
				tr.setFailedOnError(fmt.Errorf("compensation failed [%d#%s]: %w", i, tr.StepName, err))
				continue
			}
		}
	}
}
