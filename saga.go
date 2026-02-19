package oniontx

import (
	"context"
	"errors"
	"fmt"
)

var (
	ErrSagaActionFailed        = fmt.Errorf("action failed")
	ErrSagaCompensationFailed  = fmt.Errorf("compensation failed")
	ErrSagaCompensationSuccess = fmt.Errorf("compensation executed")
)

// Step of the [Sage].
type Step struct {
	// Name of the step.
	Name string

	// Action is the main operation executed within a step's transaction.
	Action func(ctx context.Context) error

	// Compensation - a compensating action that undoes the Action (if possible).
	// Called upon failure in subsequent steps.
	// Invoked when a subsequent step fails.
	//
	// Parameters:
	//   - ctx: context for cancellation and deadlines (context that is passed through the action)
	//   - aroseErr: error from the previous action that needs compensation
	Compensation func(ctx context.Context, aroseErr error) error

	// CompensationOnFail needs to add the current compensation to the list of compensations.
	CompensationOnFail bool
}

// Sage coordinates a distributed transaction using the `Saga` pattern.
type Sage struct {
	steps []Step
}

// NewSaga creates a new [Sage] instance.
func NewSaga(steps []Step) *Sage {
	return &Sage{
		steps: steps,
	}
}

// Execute runs all Saga steps.
//
// If any step fails, compensating actions are triggered for all successfully completed steps.
func (s *Sage) Execute(ctx context.Context) error {
	var completedSteps []Step

	for i, step := range s.steps {
		if step.Action == nil {
			continue
		}

		if step.CompensationOnFail {
			completedSteps = append(completedSteps, step)
		}

		err := step.Action(ctx)
		if err != nil {
			err = fmt.Errorf("step failed [%d#%s]: %w", i, step.Name, err)
		}

		if err != nil {
			// Run compensation when error arise.
			return s.compensate(ctx, completedSteps, err)
		}

		if !step.CompensationOnFail {
			completedSteps = append(completedSteps, step)
		}
	}

	return nil
}

// compensate triggers compensating actions for all steps in reverse order
func (s *Sage) compensate(ctx context.Context, completedSteps []Step, originalErr error) error {
	var compensationErrors []error

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
	}

	if len(compensationErrors) > 0 {
		compensationErrors = append(compensationErrors, ErrSagaCompensationFailed)
		return errors.Join(
			fmt.Errorf("original error: %w,  compensation errors: %w", originalErr, errors.Join(compensationErrors...)),
		)
	}

	return errors.Join(originalErr, ErrSagaCompensationSuccess, ErrSagaActionFailed)
}
