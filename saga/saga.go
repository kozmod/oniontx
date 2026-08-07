package saga

import (
	"context"
	"fmt"

	"github.com/kozmod/oniontx/internal/errors"
)

var (
	// ErrActionFailed indicates that an action execution has failed.
	// This error is typically returned when a business operation or step
	// in a workflow cannot be completed successfully.
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

	// ErrExecuteCompensationContextDone indicates that the context was cancelled
	// or timed out during compensation execution.
	ErrExecuteCompensationContextDone = fmt.Errorf("execute compensation context done")

	// ErrRetryContextDone indicates that the context was cancelled or timed out
	// during retry attempts. This error is returned when a retry operation is
	// interrupted by context cancellation, meaning the operation was not completed
	// and no more retries will be attempted.
	ErrRetryContextDone = fmt.Errorf("retry context done")

	// ErrRetryFailed indicates that all retry attempts have been exhausted without success.
	// This error is returned when the maximum number of retry attempts configured in
	// the RetryPolicy has been reached and every attempt failed. The original errors
	// from each attempt are preserved in the Track's error list for debugging.
	ErrRetryFailed = fmt.Errorf("retry failed")

	// ErrCompensationRequired indicates that a step was marked as compensation
	// required but no compensation operation was configured.
	ErrCompensationRequired = fmt.Errorf("compensation required")
)

// Saga coordinates a local compensating workflow using the saga pattern.
// It does not persist execution state or coordinate distributed participants.
type Saga struct {
	steps                      []Step
	compensationContextFactory CtxFactory
}

// NewSaga creates a new Saga instance.
func NewSaga(steps []Step) *Saga {
	return &Saga{
		steps: steps,
	}
}

// WithCompensationContext configures a context for compensation operations.
//
// The factory receives the action context, which may already be canceled.
// Use context.WithoutCancel when compensations must outlive action cancellation.
func (s *Saga) WithCompensationContext(factory CtxFactory) *Saga {
	s.compensationContextFactory = factory
	return s
}

// Execute runs all Saga steps.
//
// If any step fails, compensations are triggered for completed steps in reverse order.
func (s *Saga) Execute(ctx context.Context) (Result, error) {
	var (
		tracks         []*simpleTracker
		completedTrack []*simpleTracker
	)

stop:
	for i, step := range s.steps {
		tr := newInMemoryTrack(uint32(i), step)

		tracks = append(tracks, tr)
		select {
		case <-ctx.Done():
			tr.action.apply(
				newTrackFailedAct(
					fmt.Errorf("action failed [%d#%s]: %w", i, tr.stepName,
						errors.Join(ErrExecuteActionsContextDone, ctx.Err()),
					),
				),
			)
			s.compensate(ctx, completedTrack)
			break stop
		default:
			if step.action.fn == nil {
				continue
			}

			if step.compensationRequired {
				completedTrack = append(completedTrack, tr)
			}

			err := step.action.fn(ctx, tr.action)

			switch status := tr.action.GetTrackData().Status; {
			case err != nil || status == ExecutionStatusFail:
				if err != nil {
					err = errors.Join(ErrActionFailed, err)
					tr.action.apply(
						newTrackFailedAct(
							fmt.Errorf("action failed [%d#%s]: %w", i, tr.stepName, err),
						),
					)
				}
				// Run compensation when an action error arises.
				s.compensate(ctx, completedTrack)
				break stop
			default:
				if status != ExecutionStatusSuccess {
					tr.action.apply(newTrackSucceededAct())
				}
			}

			if !step.compensationRequired {
				completedTrack = append(completedTrack, tr)
			}
		}
	}

	result, err := prepareResult(tracks)
	return result, err
}

// compensate triggers compensation operations in reverse completion order.
func (s *Saga) compensate(ctx context.Context, tracks []*simpleTracker) {
	ctx = s.compensationContextFactory.Apply(ctx)

stop:
	for i := len(tracks) - 1; i >= 0; i-- {
		tr := tracks[i]
		if tr.compensationFunc == nil {
			if tr.compensationRequired {
				tr.compensation.apply(newTrackFailedAct(
					fmt.Errorf("compensation failed [%d#%s]: %w", i, tr.stepName, ErrCompensationRequired),
				))
			}
			continue
		}
		select {
		case <-ctx.Done():
			tr.compensation.apply(newTrackFailedAct(
				fmt.Errorf("compensation failed [%d#%s]: %w", i, tr.stepName,
					errors.Join(ErrExecuteCompensationContextDone, ctx.Err()),
				),
			))

			break stop
		default:
			err := tr.compensationFunc(ctx, tr.compensation)
			if err != nil {
				tr.compensation.apply(newTrackFailedAct(
					fmt.Errorf("compensation failed [%d#%s]: %w", i, tr.stepName, err),
				))
			} else if tr.compensation.GetTrackData().Status != ExecutionStatusSuccess {
				tr.compensation.apply(newTrackSucceededAct())
			}
		}
	}
}
