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
		tracks         []*inMemoryTracker
		completedTrack []*inMemoryTracker
	)

stop:
	for i, step := range s.steps {
		var (
			tr = newInMemoryTrack(
				uint32(i),
				step,
				func(tracker Tracker) Track {
					return NewExecutionTrack(tracker)
				},
			)
		)

		tracks = append(tracks, tr)
		select {
		case <-ctx.Done():
			tr.action.SetStatus(ExecutionStatusFail)
			tr.action.AddError(
				fmt.Errorf("action failed [%d#%s]: %w", i, tr.stepName,
					errors.Join(ctx.Err(), ErrExecuteActionsContextDone),
				),
			)
			break stop
		default:
			if step.Action == nil {
				tr.action.SetStatus(ExecutionStatusUnset)
				continue
			}

			if step.CompensationRequired {
				completedTrack = append(completedTrack, tr)
			}

			tr.action.Call()
			err := step.Action(ctx, tr.action)

			switch status := tr.action.GetTrackData().Status; {
			case err == nil && status != ExecutionStatusFail:
				tr.action.SetStatus(ExecutionStatusSuccess)
			case err != nil || status == ExecutionStatusFail:
				if err != nil {
					tr.action.SetStatus(ExecutionStatusFail)
					err = errors.Join(err, ErrActionFailed)
					tr.action.AddError(
						fmt.Errorf("action failed [%d#%s]: %w", i, tr.stepName, err),
					)

				}
				// Run compensation when error arise.
				s.compensate(ctx, completedTrack)
				break stop
			}

			if !step.CompensationRequired {
				completedTrack = append(completedTrack, tr)
			}
		}
	}

	result, err := prepareResult(tracks)
	return result, err
}

// compensate triggers compensating actions for all steps in reverse order.
func (s *Saga) compensate(ctx context.Context, tracks []*inMemoryTracker) {
stop:
	for i, tr := range tracks {
		if tr.compensationFunc == nil {
			continue
		}
		select {
		case <-ctx.Done():
			tr.compensation.SetStatus(ExecutionStatusFail)
			tr.compensation.AddError(
				fmt.Errorf("compensation failed [%d#%s]: %w", i, tr.stepName,
					errors.Join(ctx.Err(), ErrExecuteCompensationContextDone),
				),
			)

			break stop
		default:
			tr.compensation.Call()
			err := tr.compensationFunc(ctx, tr.compensation)
			switch {
			case err == nil:
				comp := tr.compensation.GetTrackData()
				// Determine final status based on error count vs calls
				if uint32(len(comp.Errors)) == comp.Calls {
					tr.compensation.SetStatus(ExecutionStatusFail)
				} else {
					tr.compensation.SetStatus(ExecutionStatusSuccess)
				}
			case err != nil:
				tr.compensation.SetStatus(ExecutionStatusFail)
				tr.compensation.AddError(fmt.Errorf("compensation failed [%d#%s]: %w", i, tr.stepName, err))
			}
		}
	}
}
