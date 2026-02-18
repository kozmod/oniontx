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

	// Transactor - a local transactor for a specific storage.
	// It should implement an interface similar to [oniontx.Transactor].
	// This [Transactor] is optional and uses as others Transactors wrapper.
	Transactor interface {
		WithinTx(ctx context.Context, fn func(ctx context.Context) error) (err error)
	}

	// Action is the main operation executed within a step's transaction.
	Action func(ctx context.Context) error

	// Compensation - a compensating action that undoes the Action (if possible).
	// Called upon failure in subsequent steps
	Compensation func(ctx context.Context) error
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
		var err error
		switch {
		case step.Transactor == nil:
			err = step.Action(ctx)
			if err != nil {
				err = fmt.Errorf("step failed [%d#%s]: %w", i, step.Name, err)
			}
		default:
			err = step.Transactor.WithinTx(ctx, func(txCtx context.Context) error {
				err = step.Action(txCtx)
				if err != nil {
					return fmt.Errorf("step failed [%d#%s]: %w", i, step.Name, err)
				}
				return nil
			})
		}

		if err != nil {
			// Run compensation when error arise.
			return s.compensate(ctx, completedSteps, err)
		}

		completedSteps = append(completedSteps, step)
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

		if err := step.Compensation(ctx); err != nil {
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
