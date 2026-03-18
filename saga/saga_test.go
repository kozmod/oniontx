package saga

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/kozmod/oniontx/internal/testtool"
	"github.com/kozmod/oniontx/internal/testtool/assert"
)

// import (
//
//	"context"
//	"Errors"
//	"fmt"
//	"slices"
//	"testing"
//	"time"
//
//	"github.com/kozmod/oniontx/internal/testtool"
//
// )
//
// // nolint: dupl
//
//	func TestSaga_Execute(t *testing.T) {
//		var (
//			ctx = context.Background()
//		)
//
//		t.Run("success_actions", func(t *testing.T) {
//			var (
//				executedActions      []string
//				executedCompensation []string
//			)
//
//			steps := []Step{
//				{
//					Name: "step0",
//					Action: func(ctx context.Context) error {
//						executedActions = append(executedActions, "action1")
//						return nil
//					},
//					Compensation: func(ctx context.Context, _ error) error {
//						executedCompensation = append(executedCompensation, "comp1")
//						t.Fatalf("should not have been called")
//						return nil
//					},
//				},
//				{
//					Name: "step1",
//					Action: func(ctx context.Context) error {
//						executedActions = append(executedActions, "action2")
//						return nil
//					},
//					Compensation: func(ctx context.Context, _ error) error {
//						executedCompensation = append(executedCompensation, "comp2")
//						t.Fatalf("should not have been called")
//						return nil
//					},
//				},
//			}
//
//			err := NewSaga(steps).Execute(ctx)
//			testtool.NoError(t, err)
//			testtool.True(t, slices.Equal([]string{"action1", "action2"}, executedActions))
//			testtool.True(t, len(executedCompensation) == 0)
//		})
//
//		t.Run("success_compensation_on_step1", func(t *testing.T) {
//			var (
//				executedActions      []string
//				executedCompensation []string
//			)
//
//			steps := []Step{
//				{
//					Name: "step0",
//					Action: func(ctx context.Context) error {
//						executedActions = append(executedActions, "action1")
//						return nil
//					},
//					Compensation: func(ctx context.Context, _ error) error {
//						executedCompensation = append(executedCompensation, "comp1")
//						return nil
//					},
//				},
//				{
//					Name: "step1",
//					Action: NewAction(func(ctx context.Context) error {
//						executedActions = append(executedActions, "action2")
//						return testtool.ErrExpTestA
//					}),
//					Compensation: NewCompensation(func(ctx context.Context, aroseErr error) error {
//						executedCompensation = append(executedCompensation, "comp2")
//						t.Fatalf("should not have been called")
//						return nil
//					}),
//				},
//			}
//
//			err := NewSaga(steps).Execute(ctx)
//			testtool.Error(t, err)
//			testtool.True(t, Errors.Is(err, testtool.ErrExpTestA))
//			testtool.True(t, slices.Equal([]string{"action1", "action2"}, executedActions))
//			testtool.True(t, slices.Equal([]string{"comp1"}, executedCompensation))
//		})
//
//		t.Run("compensation_on_fail", func(t *testing.T) {
//			t.Run("skipped", func(t *testing.T) {
//				var (
//					executedActions      []string
//					executedCompensation []string
//				)
//
//				steps := []Step{
//					{
//						Name: "step0",
//						Action: func(ctx context.Context) error {
//							executedActions = append(executedActions, "action1")
//							return testtool.ErrExpTestA
//						},
//						Compensation: func(ctx context.Context, aroseErr error) error {
//							executedCompensation = append(executedCompensation, "comp1")
//							t.Fatalf("should not have been called")
//							return nil
//						},
//					},
//				}
//
//				err := NewSaga(steps).Execute(ctx)
//				testtool.Error(t, err)
//				testtool.True(t, Errors.Is(err, testtool.ErrExpTestA))
//				testtool.True(t, slices.Equal([]string{"action1"}, executedActions))
//				testtool.True(t, len(executedCompensation) == 0)
//			})
//			t.Run("added", func(t *testing.T) {
//				var (
//					executedActions      []string
//					executedCompensation []string
//				)
//
//				steps := []Step{
//					{
//						Name: "step0",
//						Action: func(ctx context.Context) error {
//							executedActions = append(executedActions, "action1")
//							return testtool.ErrExpTestA
//						},
//						Compensation: func(ctx context.Context, aroseErr error) error {
//							executedCompensation = append(executedCompensation, "comp1")
//							return nil
//						},
//						CompensationOnFail: true,
//					},
//				}
//
//				err := NewSaga(steps).Execute(ctx)
//				testtool.Error(t, err)
//				testtool.True(t, Errors.Is(err, testtool.ErrExpTestA))
//				testtool.True(t, slices.Equal([]string{"action1"}, executedActions))
//				testtool.True(t, slices.Equal([]string{"comp1"}, executedCompensation))
//			})
//		})
//	}
//
//	func Test_Saga_panic_recovery(t *testing.T) {
//		var (
//			ctx = context.Background()
//		)
//		t.Run("static_func", func(t *testing.T) {
//			t.Run("success_v1", func(t *testing.T) {
//				steps := []Step{
//					{
//						Name: "step0",
//						Action: WithPanicRecovery(func(ctx context.Context) error {
//							panic("panic_v1!")
//						}),
//					},
//				}
//
//				err := NewSaga(steps).Execute(ctx)
//				testtool.Error(t, err)
//				testtool.True(t, Errors.Is(err, ErrPanicRecovered))
//				testtool.True(t, Errors.Is(err, ErrActionFailed))
//
//				testtool.LogError(t, err)
//			})
//		})
//		t.Run("builders", func(t *testing.T) {
//			t.Run("success_ActionFunc", func(t *testing.T) {
//				steps := []Step{
//					{
//						Name: "step0",
//						Action: ActionFunc(func(ctx context.Context) error {
//							panic("panic_v2!")
//						}).WithPanicRecovery(),
//					},
//				}
//
//				err := NewSaga(steps).Execute(ctx)
//				testtool.Error(t, err)
//				testtool.True(t, Errors.Is(err, ErrPanicRecovered))
//				testtool.True(t, Errors.Is(err, ErrActionFailed))
//
//				testtool.LogError(t, err)
//			})
//
//			t.Run("success_CompensationFunc", func(t *testing.T) {
//				steps := []Step{
//					{
//						Name: "step0",
//						Action: ActionFunc(func(ctx context.Context) error {
//							return testtool.ErrExpTestA
//						}),
//						Compensation: CompensationFunc(func(ctx context.Context, aroseErr error) error {
//							panic("panic_v3!")
//						}).WithPanicRecovery(),
//						CompensationOnFail: true,
//					},
//				}
//
//				err := NewSaga(steps).Execute(ctx)
//				testtool.Error(t, err)
//				testtool.True(t, Errors.Is(err, ErrPanicRecovered))
//				testtool.True(t, Errors.Is(err, ErrCompensationFailed))
//
//				testtool.LogError(t, err)
//			})
//		})
//	}
//
//	func Test_actions_v2(t *testing.T) {
//		var (
//			ctx = context.Background()
//		)
//
//		t.Run("success_actions", func(t *testing.T) {
//			var (
//				executedActions      []string
//				executedCompensation []string
//			)
//
//			steps := []Step{
//				NewStep("step0").
//					WithAction(func(ctx context.Context) error {
//						executedActions = append(executedActions, "action1")
//						return nil
//					}).
//					WithCompensation(func(ctx context.Context, aroseErr error) error {
//						executedCompensation = append(executedCompensation, "comp1")
//						t.Fatalf("should not have been called")
//						return nil
//					}),
//				NewStep("step1").
//					WithAction(func(ctx context.Context) error {
//						executedActions = append(executedActions, "action2")
//						return nil
//					}).
//					WithCompensation(func(ctx context.Context, aroseErr error) error {
//						executedCompensation = append(executedCompensation, "comp2")
//						t.Fatalf("should not have been called")
//						return nil
//					}),
//			}
//
//			err := NewSaga(steps).Execute(ctx)
//			testtool.NoError(t, err)
//			testtool.True(t, slices.Equal([]string{"action1", "action2"}, executedActions))
//			testtool.True(t, len(executedCompensation) == 0)
//		})
//	}
//
//	func Test_execute_context(t *testing.T) {
//		t.Run("action_ctx_cancel", func(t *testing.T) {
//			var (
//				ctx, cancel     = context.WithCancel(context.Background())
//				executedActions []string
//			)
//
//			steps := []Step{
//				NewStep("step0").
//					WithAction(nil),
//				NewStep("step1").
//					WithAction(func(ctx context.Context) error {
//						executedActions = append(executedActions, "action1")
//						return nil
//					}),
//				NewStep("step2").
//					WithAction(func(ctx context.Context) error {
//						executedActions = append(executedActions, "action2")
//						cancel() // cancel context for test
//						return nil
//					}),
//				NewStep("step3").
//					WithAction(func(ctx context.Context) error {
//						executedActions = append(executedActions, "action3")
//						t.Fatalf("should not have been called")
//						return nil
//					}),
//			}
//
//			err := NewSaga(steps).Execute(ctx)
//			testtool.Error(t, err)
//			testtool.True(t, Errors.Is(err, ErrExecuteActionsContextDone))
//			testtool.True(t, slices.Equal([]string{"action1", "action2"}, executedActions))
//		})
//		t.Run("retry_ctx_cancel", func(t *testing.T) {
//			var (
//				ctx, cancel = context.WithCancel(context.Background())
//				executed    []string
//				actionCalls = 1
//			)
//
//			steps := []Step{
//				NewStep("step0").
//					WithAction(nil),
//				NewStep("step1").
//					WithAction(
//						NewAction(func(ctx context.Context) error {
//							executed = append(executed, "action1")
//							switch {
//							case actionCalls == 1:
//								actionCalls++
//								return testtool.ErrExpTestA
//							case actionCalls == 2:
//								actionCalls++
//								return testtool.ErrExpTestA
//							case actionCalls >= 3:
//								actionCalls++
//								cancel() // cancel context for test
//								return testtool.ErrExpTestA
//							}
//							return nil
//						}).WithRetry(NewBaseRetryOpt(4, 1*time.Nanosecond)),
//					),
//			}
//			err := NewSaga(steps).Execute(ctx)
//			testtool.Error(t, err)
//			testtool.True(t, Errors.Is(err, ErrRetryContextDone))
//			testtool.True(t, 4 == actionCalls) // 3 + first execution
//			testtool.True(t, slices.Equal([]string{"action1", "action1", "action1"}, executed))
//
//			testtool.LogError(t, err)
//		})
//
//		t.Run("compensation_ctx_cancel", func(t *testing.T) {
//			var (
//				ctx, cancel = context.WithCancel(context.Background())
//				executed    []string
//			)
//
//			steps := []Step{
//				NewStep("step1").
//					WithAction(
//						NewAction(func(ctx context.Context) error {
//							executed = append(executed, "action1")
//							cancel() // cancel context for test
//							return testtool.ErrExpTestA
//						}),
//					).WithCompensation(
//					NewCompensation(func(ctx context.Context, aroseErr error) error {
//						t.Fatalf("should not have been called")
//						return nil
//					}),
//				).WithCompensationOnFail(),
//			}
//			err := NewSaga(steps).Execute(ctx)
//			testtool.Error(t, err)
//			testtool.True(t, Errors.Is(err, ErrExecuteCompensationContextDone))
//			testtool.True(t, slices.Equal([]string{"action1"}, executed))
//
//			testtool.LogError(t, err)
//		})
//	}
//
// // nolint: dupl
//
//	func Test_hooks(t *testing.T) {
//		t.Run("action_hooks", func(t *testing.T) {
//			t.Run("before", func(t *testing.T) {
//				var (
//					ctx      = context.Background()
//					executed []string
//				)
//
//				steps := []Step{
//					NewStep("step1").
//						WithAction(
//							NewAction(func(ctx context.Context) error {
//								executed = append(executed, "action1")
//								return testtool.ErrExpTestA
//							}).WithBeforeHook(func(ctx context.Context) error {
//								executed = append(executed, "hook1")
//								return nil
//							}).WithBeforeHook(func(ctx context.Context) error {
//								executed = append(executed, "hook2")
//								return nil
//							}),
//						),
//				}
//				err := NewSaga(steps).Execute(ctx)
//				testtool.Error(t, err)
//				testtool.True(t, slices.Equal([]string{"hook2", "hook1", "action1"}, executed))
//
//				testtool.LogError(t, err)
//			})
//			t.Run("before_with_retry", func(t *testing.T) {
//				var (
//					ctx      = context.Background()
//					executed []string
//				)
//
//				steps := []Step{
//					NewStep("step1").
//						WithAction(
//							NewAction(func(ctx context.Context) error {
//								executed = append(executed, "action1")
//								return testtool.ErrExpTestA
//							}).WithBeforeHook(func(ctx context.Context) error {
//								executed = append(executed, "hook1")
//								return nil
//							}).WithBeforeHook(func(ctx context.Context) error {
//								executed = append(executed, "hook2")
//								return nil
//							}).WithRetry(NewBaseRetryOpt(1, 1*time.Nanosecond)).
//								WithBeforeHook(func(ctx context.Context) error {
//									executed = append(executed, "retry_hook1")
//									return nil
//								}).WithBeforeHook(
//								func(ctx context.Context) error {
//									executed = append(executed, "retry_hook2")
//									return nil
//								}),
//						),
//				}
//				err := NewSaga(steps).Execute(ctx)
//				testtool.Error(t, err)
//				testtool.True(t, Errors.Is(err, testtool.ErrExpTestA))
//				testtool.True(t,
//					slices.Equal(
//						[]string{
//							"retry_hook2", "retry_hook1", // retry hooks
//							"hook2", "hook1", "action1", // call
//							"hook2", "hook1", "action1", // first retry
//						},
//						executed),
//				)
//
//				testtool.LogError(t, err)
//			})
//			t.Run("after", func(t *testing.T) {
//				var (
//					ctx      = context.Background()
//					executed []string
//				)
//
//				steps := []Step{
//					NewStep("step1").
//						WithAction(
//							NewAction(func(ctx context.Context) error {
//								executed = append(executed, "action1")
//								return testtool.ErrExpTestA
//							}).WithAfterHook(func(ctx context.Context, aroseError error) error {
//								testtool.Error(t, aroseError)
//								testtool.True(t, Errors.Is(aroseError, testtool.ErrExpTestA))
//								executed = append(executed, "hook1")
//								return aroseError
//							}).WithAfterHook(func(ctx context.Context, aroseError error) error {
//								testtool.Error(t, aroseError)
//								testtool.True(t, Errors.Is(aroseError, testtool.ErrExpTestA))
//								executed = append(executed, "hook2")
//								return aroseError
//							}),
//						),
//				}
//				err := NewSaga(steps).Execute(ctx)
//				testtool.Error(t, err)
//				testtool.True(t, Errors.Is(err, testtool.ErrExpTestA))
//				testtool.True(t, slices.Equal([]string{"action1", "hook1", "hook2"}, executed))
//
//				testtool.LogError(t, err)
//			})
//			t.Run("after_with_retry", func(t *testing.T) {
//				var (
//					ctx      = context.Background()
//					executed []string
//				)
//
//				steps := []Step{
//					NewStep("step1").
//						WithAction(
//							NewAction(func(ctx context.Context) error {
//								executed = append(executed, "action1")
//								return testtool.ErrExpTestA
//							}).WithAfterHook(func(ctx context.Context, aroseError error) error {
//								testtool.Error(t, aroseError)
//								testtool.True(t, Errors.Is(aroseError, testtool.ErrExpTestA))
//								executed = append(executed, "hook1")
//								return aroseError
//							}).WithAfterHook(func(ctx context.Context, aroseError error) error {
//								testtool.Error(t, aroseError)
//								testtool.True(t, Errors.Is(aroseError, testtool.ErrExpTestA))
//								executed = append(executed, "hook2")
//								return aroseError
//							}).WithRetry(NewBaseRetryOpt(2, 1*time.Nanosecond)).
//								WithAfterHook(func(ctx context.Context, aroseError error) error {
//									testtool.True(t, Errors.Is(aroseError, testtool.ErrExpTestA))
//									executed = append(executed, "retry_hook1")
//									return aroseError
//								}).
//								WithAfterHook(func(ctx context.Context, aroseError error) error {
//									testtool.True(t, Errors.Is(aroseError, testtool.ErrExpTestA))
//									executed = append(executed, "retry_hook2")
//									return aroseError
//								}),
//						),
//				}
//				err := NewSaga(steps).Execute(ctx)
//				testtool.Error(t, err)
//				testtool.True(t,
//					slices.Equal(
//						[]string{
//							"action1", "hook1", "hook2", // call
//							"action1", "hook1", "hook2", // first retry
//							"action1", "hook1", "hook2", // second retry
//							"retry_hook1", "retry_hook2", // retry hooks
//						},
//						executed),
//				)
//
//				testtool.LogError(t, err)
//			})
//		})
//		t.Run("compensation_hooks", func(t *testing.T) {
//			t.Run("before", func(t *testing.T) {
//				var (
//					ctx      = context.Background()
//					executed []string
//				)
//
//				steps := []Step{
//					NewStep("step1").
//						WithAction(
//							NewAction(func(ctx context.Context) error {
//								executed = append(executed, "action1")
//								return testtool.ErrExpTestA
//							}),
//						).WithCompensation(
//						NewCompensation(func(ctx context.Context, actionErr error) error {
//							executed = append(executed, "comp1")
//							return nil
//						}).WithBeforeHook(func(ctx context.Context, actionErr error) error {
//							testtool.Error(t, actionErr)
//							testtool.True(t, Errors.Is(actionErr, testtool.ErrExpTestA))
//							executed = append(executed, "comp_hook1")
//							return nil
//						}).WithBeforeHook(func(ctx context.Context, actionErr error) error {
//							testtool.Error(t, actionErr)
//							testtool.True(t, Errors.Is(actionErr, testtool.ErrExpTestA))
//							executed = append(executed, "comp_hook2")
//							return nil
//						}),
//					).WithCompensationOnFail(),
//				}
//				err := NewSaga(steps).Execute(ctx)
//				testtool.Error(t, err)
//				testtool.True(t, Errors.Is(err, testtool.ErrExpTestA))
//				testtool.True(t, Errors.Is(err, ErrCompensationSuccess))
//				testtool.True(t, slices.Equal([]string{"action1", "comp_hook2", "comp_hook1", "comp1"}, executed))
//
//				testtool.LogError(t, err)
//			})
//			t.Run("after", func(t *testing.T) {
//				var (
//					ctx             = context.Background()
//					previousHookErr = fmt.Errorf("previous_hook_err_1")
//					compErr         = fmt.Errorf("comp_error_1")
//					executed        []string
//				)
//
//				steps := []Step{
//					NewStep("step1").
//						WithAction(
//							NewAction(func(ctx context.Context) error {
//								executed = append(executed, "action1")
//								return testtool.ErrExpTestA
//							}),
//						).WithCompensation(
//						NewCompensation(func(ctx context.Context, actionErr error) error {
//							executed = append(executed, "comp1")
//							return compErr
//						}).WithAfterHook(func(ctx context.Context, actionErr, previousErr error) error {
//							testtool.Error(t, actionErr)
//							testtool.True(t, Errors.Is(actionErr, testtool.ErrExpTestA))
//							testtool.True(t, Errors.Is(previousErr, compErr))
//							executed = append(executed, "comp_hook1")
//							return previousHookErr
//						}).WithAfterHook(func(ctx context.Context, actionErr, previousErr error) error {
//							testtool.Error(t, actionErr)
//							testtool.True(t, Errors.Is(actionErr, testtool.ErrExpTestA))
//							testtool.True(t, Errors.Is(previousErr, previousHookErr))
//							executed = append(executed, "comp_hook2")
//							return nil
//						}),
//					).WithCompensationOnFail(),
//				}
//
//				err := NewSaga(steps).Execute(ctx)
//				testtool.Error(t, err)
//				testtool.True(t, Errors.Is(err, testtool.ErrExpTestA))
//				testtool.True(t, Errors.Is(err, ErrCompensationSuccess))
//				testtool.True(t, slices.Equal([]string{"action1", "comp1", "comp_hook1", "comp_hook2"}, executed))
//
//				testtool.LogError(t, err)
//			})
//		})
//	}
func Test_wrapper(t *testing.T) {
	//	t.Run("action", func(t *testing.T) {
	//		var (
	//			ctx      = context.Background()
	//			executed []string
	//		)
	//
	//		steps := []Step{
	//			NewStep("step1").
	//				WithAction(
	//					NewAction(func(ctx context.Context) error {
	//						executed = append(executed, "action1")
	//						return nil
	//					}).WithWrapper(func(ctx context.Context, action ActionFunc) error {
	//						executed = append(executed, "before1")
	//						err := action(ctx)
	//						testtool.NoError(t, err)
	//						executed = append(executed, "after1")
	//						return nil
	//					}),
	//				),
	//		}
	//
	//		err := NewSaga(steps).Execute(ctx)
	//		testtool.NoError(t, err)
	//		testtool.True(t, slices.Equal([]string{"before1", "action1", "after1"}, executed))
	//	})
	//	t.Run("compensation", func(t *testing.T) {
	//		var (
	//			ctx      = context.Background()
	//			expErr   = testtool.ErrExpTestA
	//			executed []string
	//		)
	//
	//		steps := []Step{
	//			NewStep("step1").
	//				WithAction(
	//					NewAction(func(ctx context.Context) error {
	//						executed = append(executed, "action1")
	//						return expErr
	//					}).WithWrapper(func(ctx context.Context, action ActionFunc) error {
	//						executed = append(executed, "before_action1")
	//						err := action(ctx) // call action
	//						testtool.Error(t, err)
	//						testtool.True(t, Errors.Is(err, expErr))
	//						executed = append(executed, "after_action1")
	//						return err
	//					}),
	//				).WithCompensation(
	//				NewCompensation(func(ctx context.Context, actionErr error) error {
	//					executed = append(executed, "com1")
	//					testtool.Error(t, actionErr)
	//					testtool.True(t, Errors.Is(actionErr, expErr))
	//					return nil
	//				}).WithWrapper(func(ctx context.Context, actionErr error, comp CompensationFunc) error {
	//					executed = append(executed, "before_comp1")
	//					testtool.Error(t, actionErr)
	//					testtool.True(t, Errors.Is(actionErr, expErr))
	//					err := comp(ctx, actionErr) // call compensation
	//					testtool.NoError(t, err)
	//					executed = append(executed, "after_comp1")
	//					return nil
	//				}),
	//			).WithCompensationOnFail(),
	//		}
	//
	//		err := NewSaga(steps).Execute(ctx)
	//		testtool.Error(t, err)
	//		testtool.True(t, Errors.Is(err, ErrActionFailed))
	//		testtool.True(t,
	//			slices.Equal(
	//				[]string{
	//					"before_action1", "action1", "after_action1",
	//					"before_comp1", "com1", "after_comp1",
	//				}, executed))
	//	})
}

func Test_steps(t *testing.T) {
	t.Run("action", func(t *testing.T) {
		var (
			ctx = context.Background()
		)
		t.Run("success_v1", func(t *testing.T) {
			steps := []Step{
				NewStep("step1").
					WithAction(
						NewAction(func(ctx context.Context, track Track) error {
							return nil
						}),
					),
			}

			res, err := NewSaga(steps).Execute(ctx)
			assert.NoError(t, err)
			assert.Equal(t, StageResultSuccess, res.Status)

			assert.Equal(t, 1, len(res.Tracks))

			assert.Equal(t, "step1", res.Tracks[0].StepName)
			assert.Equal(t, 0, res.Tracks[0].StepPosition)
			assert.Equal(t, ExecutionStatusSuccess, res.Tracks[0].Action.Status)
			assert.Equal(t, 1, res.Tracks[0].Action.Calls)
			assert.Equal(t, 0, len(res.Tracks[0].Action.Errors))

			testtool.TestFn(t, func() {
				t.Log(t, res)
			})
		})
		t.Run("fail_v1", func(t *testing.T) {
			steps := []Step{
				NewStep("step1").
					WithAction(
						NewAction(func(ctx context.Context, track Track) error {
							return testtool.ErrExpTestA
						}),
					),
			}

			res, err := NewSaga(steps).Execute(ctx)
			assert.Error(t, err)
			assert.Equal(t, StageResultFail, res.Status)
			assert.True(t, errors.Is(err, ErrActionFailed))
			assert.Equal(t, 1, len(res.Tracks))
			assert.Equal(t, "step1", res.Tracks[0].StepName)
			assert.Equal(t, 0, res.Tracks[0].StepPosition)
			assert.Equal(t, ExecutionStatusFail, res.Tracks[0].Action.Status)
			assert.Equal(t, 1, res.Tracks[0].Action.Calls)
			assert.Equal(t, 1, len(res.Tracks[0].Action.Errors))
			assert.Equal(t, testtool.ErrExpTestA, res.Tracks[0].Action.Errors[0])

			testtool.TestFn(t, func() {
				t.Log(t, res)
			})
		})
		t.Run("fail_v2", func(t *testing.T) {
			steps := []Step{
				NewStep("step1").
					WithAction(
						NewAction(func(ctx context.Context, track Track) error {
							return nil
						}),
					),
				NewStep("step2").
					WithAction(
						NewAction(func(ctx context.Context, track Track) error {
							return testtool.ErrExpTestA
						}),
					),
			}

			res, err := NewSaga(steps).Execute(ctx)
			assert.Error(t, err)
			assert.Equal(t, StageResultFail, res.Status)
			assert.True(t, errors.Is(err, ErrActionFailed))

			assert.Equal(t, 2, len(res.Tracks))

			assert.Equal(t, "step1", res.Tracks[0].StepName)
			assert.Equal(t, 0, res.Tracks[0].StepPosition)
			assert.Equal(t, ExecutionStatusSuccess, res.Tracks[0].Action.Status)
			assert.Equal(t, 1, res.Tracks[0].Action.Calls)
			assert.Equal(t, 0, len(res.Tracks[0].Action.Errors))

			assert.Equal(t, "step2", res.Tracks[1].StepName)
			assert.Equal(t, 1, res.Tracks[1].StepPosition)
			assert.Equal(t, ExecutionStatusFail, res.Tracks[1].Action.Status)
			assert.Equal(t, 1, res.Tracks[1].Action.Calls)
			assert.Equal(t, 1, len(res.Tracks[1].Action.Errors))
			assert.Equal(t, testtool.ErrExpTestA, res.Tracks[1].Action.Errors[0])

			testtool.TestFn(t, func() {
				t.Log(t, res)
			})
		})
	})

	t.Run("compensation", func(t *testing.T) {
		var (
			ctx = context.Background()
		)
		t.Run("success_v1", func(t *testing.T) {
			steps := []Step{
				NewStep("step1").
					WithAction(
						NewAction(func(ctx context.Context, track Track) error {
							return testtool.ErrExpTestA
						}),
					).
					WithCompensation(
						NewCompensation(func(ctx context.Context, track Track) error {
							str := track.GetData()
							assert.Equal(t, "step1", str.StepName)
							assert.Equal(t, 0, str.StepPosition)
							assert.Equal(t, ExecutionStatusFail, str.Action.Status)
							assert.Equal(t, 1, str.Action.Calls)
							assert.Equal(t, 1, len(str.Action.Errors))

							assert.Equal(t, ExecutionStatusUncalled, str.Compensation.Status)
							assert.Equal(t, 1, str.Compensation.Calls)
							assert.Equal(t, 0, len(str.Compensation.Errors))
							return nil
						}),
					).WithCompensationOnFail(),
			}

			res, err := NewSaga(steps).Execute(ctx)
			assert.Error(t, err)
			assert.Equal(t, StageResultCompensated, res.Status)
			assert.True(t, errors.Is(err, ErrActionFailed))

			assert.Equal(t, 1, len(res.Tracks))

			assert.Equal(t, "step1", res.Tracks[0].StepName)
			assert.Equal(t, 0, res.Tracks[0].StepPosition)
			assert.Equal(t, ExecutionStatusFail, res.Tracks[0].Action.Status)
			assert.Equal(t, 1, res.Tracks[0].Action.Calls)
			assert.Equal(t, 1, len(res.Tracks[0].Action.Errors))

			assert.Equal(t, ExecutionStatusSuccess, res.Tracks[0].Compensation.Status)
			assert.Equal(t, 1, res.Tracks[0].Compensation.Calls)
			assert.Equal(t, 0, len(res.Tracks[0].Compensation.Errors))

			testtool.TestFn(t, func() {
				t.Log(t, res)
			})
		})
		t.Run("compensate_v1", func(t *testing.T) {
			steps := []Step{
				NewStep("step1").
					WithAction(
						NewAction(func(ctx context.Context, track Track) error {
							return testtool.ErrExpTestA
						}),
					).
					WithCompensation(
						NewCompensation(func(ctx context.Context, track Track) error {
							str := track.GetData()
							assert.Equal(t, "step1", str.StepName)
							assert.Equal(t, 0, str.StepPosition)
							assert.Equal(t, ExecutionStatusFail, str.Action.Status)
							assert.Equal(t, 1, str.Action.Calls)
							assert.Equal(t, 1, len(str.Action.Errors))
							assert.Equal(t, testtool.ErrExpTestA, str.Action.Errors[0])

							assert.Equal(t, ExecutionStatusUncalled, str.Compensation.Status)
							assert.Equal(t, 1, str.Compensation.Calls)
							assert.Equal(t, 0, len(str.Compensation.Errors))
							return nil
						}),
					).WithCompensationOnFail(),
			}

			res, err := NewSaga(steps).Execute(ctx)
			assert.Error(t, err)
			assert.Equal(t, StageResultCompensated, res.Status)
			assert.True(t, errors.Is(err, ErrActionFailed))

			assert.Equal(t, 1, len(res.Tracks))

			assert.Equal(t, "step1", res.Tracks[0].StepName)
			assert.Equal(t, 0, res.Tracks[0].StepPosition)
			assert.Equal(t, ExecutionStatusFail, res.Tracks[0].Action.Status)
			assert.Equal(t, 1, res.Tracks[0].Action.Calls)
			assert.Equal(t, 1, len(res.Tracks[0].Action.Errors))

			assert.Equal(t, ExecutionStatusSuccess, res.Tracks[0].Compensation.Status)
			assert.Equal(t, 1, res.Tracks[0].Compensation.Calls)
			assert.Equal(t, 0, len(res.Tracks[0].Compensation.Errors))

			testtool.TestFn(t, func() {
				t.Log(t, res)
			})
		})
		t.Run("fail_v1", func(t *testing.T) {
			steps := []Step{
				NewStep("step1").
					WithAction(
						NewAction(func(ctx context.Context, track Track) error {
							return testtool.ErrExpTestA
						}),
					).
					WithCompensation(
						NewCompensation(func(ctx context.Context, track Track) error {
							str := track.GetData()
							assert.Equal(t, "step1", str.StepName)
							assert.Equal(t, 0, str.StepPosition)
							assert.Equal(t, ExecutionStatusFail, str.Action.Status)
							assert.Equal(t, 1, str.Action.Calls)
							assert.Equal(t, 1, len(str.Action.Errors))
							assert.Equal(t, testtool.ErrExpTestA, str.Action.Errors[0])

							assert.Equal(t, ExecutionStatusUncalled, str.Compensation.Status)
							assert.Equal(t, 1, str.Compensation.Calls)
							assert.Equal(t, 0, len(str.Compensation.Errors))
							return testtool.ErrExpTestB
						}),
					).WithCompensationOnFail(),
			}

			res, err := NewSaga(steps).Execute(ctx)
			assert.Error(t, err)
			assert.Equal(t, StageResultFail, res.Status)
			assert.ErrorIs(t, err, ErrActionFailed)
			assert.ErrorIs(t, err, ErrCompensationFailed)

			assert.Equal(t, 1, len(res.Tracks))

			assert.Equal(t, "step1", res.Tracks[0].StepName)
			assert.Equal(t, 0, res.Tracks[0].StepPosition)
			assert.Equal(t, ExecutionStatusFail, res.Tracks[0].Action.Status)
			assert.Equal(t, 1, res.Tracks[0].Action.Calls)
			assert.Equal(t, 1, len(res.Tracks[0].Action.Errors))

			assert.Equal(t, ExecutionStatusFail, res.Tracks[0].Compensation.Status)
			assert.Equal(t, 1, res.Tracks[0].Compensation.Calls)
			assert.Equal(t, 1, len(res.Tracks[0].Compensation.Errors))
			assert.ErrorIs(t, res.Tracks[0].Compensation.Errors[0], testtool.ErrExpTestB)

			testtool.TestFn(t, func() {
				t.Log(t, res)
			})
		})
	})
}

func Test_retry(t *testing.T) {
	var (
		ctx = context.Background()
	)
	t.Run("compensation", func(t *testing.T) {
		steps := []Step{
			NewStep("step1").
				WithAction(
					NewAction(func(ctx context.Context, track Track) error {
						return testtool.ErrExpTestA
					}).WithRetry(NewBaseRetryOpt(4, 5*time.Nanosecond)),
				).
				WithCompensation(
					NewCompensation(func(ctx context.Context, track Track) error {
						str := track.GetData()
						if str.Compensation.Calls < 3 {
							return fmt.Errorf("comp err [%d]: %w", len(str.Compensation.Errors), testtool.ErrExpTestA)
						}
						return nil
					}).WithRetry(NewBaseRetryOpt(4, 5*time.Nanosecond)),
				).WithCompensationOnFail(),
		}

		res, err := NewSaga(steps).Execute(ctx)
		assert.Error(t, err)
		assert.Equal(t, StageResultCompensated, res.Status)
		assert.ErrorIs(t, err, ErrActionFailed)
		assert.ErrorIsNot(t, err, ErrCompensationFailed)

		assert.Equal(t, 1, len(res.Tracks))

		testtool.TestFn(t, func() {
			t.Log(
				res,
				"error:", err,
				"\nAction errors: ", res.Tracks[0].Action.Errors,
				"\nCompensation errors: ", res.Tracks[0].Compensation.Errors,
			)
		})

	})
	t.Run("compensation", func(t *testing.T) {
		steps := []Step{
			NewStep("step1").
				WithAction(
					NewAction(func(ctx context.Context, track Track) error {
						return testtool.ErrExpTestA
					}).WithRetry(NewBaseRetryOpt(4, 5*time.Nanosecond)),
				).
				WithCompensation(
					NewCompensation(func(ctx context.Context, track Track) error {
						str := track.GetData()
						return fmt.Errorf("comp err [%d]: %w", len(str.Compensation.Errors), testtool.ErrExpTestA)
					}).WithRetry(NewBaseRetryOpt(4, 5*time.Nanosecond)),
				).WithCompensationOnFail(),
		}

		res, err := NewSaga(steps).Execute(ctx)
		assert.Error(t, err)
		assert.Equal(t, StageResultFail, res.Status)
		assert.ErrorIs(t, err, ErrActionFailed)
		assert.ErrorIs(t, err, ErrCompensationFailed)

		testtool.TestFn(t, func() {
			t.Log(t, res)
		})
	})
}
