package saga

import (
	"context"
	"fmt"
	"slices"

	"github.com/kozmod/oniontx/internal/errors"
)

var (
	// ErrActionFailed indicates that an action execution has failed.
	// This error is typically returned when a business operation or step
	// in a workflow cannot be completed successfully.
	ErrActionFailed = fmt.Errorf("action failed")

	// ErrCompensationFailed indicates that a required compensation did not
	// complete successfully. This includes an operation error, cancellation
	// before invocation, or missing compensation for a failed action configured
	// with WithCompensationOnActionFailure.
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

	// ErrNilRetryPolicy indicates that WithRetry was called without a retry policy.
	ErrNilRetryPolicy = fmt.Errorf("retry policy is nil")

	// ErrNilRetryFunc indicates that WithRetry was called without a retry function.
	ErrNilRetryFunc = fmt.Errorf("retry function is nil")

	// ErrCompensationNotConfigured indicates that compensation was requested for
	// a failed action but no compensation operation was configured.
	ErrCompensationNotConfigured = fmt.Errorf("compensation is not configured")
)

// Saga coordinates a local compensating workflow using the saga pattern.
// It does not persist execution state or coordinate distributed participants.
type Saga struct {
	steps                      []Step
	compensationContextFactory CtxFactory
}

// NewSaga creates a new Saga instance.
//
// The steps slice is copied, so later changes to the caller's slice do not
// affect the Saga.
func NewSaga(steps []Step) Saga {
	return Saga{
		steps: slices.Clone(steps),
	}
}

// WithCompensationContext returns a copy of the Saga that derives a context
// for compensation operations. The receiver is not modified; the result must
// be used.
//
// The factory receives the action context, which may already be canceled.
// Use context.WithoutCancel when compensations must outlive action cancellation.
func (s Saga) WithCompensationContext(factory CtxFactory) Saga {
	s.compensationContextFactory = factory
	return s
}

// Clone returns a copy of the Saga.
func (s Saga) Clone() Saga {
	s.steps = slices.Clone(s.steps)
	return s
}

// Execute runs all Saga steps.
//
// The returned Result contains every configured step and the complete error history.
// A non-nil error wraps failure categories and the terminal causes of failed
// operations or interrupted execution. Errors from successful retries remain
// available through Result.Errors and do not cause Execute to return an error.
func (s Saga) Execute(ctx context.Context) (Result, error) {
	var (
		track = sagaTrack{
			tracks: make([]*simpleTracker, len(s.steps)),
		}
		completedTrack []*simpleTracker
	)

	for i, step := range s.steps {
		track.tracks[i] = newInMemoryTrack(uint32(i), step)
	}

stop:
	for i, tr := range track.tracks {
		if tr.step.action.fn == nil {
			continue
		}
		select {
		case <-ctx.Done():
			track.err = fmt.Errorf(
				"execution failed [%d#%s]: %w", i, tr.step.name,
				errors.Join(ErrExecuteActionsContextDone, ctx.Err()),
			)
			track.compensating = true
			track.err = errors.Join(track.err, s.compensate(ctx, completedTrack))
			break stop
		default:
			if tr.step.compensateOnActionFailure {
				completedTrack = append(completedTrack, tr)
			}

			err := tr.step.action.fn(ctx, tr.action)

			switch status := tr.action.GetTrackData().Status; {
			case err != nil || status == ExecutionStatusFail:
				if err != nil {
					err = errors.Join(ErrActionFailed, err)
					tr.action.apply(
						newTrackFailedAct(
							fmt.Errorf("action failed [%d#%s]: %w", i, tr.step.name, err),
						),
					)
				}
				// Run compensation when an action error arises.
				track.compensating = true
				track.err = errors.Join(track.err, s.compensate(ctx, completedTrack))
				break stop
			default:
				if status != ExecutionStatusSuccess {
					tr.action.apply(newTrackSucceededAct())
				}
			}

			if !tr.step.compensateOnActionFailure {
				completedTrack = append(completedTrack, tr)
			}
		}
	}

	result, err := prepareResult(track)
	return result, err
}

// compensate triggers compensation operations in reverse completion order.
func (s Saga) compensate(ctx context.Context, tracks []*simpleTracker) error {
	compensationCtx := s.compensationContextFactory.Apply(ctx)

	for i := len(tracks) - 1; i >= 0; i-- {
		tr := tracks[i]
		position := tr.stepPosition
		if tr.step.compensation.fn == nil {
			if tr.step.compensateOnActionFailure && tr.action.GetTrackData().Status == ExecutionStatusFail {
				tr.compensation.apply(newTrackFailedAct(
					fmt.Errorf("compensation failed [%d#%s]: %w", position, tr.step.name, ErrCompensationNotConfigured),
				))
			}
			continue
		}
		select {
		case <-compensationCtx.Done():
			return fmt.Errorf("compensation interrupted [%d#%s]: %w", position, tr.step.name,
				errors.Join(ErrExecuteCompensationContextDone, compensationCtx.Err()))
		default:
			err := tr.step.compensation.fn(compensationCtx, tr.compensation)
			if err != nil {
				tr.compensation.apply(newTrackFailedAct(
					fmt.Errorf("compensation failed [%d#%s]: %w", position, tr.step.name, err),
				))
			} else if tr.compensation.GetTrackData().Status != ExecutionStatusSuccess {
				tr.compensation.apply(newTrackSucceededAct())
			}
		}
	}
	return nil
}
