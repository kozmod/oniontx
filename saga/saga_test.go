package saga

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/kozmod/oniontx/internal/testtool"
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
		testtool.AssertNoError(t, err)
		testtool.AssertTrue(t, slices.Equal([]string{"action1", "action2"}, executedActions))
		testtool.AssertTrue(t, len(executedCompensation) == 0)
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
				Action: NewAction(func(ctx context.Context) error {
					executedActions = append(executedActions, "action2")
					return testtool.ErrExpTest
				}),
				Compensation: NewCompensation(func(ctx context.Context, aroseErr error) error {
					executedCompensation = append(executedCompensation, "comp2")
					t.Fatalf("should not have been called")
					return nil
				}),
			},
		}

		err := NewSaga(steps).Execute(ctx)
		testtool.AssertError(t, err)
		testtool.AssertTrue(t, errors.Is(err, testtool.ErrExpTest))
		testtool.AssertTrue(t, slices.Equal([]string{"action1", "action2"}, executedActions))
		testtool.AssertTrue(t, slices.Equal([]string{"comp1"}, executedCompensation))
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
						return testtool.ErrExpTest
					},
					Compensation: func(ctx context.Context, aroseErr error) error {
						executedCompensation = append(executedCompensation, "comp1")
						t.Fatalf("should not have been called")
						return nil
					},
				},
			}

			err := NewSaga(steps).Execute(ctx)
			testtool.AssertError(t, err)
			testtool.AssertTrue(t, errors.Is(err, testtool.ErrExpTest))
			testtool.AssertTrue(t, slices.Equal([]string{"action1"}, executedActions))
			testtool.AssertTrue(t, len(executedCompensation) == 0)
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
						return testtool.ErrExpTest
					},
					Compensation: func(ctx context.Context, aroseErr error) error {
						executedCompensation = append(executedCompensation, "comp1")
						return nil
					},
					CompensationOnFail: true,
				},
			}

			err := NewSaga(steps).Execute(ctx)
			testtool.AssertError(t, err)
			testtool.AssertTrue(t, errors.Is(err, testtool.ErrExpTest))
			testtool.AssertTrue(t, slices.Equal([]string{"action1"}, executedActions))
			testtool.AssertTrue(t, slices.Equal([]string{"comp1"}, executedCompensation))
		})
	})
}

func Test_Saga_panic_recovery(t *testing.T) {
	var (
		ctx = context.Background()
	)
	t.Run("static_func", func(t *testing.T) {
		t.Run("success_v1", func(t *testing.T) {
			steps := []Step{
				{
					Name: "step0",
					Action: WithPanicRecovery(func(ctx context.Context) error {
						panic("panic_v1!")
					}),
				},
			}

			err := NewSaga(steps).Execute(ctx)
			testtool.AssertError(t, err)
			testtool.AssertTrue(t, errors.Is(err, ErrPanicRecovered))
			testtool.AssertTrue(t, errors.Is(err, ErrActionFailed))

			testtool.LogError(t, err)
		})
	})
	t.Run("builders", func(t *testing.T) {
		t.Run("success_ActionFunc", func(t *testing.T) {
			steps := []Step{
				{
					Name: "step0",
					Action: ActionFunc(func(ctx context.Context) error {
						panic("panic_v2!")
					}).WithPanicRecovery(),
				},
			}

			err := NewSaga(steps).Execute(ctx)
			testtool.AssertError(t, err)
			testtool.AssertTrue(t, errors.Is(err, ErrPanicRecovered))
			testtool.AssertTrue(t, errors.Is(err, ErrActionFailed))

			testtool.LogError(t, err)
		})

		t.Run("success_CompensationFunc", func(t *testing.T) {
			steps := []Step{
				{
					Name: "step0",
					Action: ActionFunc(func(ctx context.Context) error {
						return testtool.ErrExpTest
					}),
					Compensation: CompensationFunc(func(ctx context.Context, aroseErr error) error {
						panic("panic_v3!")
					}).WithPanicRecovery(),
					CompensationOnFail: true,
				},
			}

			err := NewSaga(steps).Execute(ctx)
			testtool.AssertError(t, err)
			testtool.AssertTrue(t, errors.Is(err, ErrPanicRecovered))
			testtool.AssertTrue(t, errors.Is(err, ErrCompensationFailed))

			testtool.LogError(t, err)
		})
	})
}

func Test_actions_v2(t *testing.T) {
	var (
		ctx = context.Background()
	)

	t.Run("success_actions", func(t *testing.T) {
		var (
			executedActions      []string
			executedCompensation []string
		)

		steps := []Step{
			NewStep("step0").
				WithAction(func(ctx context.Context) error {
					executedActions = append(executedActions, "action1")
					return nil
				}).
				WithCompensation(func(ctx context.Context, aroseErr error) error {
					executedCompensation = append(executedCompensation, "comp1")
					t.Fatalf("should not have been called")
					return nil
				}),
			NewStep("step1").
				WithAction(func(ctx context.Context) error {
					executedActions = append(executedActions, "action2")
					return nil
				}).
				WithCompensation(func(ctx context.Context, aroseErr error) error {
					executedCompensation = append(executedCompensation, "comp2")
					t.Fatalf("should not have been called")
					return nil
				}),
		}

		err := NewSaga(steps).Execute(ctx)
		testtool.AssertNoError(t, err)
		testtool.AssertTrue(t, slices.Equal([]string{"action1", "action2"}, executedActions))
		testtool.AssertTrue(t, len(executedCompensation) == 0)
	})
}

func Test_execute_context(t *testing.T) {
	t.Run("action_ctx_cancel", func(t *testing.T) {
		var (
			ctx, cancel     = context.WithCancel(context.Background())
			executedActions []string
		)

		steps := []Step{
			NewStep("step0").
				WithAction(nil),
			NewStep("step1").
				WithAction(func(ctx context.Context) error {
					executedActions = append(executedActions, "action1")
					return nil
				}),
			NewStep("step2").
				WithAction(func(ctx context.Context) error {
					executedActions = append(executedActions, "action2")
					cancel() // cancel context for test
					return nil
				}),
			NewStep("step3").
				WithAction(func(ctx context.Context) error {
					executedActions = append(executedActions, "action3")
					t.Fatalf("should not have been called")
					return nil
				}),
		}

		err := NewSaga(steps).Execute(ctx)
		testtool.AssertError(t, err)
		testtool.AssertTrue(t, errors.Is(err, ErrExecuteActionsContextDone))
		testtool.AssertTrue(t, slices.Equal([]string{"action1", "action2"}, executedActions))
	})
	t.Run("retry_ctx_cancel", func(t *testing.T) {
		var (
			ctx, cancel     = context.WithCancel(context.Background())
			executedActions []string
			actionCalls     = 1
		)

		steps := []Step{
			NewStep("step0").
				WithAction(nil),
			NewStep("step1").
				WithAction(
					NewAction(func(ctx context.Context) error {
						executedActions = append(executedActions, "action1")
						switch {
						case actionCalls == 1:
							actionCalls++
							return testtool.ErrExpTest
						case actionCalls == 2:
							actionCalls++
							return testtool.ErrExpTest
						case actionCalls >= 3:
							actionCalls++
							cancel() // cancel context for test
							return testtool.ErrExpTest
						}
						return nil
					}).WithRetry(NewBaseRetryOpt(4, 1*time.Nanosecond)),
				),
		}
		err := NewSaga(steps).Execute(ctx)
		testtool.AssertError(t, err)
		testtool.AssertTrue(t, errors.Is(err, ErrRetryContextDone))
		testtool.AssertTrue(t, 4 == actionCalls) // 3 + first execution
		testtool.AssertTrue(t, slices.Equal([]string{"action1", "action1", "action1"}, executedActions))

		testtool.LogError(t, err)

	})

}
