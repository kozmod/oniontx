package sage

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/kozmod/oniontx/internal/tool"
)

func Test_Saga_example(t *testing.T) {
	t.Skip()

	steps := []Step{
		{
			Name: "first step",
			// Action — a function to execute
			Action: func(ctx context.Context) error {
				// Action logic.
				return nil
			},

			// Compensation — a function to compensate an action when an error occurs.
			//
			// Parameters:
			//   - ctx: context for cancellation and deadlines (context that is passed through the action)
			//   - aroseErr: error from the previous action that needs compensation
			Compensation: func(ctx context.Context, aroseErr error) error {
				// Action compensation logic.
				return nil
			},
			// CompensationOnFail needs to add the current compensation to the list of compensations.
			CompensationOnFail: true,
		},
	}
	// Saga execution.
	err := NewSaga(steps).Execute(context.Background())
	//nolint: staticcheck
	if err != nil {
		// Error handling.
	}
}

// nolint: dupl
func TestSaga_Execute(t *testing.T) {

	var (
		ctx = context.Background()

		expErr = fmt.Errorf("exp_sage_error")
	)

	t.Run("success_actions", func(t *testing.T) {
		var (
			executedActions      []string
			executedCompensation []string
		)

		steps := []Step{
			{
				Name: "step0",
				Action: func(ctx context.Context) error {
					executedActions = append(executedActions, "action1")
					return nil
				},
				Compensation: func(ctx context.Context, _ error) error {
					executedCompensation = append(executedCompensation, "comp1")
					t.Fatalf("should not have been called")
					return nil
				},
			},
			{
				Name: "step1",
				Action: func(ctx context.Context) error {
					executedActions = append(executedActions, "action2")
					return nil
				},
				Compensation: func(ctx context.Context, _ error) error {
					executedCompensation = append(executedCompensation, "comp2")
					t.Fatalf("should not have been called")
					return nil
				},
			},
		}

		err := NewSaga(steps).Execute(ctx)
		tool.AssertNoError(t, err)
		tool.AssertTrue(t, slices.Equal([]string{"action1", "action2"}, executedActions))
		tool.AssertTrue(t, len(executedCompensation) == 0)
	})

	t.Run("success_compensation_on_step1", func(t *testing.T) {
		var (
			executedActions      []string
			executedCompensation []string
		)

		steps := []Step{
			{
				Name: "step0",
				Action: func(ctx context.Context) error {
					executedActions = append(executedActions, "action1")
					return nil
				},
				Compensation: func(ctx context.Context, _ error) error {
					executedCompensation = append(executedCompensation, "comp1")
					return nil
				},
			},
			{
				Name: "step1",
				Action: func(ctx context.Context) error {
					executedActions = append(executedActions, "action2")
					return expErr
				},
				Compensation: func(ctx context.Context, aroseErr error) error {
					executedCompensation = append(executedCompensation, "comp2")
					t.Fatalf("should not have been called")
					return nil
				},
			},
		}

		err := NewSaga(steps).Execute(ctx)
		tool.AssertError(t, err)
		tool.AssertTrue(t, errors.Is(err, expErr))
		tool.AssertTrue(t, slices.Equal([]string{"action1", "action2"}, executedActions))
		tool.AssertTrue(t, slices.Equal([]string{"comp1"}, executedCompensation))
	})

	t.Run("compensation_on_fail", func(t *testing.T) {
		t.Run("skipped", func(t *testing.T) {
			var (
				executedActions      []string
				executedCompensation []string
			)

			steps := []Step{
				{
					Name: "step0",
					Action: func(ctx context.Context) error {
						executedActions = append(executedActions, "action1")
						return expErr
					},
					Compensation: func(ctx context.Context, aroseErr error) error {
						executedCompensation = append(executedCompensation, "comp1")
						t.Fatalf("should not have been called")
						return nil
					},
				},
			}

			err := NewSaga(steps).Execute(ctx)
			tool.AssertError(t, err)
			tool.AssertTrue(t, errors.Is(err, expErr))
			tool.AssertTrue(t, slices.Equal([]string{"action1"}, executedActions))
			tool.AssertTrue(t, len(executedCompensation) == 0)
		})
		t.Run("added", func(t *testing.T) {
			var (
				executedActions      []string
				executedCompensation []string
			)

			steps := []Step{
				{
					Name: "step0",
					Action: func(ctx context.Context) error {
						executedActions = append(executedActions, "action1")
						return expErr
					},
					Compensation: func(ctx context.Context, aroseErr error) error {
						executedCompensation = append(executedCompensation, "comp1")
						return nil
					},
					CompensationOnFail: true,
				},
			}

			err := NewSaga(steps).Execute(ctx)
			tool.AssertError(t, err)
			tool.AssertTrue(t, errors.Is(err, expErr))
			tool.AssertTrue(t, slices.Equal([]string{"action1"}, executedActions))
			tool.AssertTrue(t, slices.Equal([]string{"comp1"}, executedCompensation))
		})
	})
}
