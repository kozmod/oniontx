package saga

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/kozmod/oniontx/internal/testtool"
)

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
			ctx, cancel = context.WithCancel(context.Background())
			executed    []string
			actionCalls = 1
		)

		steps := []Step{
			NewStep("step0").
				WithAction(nil),
			NewStep("step1").
				WithAction(
					NewAction(func(ctx context.Context) error {
						executed = append(executed, "action1")
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
		testtool.AssertTrue(t, slices.Equal([]string{"action1", "action1", "action1"}, executed))

		testtool.LogError(t, err)
	})

	t.Run("compensation_ctx_cancel", func(t *testing.T) {
		var (
			ctx, cancel = context.WithCancel(context.Background())
			executed    []string
		)

		steps := []Step{
			NewStep("step1").
				WithAction(
					NewAction(func(ctx context.Context) error {
						executed = append(executed, "action1")
						cancel() // cancel context for test
						return testtool.ErrExpTest
					}),
				).WithCompensation(
				NewCompensation(func(ctx context.Context, aroseErr error) error {
					t.Fatalf("should not have been called")
					return nil
				}),
			).WithCompensationOnFail(),
		}
		err := NewSaga(steps).Execute(ctx)
		testtool.AssertError(t, err)
		testtool.AssertTrue(t, errors.Is(err, ErrExecuteCompensationContextDone))
		testtool.AssertTrue(t, slices.Equal([]string{"action1"}, executed))

		testtool.LogError(t, err)
	})
}

// nolint: dupl
func Test_hooks(t *testing.T) {
	t.Run("action_hooks", func(t *testing.T) {
		t.Run("before", func(t *testing.T) {
			var (
				ctx      = context.Background()
				executed []string
			)

			steps := []Step{
				NewStep("step1").
					WithAction(
						NewAction(func(ctx context.Context) error {
							executed = append(executed, "action1")
							return testtool.ErrExpTest
						}).WithBeforeHook(func(ctx context.Context) error {
							executed = append(executed, "hook1")
							return nil
						}).WithBeforeHook(func(ctx context.Context) error {
							executed = append(executed, "hook2")
							return nil
						}),
					),
			}
			err := NewSaga(steps).Execute(ctx)
			testtool.AssertError(t, err)
			testtool.AssertTrue(t, slices.Equal([]string{"hook2", "hook1", "action1"}, executed))

			testtool.LogError(t, err)
		})
		t.Run("before_with_retry", func(t *testing.T) {
			var (
				ctx      = context.Background()
				executed []string
			)

			steps := []Step{
				NewStep("step1").
					WithAction(
						NewAction(func(ctx context.Context) error {
							executed = append(executed, "action1")
							return testtool.ErrExpTest
						}).WithBeforeHook(func(ctx context.Context) error {
							executed = append(executed, "hook1")
							return nil
						}).WithBeforeHook(func(ctx context.Context) error {
							executed = append(executed, "hook2")
							return nil
						}).WithRetry(NewBaseRetryOpt(1, 1*time.Nanosecond)).
							WithBeforeHook(func(ctx context.Context) error {
								executed = append(executed, "retry_hook1")
								return nil
							}).WithBeforeHook(
							func(ctx context.Context) error {
								executed = append(executed, "retry_hook2")
								return nil
							}),
					),
			}
			err := NewSaga(steps).Execute(ctx)
			testtool.AssertError(t, err)
			testtool.AssertTrue(t, errors.Is(err, testtool.ErrExpTest))
			testtool.AssertTrue(t,
				slices.Equal(
					[]string{
						"retry_hook2", "retry_hook1", // retry hooks
						"hook2", "hook1", "action1", // call
						"hook2", "hook1", "action1", // first retry
					},
					executed),
			)

			testtool.LogError(t, err)
		})
		t.Run("after", func(t *testing.T) {
			var (
				ctx      = context.Background()
				executed []string
			)

			steps := []Step{
				NewStep("step1").
					WithAction(
						NewAction(func(ctx context.Context) error {
							executed = append(executed, "action1")
							return testtool.ErrExpTest
						}).WithAfterHook(func(ctx context.Context, aroseError error) error {
							testtool.AssertError(t, aroseError)
							testtool.AssertTrue(t, errors.Is(aroseError, testtool.ErrExpTest))
							executed = append(executed, "hook1")
							return aroseError
						}).WithAfterHook(func(ctx context.Context, aroseError error) error {
							testtool.AssertError(t, aroseError)
							testtool.AssertTrue(t, errors.Is(aroseError, testtool.ErrExpTest))
							executed = append(executed, "hook2")
							return aroseError
						}),
					),
			}
			err := NewSaga(steps).Execute(ctx)
			testtool.AssertError(t, err)
			testtool.AssertTrue(t, errors.Is(err, testtool.ErrExpTest))
			testtool.AssertTrue(t, slices.Equal([]string{"action1", "hook1", "hook2"}, executed))

			testtool.LogError(t, err)
		})
		t.Run("after_with_retry", func(t *testing.T) {
			var (
				ctx      = context.Background()
				executed []string
			)

			steps := []Step{
				NewStep("step1").
					WithAction(
						NewAction(func(ctx context.Context) error {
							executed = append(executed, "action1")
							return testtool.ErrExpTest
						}).WithAfterHook(func(ctx context.Context, aroseError error) error {
							testtool.AssertError(t, aroseError)
							testtool.AssertTrue(t, errors.Is(aroseError, testtool.ErrExpTest))
							executed = append(executed, "hook1")
							return aroseError
						}).WithAfterHook(func(ctx context.Context, aroseError error) error {
							testtool.AssertError(t, aroseError)
							testtool.AssertTrue(t, errors.Is(aroseError, testtool.ErrExpTest))
							executed = append(executed, "hook2")
							return aroseError
						}).WithRetry(NewBaseRetryOpt(2, 1*time.Nanosecond)).
							WithAfterHook(func(ctx context.Context, aroseError error) error {
								testtool.AssertTrue(t, errors.Is(aroseError, testtool.ErrExpTest))
								executed = append(executed, "retry_hook1")
								return aroseError
							}).
							WithAfterHook(func(ctx context.Context, aroseError error) error {
								testtool.AssertTrue(t, errors.Is(aroseError, testtool.ErrExpTest))
								executed = append(executed, "retry_hook2")
								return aroseError
							}),
					),
			}
			err := NewSaga(steps).Execute(ctx)
			testtool.AssertError(t, err)
			testtool.AssertTrue(t,
				slices.Equal(
					[]string{
						"action1", "hook1", "hook2", // call
						"action1", "hook1", "hook2", // first retry
						"action1", "hook1", "hook2", // second retry
						"retry_hook1", "retry_hook2", // retry hooks
					},
					executed),
			)

			testtool.LogError(t, err)
		})
	})
	t.Run("compensation_hooks", func(t *testing.T) {
		t.Run("before", func(t *testing.T) {
			var (
				ctx      = context.Background()
				executed []string
			)

			steps := []Step{
				NewStep("step1").
					WithAction(
						NewAction(func(ctx context.Context) error {
							executed = append(executed, "action1")
							return testtool.ErrExpTest
						}),
					).WithCompensation(
					NewCompensation(func(ctx context.Context, actionErr error) error {
						executed = append(executed, "comp1")
						return nil
					}).WithBeforeHook(func(ctx context.Context, actionErr error) error {
						testtool.AssertError(t, actionErr)
						testtool.AssertTrue(t, errors.Is(actionErr, testtool.ErrExpTest))
						executed = append(executed, "comp_hook1")
						return nil
					}).WithBeforeHook(func(ctx context.Context, actionErr error) error {
						testtool.AssertError(t, actionErr)
						testtool.AssertTrue(t, errors.Is(actionErr, testtool.ErrExpTest))
						executed = append(executed, "comp_hook2")
						return nil
					}),
				).WithCompensationOnFail(),
			}
			err := NewSaga(steps).Execute(ctx)
			testtool.AssertError(t, err)
			testtool.AssertTrue(t, errors.Is(err, testtool.ErrExpTest))
			testtool.AssertTrue(t, errors.Is(err, ErrCompensationSuccess))
			testtool.AssertTrue(t, slices.Equal([]string{"action1", "comp_hook2", "comp_hook1", "comp1"}, executed))

			testtool.LogError(t, err)
		})
		t.Run("after", func(t *testing.T) {
			var (
				ctx             = context.Background()
				previousHookErr = fmt.Errorf("previous_hook_err_1")
				compErr         = fmt.Errorf("comp_error_1")
				executed        []string
			)

			steps := []Step{
				NewStep("step1").
					WithAction(
						NewAction(func(ctx context.Context) error {
							executed = append(executed, "action1")
							return testtool.ErrExpTest
						}),
					).WithCompensation(
					NewCompensation(func(ctx context.Context, actionErr error) error {
						executed = append(executed, "comp1")
						return compErr
					}).WithAfterHook(func(ctx context.Context, actionErr, previousErr error) error {
						testtool.AssertError(t, actionErr)
						testtool.AssertTrue(t, errors.Is(actionErr, testtool.ErrExpTest))
						testtool.AssertTrue(t, errors.Is(previousErr, compErr))
						executed = append(executed, "comp_hook1")
						return previousHookErr
					}).WithAfterHook(func(ctx context.Context, actionErr, previousErr error) error {
						testtool.AssertError(t, actionErr)
						testtool.AssertTrue(t, errors.Is(actionErr, testtool.ErrExpTest))
						testtool.AssertTrue(t, errors.Is(previousErr, previousHookErr))
						executed = append(executed, "comp_hook2")
						return nil
					}),
				).WithCompensationOnFail(),
			}

			err := NewSaga(steps).Execute(ctx)
			testtool.AssertError(t, err)
			testtool.AssertTrue(t, errors.Is(err, testtool.ErrExpTest))
			testtool.AssertTrue(t, errors.Is(err, ErrCompensationSuccess))
			testtool.AssertTrue(t, slices.Equal([]string{"action1", "comp1", "comp_hook1", "comp_hook2"}, executed))

			testtool.LogError(t, err)
		})
	})
}
