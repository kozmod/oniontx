package saga

import (
	"context"
	"errors"
	"fmt"
	"slices"
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
func Test_Saga_panic_recovery(t *testing.T) {
	var (
		ctx = context.Background()
	)
	t.Run("static_func", func(t *testing.T) {
		t.Run("success_v1", func(t *testing.T) {
			steps := []Step{
				{
					Name: "step0",
					Action: WithPanicRecovery(func(ctx context.Context, _ Track) error {
						panic("panic_v1!")
					}),
				},
			}

			res, err := NewSaga(steps).Execute(ctx)
			assert.Error(t, err)
			assert.Equal(t, StageResultFail, res.Status)
			assert.Equal(t, 1, len(res.Tracks))
			assert.Equal(t, ExecutionStatusFail, res.Tracks[0].Action.Status)
			assert.Equal(t, 1, len(res.Tracks[0].Action.Errors))
			assert.ErrorIs(t, res.Tracks[0].Action.Errors[0], ErrPanicRecovered)
			assert.Equal(t, ExecutionStatusUncalled, res.Tracks[0].Compensation.Status)
			assert.Equal(t, 0, res.Tracks[0].Compensation.Calls)
			assert.Equal(t, 0, len(res.Tracks[0].Compensation.Errors))

		})
	})
	t.Run("builder_stile", func(t *testing.T) {
		t.Run("success_ActionFunc", func(t *testing.T) {
			steps := []Step{
				NewStep("step0").
					WithAction(
						NewAction(func(ctx context.Context, _ Track) error {
							panic("panic_v2!")
						}).WithPanicRecovery(),
					),
			}

			res, err := NewSaga(steps).Execute(ctx)
			assert.Error(t, err)
			assert.Equal(t, StageResultFail, res.Status)
			assert.Equal(t, 1, len(res.Tracks))
			assert.Equal(t, ExecutionStatusFail, res.Tracks[0].Action.Status)
			assert.Equal(t, 1, len(res.Tracks[0].Action.Errors))
			assert.ErrorIs(t, res.Tracks[0].Action.Errors[0], ErrPanicRecovered)
			assert.Equal(t, ExecutionStatusUncalled, res.Tracks[0].Compensation.Status)
			assert.Equal(t, 0, res.Tracks[0].Compensation.Calls)
			assert.Equal(t, 0, len(res.Tracks[0].Compensation.Errors))
		})

		t.Run("success_CompensationFunc", func(t *testing.T) {
			steps := []Step{
				{
					Name: "step0",
					Action: ActionFunc(func(ctx context.Context, _ Track) error {
						return testtool.ErrExpTestA
					}),
					Compensation: CompensationFunc(func(ctx context.Context, track Track) error {
						str := track.GetData()
						assert.Equal(t, 1, len(str.Action.Errors))
						assert.Equal(t, 1, str.Action.Calls)
						assert.Equal(t, ExecutionStatusFail, str.Action.Status)
						assert.Error(t, str.Action.Errors[0])
						assert.ErrorIs(t, str.Action.Errors[0], testtool.ErrExpTestA)

						panic("panic_v3!")
					}).WithPanicRecovery(),
					CompensationOnFail: true,
				},
			}

			res, err := NewSaga(steps).Execute(ctx)
			assert.Error(t, err)
			assert.Equal(t, StageResultFail, res.Status)
			assert.Equal(t, 1, len(res.Tracks))
			assert.Equal(t, ExecutionStatusFail, res.Tracks[0].Action.Status)
			assert.Equal(t, 1, len(res.Tracks[0].Action.Errors))
			assert.ErrorIs(t, res.Tracks[0].Action.Errors[0], testtool.ErrExpTestA)
			assert.Equal(t, ExecutionStatusFail, res.Tracks[0].Compensation.Status)
			assert.Equal(t, 1, res.Tracks[0].Compensation.Calls)
			assert.Equal(t, 1, len(res.Tracks[0].Compensation.Errors))
			assert.ErrorIs(t, res.Tracks[0].Compensation.Errors[0], ErrPanicRecovered)
		})
	})
}

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
				WithAction(func(ctx context.Context, _ Track) error {
					executedActions = append(executedActions, "action1")
					return nil
				}),
			NewStep("step2").
				WithAction(func(ctx context.Context, _ Track) error {
					executedActions = append(executedActions, "action2")
					cancel() // cancel context for test
					return nil
				}),
			NewStep("step3").
				WithAction(func(ctx context.Context, _ Track) error {
					executedActions = append(executedActions, "action3")
					t.Fatalf("should not have been called")
					return nil
				}),
		}

		res, err := NewSaga(steps).Execute(ctx)
		assert.Error(t, err)
		//assert.ErrorIs(t, err, ErrExecuteActionsContextDone)

		assert.Equal(t, StageResultFail, res.Status)
		assert.Equal(t, 4, len(res.Tracks))
		assert.Equal(t, "step0", res.Tracks[0].StepName)
		assert.Equal(t, 0, res.Tracks[0].StepPosition)
		assert.Equal(t, ExecutionStatusUncalled, res.Tracks[0].Action.Status)
		assert.Equal(t, ExecutionStatusUncalled, res.Tracks[0].Compensation.Status)

		assert.Equal(t, "step1", res.Tracks[1].StepName)
		assert.Equal(t, 1, res.Tracks[1].StepPosition)
		assert.Equal(t, ExecutionStatusSuccess, res.Tracks[1].Action.Status)
		assert.Equal(t, ExecutionStatusUncalled, res.Tracks[1].Compensation.Status)

		assert.Equal(t, "step2", res.Tracks[2].StepName)
		assert.Equal(t, 2, res.Tracks[2].StepPosition)
		assert.Equal(t, ExecutionStatusSuccess, res.Tracks[2].Action.Status)
		assert.Equal(t, ExecutionStatusUncalled, res.Tracks[2].Compensation.Status)

		assert.Equal(t, "step3", res.Tracks[3].StepName)
		assert.Equal(t, 3, res.Tracks[3].StepPosition)
		assert.Equal(t, ExecutionStatusFail, res.Tracks[3].Action.Status)
		assert.Equal(t, ExecutionStatusUncalled, res.Tracks[3].Compensation.Status)
		//assert.Equal(t, ExecutionStatusUncalled, res.Tracks[3].Compensation.Status) //TODO: check errors

		assert.True(t, slices.Equal([]string{"action1", "action2"}, executedActions))
	})
	//t.Run("retry_ctx_cancel", func(t *testing.T) {
	//	var (
	//		ctx, cancel = context.WithCancel(context.Background())
	//		executed    []string
	//		actionCalls = 1
	//	)
	//
	//	steps := []Step{
	//		NewStep("step0").
	//			WithAction(nil),
	//		NewStep("step1").
	//			WithAction(
	//				NewAction(func(ctx context.Context) error {
	//					executed = append(executed, "action1")
	//					switch {
	//					case actionCalls == 1:
	//						actionCalls++
	//						return testtool.ErrExpTestA
	//					case actionCalls == 2:
	//						actionCalls++
	//						return testtool.ErrExpTestA
	//					case actionCalls >= 3:
	//						actionCalls++
	//						cancel() // cancel context for test
	//						return testtool.ErrExpTestA
	//					}
	//					return nil
	//				}).WithRetry(NewBaseRetryOpt(4, 1*time.Nanosecond)),
	//			),
	//	}
	//	err := NewSaga(steps).Execute(ctx)
	//	testtool.Error(t, err)
	//	testtool.True(t, Errors.Is(err, ErrRetryContextDone))
	//	testtool.True(t, 4 == actionCalls) // 3 + first execution
	//	testtool.True(t, slices.Equal([]string{"action1", "action1", "action1"}, executed))
	//
	//	testtool.LogError(t, err)
	//})
	//
	//t.Run("compensation_ctx_cancel", func(t *testing.T) {
	//	var (
	//		ctx, cancel = context.WithCancel(context.Background())
	//		executed    []string
	//	)
	//
	//	steps := []Step{
	//		NewStep("step1").
	//			WithAction(
	//				NewAction(func(ctx context.Context) error {
	//					executed = append(executed, "action1")
	//					cancel() // cancel context for test
	//					return testtool.ErrExpTestA
	//				}),
	//			).WithCompensation(
	//			NewCompensation(func(ctx context.Context, aroseErr error) error {
	//				t.Fatalf("should not have been called")
	//				return nil
	//			}),
	//		).WithCompensationOnFail(),
	//	}
	//	err := NewSaga(steps).Execute(ctx)
	//	testtool.Error(t, err)
	//	testtool.True(t, Errors.Is(err, ErrExecuteCompensationContextDone))
	//	testtool.True(t, slices.Equal([]string{"action1"}, executed))
	//
	//	testtool.LogError(t, err)
	//})
}

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
	var (
		ctx = context.Background()
	)
	t.Run("action", func(t *testing.T) {
		var (
			calls = make([]string, 0, 3)
		)

		steps := []Step{
			NewStep("step1").
				WithAction(
					NewAction(func(ctx context.Context, _ Track) error {
						calls = append(calls, "action1")
						return nil
					}).WithWrapper(func(ctx context.Context, track Track, action ActionFunc) error {
						calls = append(calls, "before1")
						err := action(ctx, track)
						assert.NoError(t, err)
						calls = append(calls, "after1")
						return nil
					}),
				),
		}

		res, err := NewSaga(steps).Execute(ctx)
		assert.NoError(t, err)

		assert.Equal(t, StageResultSuccess, res.Status)
		assert.Equal(t, 1, res.Tracks[0].Action.Calls)
		assert.Equal(t, ExecutionStatusSuccess, res.Tracks[0].Action.Status)
		assert.Equal(t, 0, res.Tracks[0].Compensation.Calls)
		assert.Equal(t, ExecutionStatusUncalled, res.Tracks[0].Compensation.Status)

		assert.True(t, slices.Equal([]string{"before1", "action1", "after1"}, calls))
	})
	t.Run("compensation", func(t *testing.T) {
		var (
			expErr = testtool.ErrExpTestA
			calls  = make([]string, 0, 6)
		)

		steps := []Step{
			NewStep("step1").
				WithAction(
					NewAction(func(ctx context.Context, _ Track) error {
						calls = append(calls, "action1")
						return expErr
					}).WithWrapper(func(ctx context.Context, track Track, action ActionFunc) error {
						calls = append(calls, "before_action1")
						err := action(ctx, track) // call action
						assert.Error(t, err)
						assert.ErrorIs(t, err, expErr)
						calls = append(calls, "after_action1")
						return err
					}),
				).WithCompensation(
				NewCompensation(func(ctx context.Context, track Track) error {
					calls = append(calls, "com1")
					actionErr := track.GetData().Action.Errors[0]
					assert.Error(t, actionErr)
					assert.ErrorIs(t, actionErr, expErr)
					return nil
				}).WithWrapper(func(ctx context.Context, track Track, comp CompensationFunc) error {
					calls = append(calls, "before_comp1")
					actionErr := track.GetData().Action.Errors[0]
					assert.Error(t, actionErr)
					assert.ErrorIs(t, actionErr, expErr)
					err := comp(ctx, track) // call compensation
					assert.NoError(t, err)
					calls = append(calls, "after_comp1")
					return nil
				}),
			).WithCompensationOnFail(),
		}

		res, err := NewSaga(steps).Execute(ctx)
		assert.Error(t, err)
		assert.Equal(t, StageResultCompensated, res.Status)
		assert.Equal(t, 1, res.Tracks[0].Action.Calls)
		assert.Equal(t, 1, len(res.Tracks[0].Action.Errors))
		assert.Equal(t, ExecutionStatusFail, res.Tracks[0].Action.Status)
		assert.Equal(t, 1, res.Tracks[0].Compensation.Calls)
		assert.Equal(t, ExecutionStatusSuccess, res.Tracks[0].Compensation.Status)
		assert.Equal(t, 0, len(res.Tracks[0].Compensation.Errors))

		assert.True(t,
			slices.Equal(
				[]string{
					"before_action1", "action1", "after_action1",
					"before_comp1", "com1", "after_comp1",
				}, calls))
	})
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
						NewAction(func(ctx context.Context, _ Track) error {
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
				t.Log(res)
			})
		})
		t.Run("fail_v1", func(t *testing.T) {
			steps := []Step{
				NewStep("step1").
					WithAction(
						NewAction(func(ctx context.Context, _ Track) error {
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
				t.Log(res)
			})
		})
		t.Run("fail_v2", func(t *testing.T) {
			steps := []Step{
				NewStep("step1").
					WithAction(
						NewAction(func(ctx context.Context, _ Track) error {
							return nil
						}),
					),
				NewStep("step2").
					WithAction(
						NewAction(func(ctx context.Context, _ Track) error {
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
				t.Log(res)
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
						NewAction(func(ctx context.Context, _ Track) error {
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
				t.Log(res)
			})
		})
		t.Run("compensate_v1", func(t *testing.T) {
			steps := []Step{
				NewStep("step1").
					WithAction(
						NewAction(func(ctx context.Context, _ Track) error {
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
				t.Log(res)
			})
		})
		t.Run("fail_v1", func(t *testing.T) {
			steps := []Step{
				NewStep("step1").
					WithAction(
						NewAction(func(ctx context.Context, _ Track) error {
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
				t.Log(res)
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
					NewAction(func(ctx context.Context, _ Track) error {
						return testtool.ErrExpTestA
					}).
						WithRetry(NewBaseRetryOpt(4, 5*time.Nanosecond)),
				).
				WithCompensation(
					NewCompensation(func(ctx context.Context, track Track) error {
						str := track.GetData()
						if str.Compensation.Calls < 5 {
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

		assert.Equal(t, "step1", res.Tracks[0].StepName)
		assert.Equal(t, 0, res.Tracks[0].StepPosition)
		assert.Equal(t, ExecutionStatusFail, res.Tracks[0].Action.Status)
		assert.Equal(t, 5, res.Tracks[0].Action.Calls)
		assert.Equal(t, 5, len(res.Tracks[0].Action.Errors))

		for _, e := range res.Tracks[0].Action.Errors {
			assert.ErrorIs(t, e, testtool.ErrExpTestA)
		}

		assert.Equal(t, ExecutionStatusSuccess, res.Tracks[0].Compensation.Status)
		assert.Equal(t, 5, res.Tracks[0].Compensation.Calls)
		assert.Equal(t, 4, len(res.Tracks[0].Compensation.Errors))
		for _, e := range res.Tracks[0].Compensation.Errors {
			assert.ErrorIs(t, e, testtool.ErrExpTestA)
		}

		testtool.TestFn(t, func() {
			t.Log(
				res,
				"+ error:", err,
				"\n + Action errors: ", res.Tracks[0].Action.Errors,
				"\n + Compensation errors: ", res.Tracks[0].Compensation.Errors,
			)
		})

	})
	t.Run("compensation", func(t *testing.T) {
		steps := []Step{
			NewStep("step1").
				WithAction(
					NewAction(func(ctx context.Context, _ Track) error {
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

		assert.Equal(t, 1, len(res.Tracks))

		assert.Equal(t, "step1", res.Tracks[0].StepName)
		assert.Equal(t, 0, res.Tracks[0].StepPosition)
		assert.Equal(t, ExecutionStatusFail, res.Tracks[0].Action.Status)
		assert.Equal(t, 5, res.Tracks[0].Action.Calls)
		assert.Equal(t, 5, len(res.Tracks[0].Action.Errors))

		for _, e := range res.Tracks[0].Action.Errors {
			assert.ErrorIs(t, e, testtool.ErrExpTestA)
		}

		assert.Equal(t, ExecutionStatusFail, res.Tracks[0].Compensation.Status)
		assert.Equal(t, 5, res.Tracks[0].Compensation.Calls)
		assert.Equal(t, 5, len(res.Tracks[0].Compensation.Errors))
		for _, e := range res.Tracks[0].Compensation.Errors {
			assert.ErrorIs(t, e, testtool.ErrExpTestA)
		}

		testtool.TestFn(t, func() {
			t.Log(
				res,
				"+ error:", err,
				"\n + Action errors: ", res.Tracks[0].Action.Errors,
				"\n + Compensation errors: ", res.Tracks[0].Compensation.Errors,
			)
		})

	})
}
