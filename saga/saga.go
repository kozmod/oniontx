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

	// ErrCompensationSuccess indicates that a compensation was executed successfully.
	// This error can be used to signal that compensation logic has been applied,
	// which might be useful for logging or monitoring purposes.
	// Note: Despite being an error type, it represents a successful compensation
	// execution, not a failure.
	ErrCompensationSuccess = fmt.Errorf("compensation executed")

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
func (s *Saga) Execute(ctx context.Context) error {
	var completedSteps []Step

	for i, step := range s.steps {
		select {
		case <-ctx.Done():
			return errors.Join(ErrExecuteActionsContextDone, ctx.Err())
		default:
			if step.Action == nil {
				continue
			}

			if step.CompensationOnFail {
				completedSteps = append(completedSteps, step)
			}

			err := step.Action(ctx)
			if err != nil {
				err = fmt.Errorf("action failed [%d#%s]: %w", i, step.Name, errors.Join(ErrActionFailed, err))
				// Run compensation when error arise.
				return s.compensate(ctx, completedSteps, err)
			}

			if !step.CompensationOnFail {
				completedSteps = append(completedSteps, step)
			}
		}
	}

	return nil
}

// compensate triggers compensating actions for all steps in reverse order.
func (s *Saga) compensate(ctx context.Context, completedSteps []Step, originalErr error) error {
	var (
		compensationErrors    []error
		compensationsExecuted int32
	)

stop:
	for i, step := range completedSteps {
		select {
		case <-ctx.Done():
			compensationErrors = append(compensationErrors, errors.Join(ErrExecuteCompensationContextDone, ctx.Err()))
			break stop
		default:
			if step.Compensation == nil {
				continue
			}

			if err := step.Compensation(ctx, originalErr); err != nil {
				compensationErrors = append(
					compensationErrors,
					fmt.Errorf("compensation failed [%d#%s]: %w", i, step.Name, err),
				)
			}
			compensationsExecuted++
		}
	}

	var err error
	if len(compensationErrors) > 0 {
		err = errors.Join(errors.Join(compensationErrors...), originalErr)
	}

	if err != nil {
		return errors.Join(ErrCompensationFailed, err)

	}

	return errors.Join(ErrCompensationSuccess, originalErr)
}
