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

	ErrExecuteActionsContextDone = fmt.Errorf("execute actions context done")
	ErrRetryContextDone          = fmt.Errorf("retry context done")
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
			return errors.Join(ctx.Err(), ErrExecuteActionsContextDone)
		default:
			if step.Action == nil {
				continue
			}

			if step.CompensationOnFail {
				completedSteps = append(completedSteps, step)
			}

			err := step.Action(ctx)
			if err != nil {
				err = errors.Join(fmt.Errorf("step failed [%d#%s]", i, step.Name), err)
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

// compensate triggers compensating actions for all steps in reverse order
func (s *Saga) compensate(ctx context.Context, completedSteps []Step, originalErr error) error {
	var (
		compensationErrors    []error
		compensationsExecuted int32
	)

	for i, step := range completedSteps {
		if step.Compensation == nil {
			continue
		}

		if err := step.Compensation(ctx, originalErr); err != nil {
			compensationErrors = append(
				compensationErrors,
				fmt.Errorf("compensation failed - step [%d#%s]: %w", i, step.Name, err),
			)
		}
		compensationsExecuted++
	}

	if len(compensationErrors) > 0 {
		compensationErrors = append(compensationErrors, ErrCompensationFailed)
		return errors.Join(
			fmt.Errorf("original error: %w,  compensation errors: %w", originalErr, errors.Join(compensationErrors...)),
		)
	}

	if compensationsExecuted <= 0 {
		return errors.Join(originalErr, ErrActionFailed)
	}

	return errors.Join(originalErr, ErrCompensationSuccess, ErrActionFailed)
}
