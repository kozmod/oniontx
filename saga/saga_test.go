package saga

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/kozmod/oniontx/internal/testtool"
	"github.com/kozmod/oniontx/internal/testtool/assert"
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
				Action: func(ctx context.Context, _ Track) error {
					executedActions = append(executedActions, "action1")
					return nil
				},
				Compensation: func(ctx context.Context, _ Track) error {
					executedCompensation = append(executedCompensation, "comp1")
					t.Fatalf("should not have been called")
					return nil
				},
			},
			{
				Name: "step1",
				Action: func(ctx context.Context, _ Track) error {
					executedActions = append(executedActions, "action2")
					return nil
				},
				Compensation: func(ctx context.Context, _ Track) error {
					executedCompensation = append(executedCompensation, "comp2")
					t.Fatalf("should not have been called")
					return nil
				},
			},
		}

		res, err := NewSaga(steps).Execute(ctx)
		assert.NoError(t, err)
		assert.Equal(t, StageResultSuccess, res.Status)
		assert.Equal(t, 2, len(res.Steps))

		assert.Equal(t, "step0", res.Steps[0].StepName)
		assert.Equal(t, 0, res.Steps[0].StepPosition)
		assert.Equal(t, 1, res.Steps[0].Action.Calls)
		assert.Equal(t, ExecutionStatusSuccess, res.Steps[0].Action.Status)
		assert.Equal(t, 0, res.Steps[0].Compensation.Calls)

		assert.Equal(t, "step1", res.Steps[1].StepName)
		assert.Equal(t, 1, res.Steps[1].StepPosition)
		assert.Equal(t, 1, res.Steps[1].Action.Calls)
		assert.Equal(t, ExecutionStatusSuccess, res.Steps[1].Action.Status)
		assert.Equal(t, 0, res.Steps[1].Compensation.Calls)

		assert.True(t, slices.Equal([]string{"action1", "action2"}, executedActions))
		assert.True(t, len(executedCompensation) == 0)
	})

	t.Run("success_compensation_on_step1", func(t *testing.T) {
		var (
			executedActions      []string
			executedCompensation []string
		)

		steps := []Step{
			{
				Name: "step0",
				Action: func(ctx context.Context, _ Track) error {
					executedActions = append(executedActions, "action1")
					return nil
				},
				Compensation: func(ctx context.Context, _ Track) error {
					executedCompensation = append(executedCompensation, "comp1")
					return nil
				},
			},
			{
				Name: "step1",
				Action: NewAction(func(ctx context.Context, _ Track) error {
					executedActions = append(executedActions, "action2")
					return testtool.ErrExpTestA
				}),
				Compensation: NewCompensation(func(ctx context.Context, _ Track) error {
					executedCompensation = append(executedCompensation, "comp2")
					t.Fatalf("should not have been called")
					return nil
				}),
			},
		}

		res, err := NewSaga(steps).Execute(ctx)
		assert.Error(t, err)
		assert.Equal(t, StageResultCompensated, res.Status)
		assert.Equal(t, 2, len(res.Steps))

		assert.Equal(t, "step0", res.Steps[0].StepName)
		assert.Equal(t, 0, res.Steps[0].StepPosition)
		assert.Equal(t, 1, res.Steps[0].Action.Calls)
		assert.Equal(t, 1, res.Steps[0].Compensation.Calls)
		assert.Equal(t, ExecutionStatusSuccess, res.Steps[0].Compensation.Status)

		assert.Equal(t, "step1", res.Steps[1].StepName)
		assert.Equal(t, 1, res.Steps[1].StepPosition)
		assert.Equal(t, 1, res.Steps[1].Action.Calls)
		assert.Equal(t, 0, res.Steps[1].Compensation.Calls)

		assert.True(t, slices.Equal([]string{"action1", "action2"}, executedActions))
		assert.True(t, slices.Equal([]string{"comp1"}, executedCompensation))
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
					Action: func(ctx context.Context, _ Track) error {
						executedActions = append(executedActions, "action1")
						return testtool.ErrExpTestA
					},
					Compensation: func(ctx context.Context, _ Track) error {
						executedCompensation = append(executedCompensation, "comp1")
						t.Fatalf("should not have been called")
						return nil
					},
				},
			}

			res, err := NewSaga(steps).Execute(ctx)
			assert.Error(t, err)
			assert.Equal(t, StageResultFail, res.Status)
			assert.Equal(t, 1, len(res.Steps))

			assert.Equal(t, "step0", res.Steps[0].StepName)
			assert.Equal(t, 0, res.Steps[0].StepPosition)
			assert.Equal(t, 1, res.Steps[0].Action.Calls)
			assert.Equal(t, 1, len(res.Steps[0].Action.Errors))
			assert.ErrorIs(t, res.Steps[0].Action.Errors[0], testtool.ErrExpTestA)
			assert.Equal(t, ExecutionStatusFail, res.Steps[0].Action.Status)
			assert.Equal(t, 0, res.Steps[0].Compensation.Calls)
			assert.Equal(t, ExecutionStatusUncalled, res.Steps[0].Compensation.Status)

			assert.True(t, slices.Equal([]string{"action1"}, executedActions))
			assert.True(t, len(executedCompensation) == 0)
		})
		t.Run("added", func(t *testing.T) {
			var (
				executedActions      []string
				executedCompensation []string
			)

			steps := []Step{
				{
					Name: "step0",
					Action: func(ctx context.Context, _ Track) error {
						executedActions = append(executedActions, "action1")
						return testtool.ErrExpTestA
					},
					Compensation: func(ctx context.Context, _ Track) error {
						executedCompensation = append(executedCompensation, "comp1")
						return nil
					},
					CompensationRequired: true,
				},
			}

			res, err := NewSaga(steps).Execute(ctx)
			assert.Error(t, err)
			assert.Equal(t, StageResultCompensated, res.Status)
			assert.Equal(t, 1, len(res.Steps))

			assert.Equal(t, "step0", res.Steps[0].StepName)
			assert.Equal(t, 0, res.Steps[0].StepPosition)
			assert.Equal(t, 1, res.Steps[0].Action.Calls)
			assert.Equal(t, 1, len(res.Steps[0].Action.Errors))
			assert.ErrorIs(t, res.Steps[0].Action.Errors[0], testtool.ErrExpTestA)
			assert.Equal(t, ExecutionStatusFail, res.Steps[0].Action.Status)
			assert.Equal(t, 1, res.Steps[0].Compensation.Calls)
			assert.Equal(t, ExecutionStatusSuccess, res.Steps[0].Compensation.Status)

			assert.True(t, slices.Equal([]string{"action1"}, executedActions))
			assert.True(t, slices.Equal([]string{"comp1"}, executedCompensation))
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
					Action: WithPanicRecovery(func(ctx context.Context, _ Track) error {
						panic("panic_v1!")
					}),
				},
			}

			res, err := NewSaga(steps).Execute(ctx)
			assert.Error(t, err)
			assert.Equal(t, StageResultFail, res.Status)
			assert.Equal(t, 1, len(res.Steps))
			assert.Equal(t, ExecutionStatusFail, res.Steps[0].Action.Status)
			assert.Equal(t, 1, len(res.Steps[0].Action.Errors))
			assert.ErrorIs(t, res.Steps[0].Action.Errors[0], ErrPanicRecovered)
			assert.Equal(t, ExecutionStatusUnset, res.Steps[0].Compensation.Status)
			assert.Equal(t, 0, res.Steps[0].Compensation.Calls)
			assert.Equal(t, 0, len(res.Steps[0].Compensation.Errors))

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
			assert.Equal(t, 1, len(res.Steps))
			assert.Equal(t, ExecutionStatusFail, res.Steps[0].Action.Status)
			assert.Equal(t, 1, len(res.Steps[0].Action.Errors))
			assert.ErrorIs(t, res.Steps[0].Action.Errors[0], ErrPanicRecovered)
			assert.Equal(t, ExecutionStatusUnset, res.Steps[0].Compensation.Status)
			assert.Equal(t, 0, res.Steps[0].Compensation.Calls)
			assert.Equal(t, 0, len(res.Steps[0].Compensation.Errors))
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
					CompensationRequired: true,
				},
			}

			res, err := NewSaga(steps).Execute(ctx)
			assert.Error(t, err)
			assert.Equal(t, StageResultFail, res.Status)
			assert.Equal(t, 1, len(res.Steps))
			assert.Equal(t, ExecutionStatusFail, res.Steps[0].Action.Status)
			assert.Equal(t, 1, len(res.Steps[0].Action.Errors))
			assert.ErrorIs(t, res.Steps[0].Action.Errors[0], testtool.ErrExpTestA)
			assert.Equal(t, ExecutionStatusFail, res.Steps[0].Compensation.Status)
			assert.Equal(t, 1, res.Steps[0].Compensation.Calls)
			assert.Equal(t, 1, len(res.Steps[0].Compensation.Errors))
			assert.ErrorIs(t, res.Steps[0].Compensation.Errors[0], ErrPanicRecovered)
		})
	})
}

func Test_actions_v2(t *testing.T) {
	var (
		ctx = context.Background()
	)

	t.Run("success_actions", func(t *testing.T) {
		steps := []Step{
			NewStep("step0").
				WithAction(func(ctx context.Context, _ Track) error {
					return nil
				}).
				WithCompensation(func(ctx context.Context, _ Track) error {
					t.Fatalf("should not have been called")
					return nil
				}),
			NewStep("step1").
				WithAction(func(ctx context.Context, _ Track) error {
					return nil
				}).
				WithCompensation(func(ctx context.Context, _ Track) error {
					t.Fatalf("should not have been called")
					return nil
				}),
		}

		res, err := NewSaga(steps).Execute(ctx)
		assert.NoError(t, err)
		assert.Equal(t, StageResultSuccess, res.Status)
		assert.Equal(t, 2, len(res.Steps))
		assert.Equal(t, ExecutionStatusSuccess, res.Steps[0].Action.Status)
		assert.Equal(t, 1, res.Steps[0].Action.Calls)
		assert.Equal(t, 0, len(res.Steps[0].Action.Errors))
		assert.Equal(t, ExecutionStatusUncalled, res.Steps[0].Compensation.Status)
		assert.Equal(t, 0, res.Steps[0].Compensation.Calls)
		assert.Equal(t, 0, len(res.Steps[0].Compensation.Errors))

		assert.Equal(t, ExecutionStatusSuccess, res.Steps[1].Action.Status)
		assert.Equal(t, 1, res.Steps[1].Action.Calls)
		assert.Equal(t, 0, len(res.Steps[1].Action.Errors))
		assert.Equal(t, ExecutionStatusUncalled, res.Steps[1].Compensation.Status)
		assert.Equal(t, 0, res.Steps[1].Compensation.Calls)
		assert.Equal(t, 0, len(res.Steps[1].Compensation.Errors))
	})
}
func Test_execute_context(t *testing.T) {
	t.Run("action_ctx_cancel", func(t *testing.T) {
		var (
			ctx, cancel = context.WithCancel(context.Background())
			calls       = make([]string, 0, 2)
		)

		steps := []Step{
			NewStep("step0").
				WithAction(nil),
			NewStep("step1").
				WithAction(func(ctx context.Context, _ Track) error {
					calls = append(calls, "action1")
					return nil
				}),
			NewStep("step2").
				WithAction(func(ctx context.Context, _ Track) error {
					calls = append(calls, "action2")
					cancel() // cancel context for test
					return nil
				}),
			NewStep("step3").
				WithAction(func(ctx context.Context, _ Track) error {
					calls = append(calls, "action3")
					t.Fatalf("should not have been called")
					return nil
				}),
		}

		res, err := NewSaga(steps).Execute(ctx)
		assert.Error(t, err)

		assert.Equal(t, StageResultCompensated, res.Status)
		assert.Equal(t, 4, len(res.Steps))
		assert.Equal(t, "step0", res.Steps[0].StepName)
		assert.Equal(t, 0, res.Steps[0].StepPosition)
		assert.Equal(t, ExecutionStatusUnset, res.Steps[0].Action.Status)
		assert.Equal(t, ExecutionStatusUnset, res.Steps[0].Compensation.Status)
		assert.Equal(t, false, res.Steps[0].CompensationRequired)

		assert.Equal(t, "step1", res.Steps[1].StepName)
		assert.Equal(t, 1, res.Steps[1].StepPosition)
		assert.Equal(t, ExecutionStatusSuccess, res.Steps[1].Action.Status)
		assert.Equal(t, ExecutionStatusUnset, res.Steps[1].Compensation.Status)
		assert.Equal(t, false, res.Steps[1].CompensationRequired)

		assert.Equal(t, "step2", res.Steps[2].StepName)
		assert.Equal(t, 2, res.Steps[2].StepPosition)
		assert.Equal(t, ExecutionStatusSuccess, res.Steps[2].Action.Status)
		assert.Equal(t, ExecutionStatusUnset, res.Steps[2].Compensation.Status)
		assert.Equal(t, false, res.Steps[2].CompensationRequired)

		assert.Equal(t, "step3", res.Steps[3].StepName)
		assert.Equal(t, 3, res.Steps[3].StepPosition)
		assert.Equal(t, ExecutionStatusFail, res.Steps[3].Action.Status)
		assert.Equal(t, 1, len(res.Steps[3].Action.Errors))
		assert.ErrorIs(t, res.Steps[3].Action.Errors[0], ErrExecuteActionsContextDone)
		assert.Equal(t, ExecutionStatusUnset, res.Steps[3].Compensation.Status)
		assert.Equal(t, 0, len(res.Steps[3].Compensation.Errors))
		assert.Equal(t, false, res.Steps[3].CompensationRequired)

		assert.True(t, slices.Equal([]string{"action1", "action2"}, calls))
	})
	t.Run("retry_ctx_cancel", func(t *testing.T) {
		var (
			ctx, cancel = context.WithCancel(context.Background())
		)

		steps := []Step{
			NewStep("step0").
				WithAction(
					NewAction(func(ctx context.Context, track Track) error {
						data := track.GetData()
						if data.Action.Calls >= 2 {
							cancel() // cancel context for test
						}
						return testtool.ErrExpTestA
					}).WithRetry(NewBaseRetryOpt(10, 1*time.Nanosecond)),
				),
		}
		res, err := NewSaga(steps).Execute(ctx)
		assert.Error(t, err)

		assert.Equal(t, StageResultFail, res.Status)
		assert.Equal(t, 1, len(res.Steps))
		assert.Equal(t, "step0", res.Steps[0].StepName)
		assert.Equal(t, 0, res.Steps[0].StepPosition)
		assert.Equal(t, ExecutionStatusFail, res.Steps[0].Action.Status)
		assert.Equal(t, ExecutionStatusUnset, res.Steps[0].Compensation.Status)
		assert.Equal(t, 3, len(res.Steps[0].Action.Errors))
		assert.ErrorIs(t, res.Steps[0].Action.Errors[0], testtool.ErrExpTestA)
		assert.ErrorIs(t, res.Steps[0].Action.Errors[1], testtool.ErrExpTestA)
		assert.ErrorIs(t, res.Steps[0].Action.Errors[2], ErrRetryContextDone)
	})
	//
	t.Run("compensation_ctx_cancel", func(t *testing.T) {
		var (
			ctx, cancel = context.WithCancel(context.Background())
		)

		steps := []Step{
			NewStep("step0").
				WithAction(
					NewAction(func(ctx context.Context, _ Track) error {
						cancel() // cancel context for test
						return testtool.ErrExpTestA
					}),
				).WithCompensation(
				NewCompensation(func(ctx context.Context, _ Track) error {
					t.Fatalf("should not have been called")
					return nil
				}),
			).WithCompensationRequired(),
		}
		res, err := NewSaga(steps).Execute(ctx)
		assert.Error(t, err)
		assert.Equal(t, StageResultFail, res.Status)
		assert.Equal(t, 1, len(res.Steps))
		assert.Equal(t, "step0", res.Steps[0].StepName)
		assert.Equal(t, 0, res.Steps[0].StepPosition)
		assert.Equal(t, ExecutionStatusFail, res.Steps[0].Action.Status)
		assert.Equal(t, 1, res.Steps[0].Action.Calls)
		assert.Equal(t, 1, len(res.Steps[0].Action.Errors))
		assert.ErrorIs(t, res.Steps[0].Action.Errors[0], testtool.ErrExpTestA)
		assert.Equal(t, ExecutionStatusFail, res.Steps[0].Compensation.Status)
		assert.Equal(t, 0, res.Steps[0].Compensation.Calls)
		assert.Equal(t, 1, len(res.Steps[0].Compensation.Errors))
		assert.ErrorIs(t, res.Steps[0].Compensation.Errors[0], ErrExecuteCompensationContextDone)
	})
}

// nolint: dupl
func Test_hooks(t *testing.T) {
	t.Run("action_hooks", func(t *testing.T) {
		t.Run("before", func(t *testing.T) {
			var (
				ctx      = context.Background()
				executed = make([]string, 0, 3)
			)

			steps := []Step{
				NewStep("step1").
					WithAction(
						NewAction(func(ctx context.Context, _ Track) error {
							executed = append(executed, "action1")
							return testtool.ErrExpTestA
						}).WithBeforeHook(func(ctx context.Context, _ Track) error {
							executed = append(executed, "hook1")
							return nil
						}).WithBeforeHook(func(ctx context.Context, _ Track) error {
							executed = append(executed, "hook2")
							return nil
						}),
					),
			}

			res, err := NewSaga(steps).Execute(ctx)
			assert.Error(t, err)

			assert.Equal(t, StageResultFail, res.Status)
			assert.Equal(t, 1, len(res.Steps))
			assert.Equal(t, 1, res.Steps[0].Action.Calls)
			assert.Equal(t, ExecutionStatusFail, res.Steps[0].Action.Status)
			assert.Equal(t, 1, len(res.Steps[0].Action.Errors))
			assert.ErrorIs(t, res.Steps[0].Action.Errors[0], testtool.ErrExpTestA)

			assert.True(t, slices.Equal([]string{"hook2", "hook1", "action1"}, executed))
		})
		t.Run("before_with_retry", func(t *testing.T) {
			var (
				ctx      = context.Background()
				executed = make([]string, 0, 8)
			)

			steps := []Step{
				NewStep("step1").
					WithAction(
						NewAction(func(ctx context.Context, _ Track) error {
							executed = append(executed, "action1")
							return testtool.ErrExpTestA
						}).WithBeforeHook(func(ctx context.Context, _ Track) error {
							executed = append(executed, "hook1")
							return nil
						}).WithBeforeHook(func(ctx context.Context, _ Track) error {
							executed = append(executed, "hook2")
							return nil
						}).WithRetry(NewBaseRetryOpt(1, 1*time.Nanosecond)).
							WithBeforeHook(func(ctx context.Context, _ Track) error {
								executed = append(executed, "retry_hook1")
								return nil
							}).WithBeforeHook(
							func(ctx context.Context, _ Track) error {
								executed = append(executed, "retry_hook2")
								return nil
							}),
					),
			}

			res, err := NewSaga(steps).Execute(ctx)
			assert.Error(t, err)
			assert.Equal(t, StageResultFail, res.Status)
			assert.Equal(t, 1, len(res.Steps))
			assert.Equal(t, 2, res.Steps[0].Action.Calls)
			assert.Equal(t, ExecutionStatusFail, res.Steps[0].Action.Status)
			assert.Equal(t, 2, len(res.Steps[0].Action.Errors))
			assert.ErrorIs(t, res.Steps[0].Action.Errors[0], testtool.ErrExpTestA)
			assert.ErrorIs(t, res.Steps[0].Action.Errors[1], testtool.ErrExpTestA)
			assert.True(t,
				slices.Equal(
					[]string{
						"retry_hook2", "retry_hook1", // retry hooks
						"hook2", "hook1", "action1", // call
						"hook2", "hook1", "action1", // first retry
					},
					executed),
			)
		})

		var (
			errHook1 = fmt.Errorf("error_hook1")
			errHook2 = fmt.Errorf("error_hook2")
			errHook3 = fmt.Errorf("error_hook_3")
			errHook4 = fmt.Errorf("error_hook_4")
		)

		t.Run("after", func(t *testing.T) {
			var (
				ctx      = context.Background()
				executed = make([]string, 0, 3)
			)

			steps := []Step{
				NewStep("step1").
					WithAction(
						NewAction(func(ctx context.Context, track Track) error {
							executed = append(executed, "action1")
							return testtool.ErrExpTestA
						}).WithAfterHook(func(ctx context.Context, track Track) error {
							executed = append(executed, "hook1")

							data := track.GetData()
							assert.Equal(t, 1, data.Action.Calls)
							assert.Equal(t, 1, len(data.Action.Errors))
							assert.ErrorIs(t, data.Action.Errors[0], testtool.ErrExpTestA)

							data.Action.Errors = append(data.Action.Errors, errHook1)
							return errHook1
						}).WithAfterHook(func(ctx context.Context, track Track) error {
							executed = append(executed, "hook2")

							data := track.GetData()
							assert.Equal(t, 1, data.Action.Calls)
							assert.Equal(t, 2, len(data.Action.Errors))
							assert.ErrorIs(t, data.Action.Errors[0], testtool.ErrExpTestA)
							assert.ErrorIs(t, data.Action.Errors[1], errHook1)
							return errHook2
						}),
					),
			}
			res, err := NewSaga(steps).Execute(ctx)
			assert.Error(t, err)
			assert.Equal(t, StageResultFail, res.Status)
			assert.Equal(t, 1, len(res.Steps))
			assert.Equal(t, 1, res.Steps[0].Action.Calls)
			assert.Equal(t, ExecutionStatusFail, res.Steps[0].Action.Status)
			assert.Equal(t, 3, len(res.Steps[0].Action.Errors))
			assert.ErrorIs(t, res.Steps[0].Action.Errors[0], testtool.ErrExpTestA)
			assert.ErrorIs(t, res.Steps[0].Action.Errors[1], errHook1)
			assert.ErrorIs(t, res.Steps[0].Action.Errors[2], errHook2)

			assert.True(t, slices.Equal([]string{"action1", "hook1", "hook2"}, executed))

		})
		t.Run("after_with_retry___complicated_v1", func(t *testing.T) {
			var (
				ctx      = context.Background()
				executed = make([]string, 0, 11)

				checkRetryStr = func(i uint8, err error) bool {
					return strings.Contains(err.Error(), fmt.Sprintf("retry [%d]", i))
				}
			)

			steps := []Step{
				NewStep("step1").
					WithAction(
						NewAction(func(ctx context.Context, track Track) error {
							executed = append(executed, "action1")

							data := track.GetData()
							switch data.Action.Calls {
							case 1:
								return testtool.ErrExpTestA
							case 2:
								return testtool.ErrExpTestB
							case 3:
								return testtool.ErrExpTestC
							}
							t.Fatalf("should not have been called")
							return nil

						}).WithAfterHook(func(ctx context.Context, track Track) error {
							executed = append(executed, "hook1")

							data := track.GetData()
							switch data.Action.Calls {
							case 1:
								assert.Equal(t, 1, len(data.Action.Errors))
								assert.ErrorIs(t, data.Action.Errors[0], testtool.ErrExpTestA)
							case 2:
								assert.Equal(t, 4, len(data.Action.Errors))
								assert.ErrorIs(t, data.Action.Errors[0], testtool.ErrExpTestA)
								assert.ErrorIs(t, data.Action.Errors[1], errHook1)
								assert.ErrorIs(t, data.Action.Errors[2], errHook2)
								assert.ErrorIs(t, data.Action.Errors[3], testtool.ErrExpTestB)
								assert.True(t, checkRetryStr(0, data.Action.Errors[3]))
							case 3:
								assert.Equal(t, 7, len(data.Action.Errors))
								assert.ErrorIs(t, data.Action.Errors[0], testtool.ErrExpTestA)
								assert.ErrorIs(t, data.Action.Errors[1], errHook1)
								assert.ErrorIs(t, data.Action.Errors[2], errHook2)
								assert.ErrorIs(t, data.Action.Errors[3], testtool.ErrExpTestB)
								assert.True(t, checkRetryStr(0, data.Action.Errors[3]))
								assert.ErrorIs(t, data.Action.Errors[4], errHook1)
								assert.True(t, checkRetryStr(0, data.Action.Errors[4]))
								assert.ErrorIs(t, data.Action.Errors[5], errHook2)
								assert.True(t, checkRetryStr(0, data.Action.Errors[5]))
								assert.ErrorIs(t, data.Action.Errors[6], testtool.ErrExpTestC)
								assert.True(t, checkRetryStr(1, data.Action.Errors[6]))
							case 4:
								t.Fatalf("should not have been called")
							}
							return errHook1
						}).WithAfterHook(func(ctx context.Context, track Track) error {
							executed = append(executed, "hook2")

							data := track.GetData()
							switch data.Action.Calls {
							case 1:
								assert.Equal(t, 2, len(data.Action.Errors))
								assert.ErrorIs(t, data.Action.Errors[0], testtool.ErrExpTestA)
								assert.ErrorIs(t, data.Action.Errors[1], errHook1)
							case 2:
								assert.Equal(t, 5, len(data.Action.Errors))
								assert.ErrorIs(t, data.Action.Errors[0], testtool.ErrExpTestA)
								assert.ErrorIs(t, data.Action.Errors[1], errHook1)
								assert.ErrorIs(t, data.Action.Errors[2], errHook2)
								assert.ErrorIs(t, data.Action.Errors[3], testtool.ErrExpTestB)
								assert.ErrorIs(t, data.Action.Errors[4], errHook1)
							case 3:
								assert.Equal(t, 8, len(data.Action.Errors))
								assert.ErrorIs(t, data.Action.Errors[0], testtool.ErrExpTestA)
								assert.ErrorIs(t, data.Action.Errors[1], errHook1)
								assert.ErrorIs(t, data.Action.Errors[2], errHook2)
								assert.ErrorIs(t, data.Action.Errors[3], testtool.ErrExpTestB)
								assert.True(t, checkRetryStr(0, data.Action.Errors[3]))
								assert.ErrorIs(t, data.Action.Errors[4], errHook1)
								assert.True(t, checkRetryStr(0, data.Action.Errors[4]))
								assert.ErrorIs(t, data.Action.Errors[5], errHook2)
								assert.True(t, checkRetryStr(0, data.Action.Errors[5]))
								assert.ErrorIs(t, data.Action.Errors[6], testtool.ErrExpTestC)
								assert.True(t, checkRetryStr(1, data.Action.Errors[6]))
								assert.ErrorIs(t, data.Action.Errors[7], errHook1)
								assert.True(t, checkRetryStr(1, data.Action.Errors[7]))
							case 4:
								t.Fatalf("should not have been called")
							}

							return errHook2
						}).WithRetry(NewBaseRetryOpt(2, 1*time.Nanosecond)).
							WithAfterHook(func(ctx context.Context, track Track) error {
								executed = append(executed, "retry_hook1")

								data := track.GetData()
								assert.Equal(t, 3, data.Action.Calls)
								assert.Equal(t, 9, len(data.Action.Errors))
								assert.ErrorIs(t, data.Action.Errors[8], errHook2)
								assert.True(t, checkRetryStr(1, data.Action.Errors[8]))
								return errHook3
							}).
							WithAfterHook(func(ctx context.Context, track Track) error {
								executed = append(executed, "retry_hook2")

								data := track.GetData()
								assert.Equal(t, 3, data.Action.Calls)
								assert.Equal(t, 10, len(data.Action.Errors))
								assert.ErrorIs(t, data.Action.Errors[8], errHook2)
								assert.True(t, checkRetryStr(1, data.Action.Errors[8]))
								assert.ErrorIs(t, data.Action.Errors[9], errHook3)
								return errHook4
							}),
					),
			}
			res, err := NewSaga(steps).Execute(ctx)
			assert.Error(t, err)
			assert.Equal(t, StageResultFail, res.Status)

			testtool.TestFn(t, func() {
				t.Logf("\nresult:\n%v", res)
				t.Logf("\nexecution error: %v", err)
				step := res.Steps[0]
				t.Logf("\nstep [%d#%s] action errors:", step.StepPosition, step.StepName)
				for i, e := range res.Steps[0].Action.Errors {
					fmt.Printf("%d: %v\n", i, e)
				}
			})

			assert.True(t,
				slices.Equal(
					[]string{
						"action1", "hook1", "hook2", // call
						"action1", "hook1", "hook2", // first retry
						"action1", "hook1", "hook2", // second retry
						"retry_hook1", "retry_hook2", // retry hooks
					},
					executed),
			)

			assert.Equal(t, 1, len(res.Steps))

			action := res.Steps[0].Action
			assert.Equal(t, 3, res.Steps[0].Action.Calls)
			assert.Equal(t, ExecutionStatusFail, action.Status)
			assert.Equal(t, 11, len(action.Errors))
			assert.ErrorIs(t, action.Errors[0], testtool.ErrExpTestA)
			assert.ErrorIs(t, action.Errors[1], errHook1)
			assert.ErrorIs(t, action.Errors[2], errHook2)
			assert.ErrorIs(t, action.Errors[3], testtool.ErrExpTestB)
			assert.True(t, checkRetryStr(0, action.Errors[3]))
			assert.ErrorIs(t, action.Errors[4], errHook1)
			assert.True(t, checkRetryStr(0, action.Errors[4]))
			assert.ErrorIs(t, action.Errors[5], errHook2)
			assert.True(t, checkRetryStr(0, action.Errors[5]))
			assert.ErrorIs(t, action.Errors[6], testtool.ErrExpTestC)
			assert.True(t, checkRetryStr(1, action.Errors[6]))
			assert.ErrorIs(t, action.Errors[7], errHook1)
			assert.True(t, checkRetryStr(1, action.Errors[7]))
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
						NewAction(func(ctx context.Context, _ Track) error {
							executed = append(executed, "action1")
							return testtool.ErrExpTestA
						}),
					).WithCompensation(
					NewCompensation(func(ctx context.Context, track Track) error {
						executed = append(executed, "comp1")

						data := track.GetData()
						assert.Equal(t, 1, data.Action.Calls)
						assert.Equal(t, 1, data.Compensation.Calls)
						assert.Equal(t, 1, len(data.Action.Errors))
						assert.ErrorIs(t, data.Action.Errors[0], testtool.ErrExpTestA)

						return nil
					}).WithBeforeHook(func(ctx context.Context, track Track) error {
						executed = append(executed, "comp_hook1")

						data := track.GetData()
						assert.Equal(t, 1, data.Action.Calls)
						assert.Equal(t, 1, data.Compensation.Calls)
						assert.Equal(t, 1, len(data.Action.Errors))
						assert.ErrorIs(t, data.Action.Errors[0], testtool.ErrExpTestA)
						return nil
					}).WithBeforeHook(func(ctx context.Context, track Track) error {
						executed = append(executed, "comp_hook2")

						data := track.GetData()
						assert.Equal(t, 1, data.Action.Calls)
						assert.Equal(t, 1, data.Compensation.Calls)
						assert.Equal(t, 1, len(data.Action.Errors))
						assert.ErrorIs(t, data.Action.Errors[0], testtool.ErrExpTestA)
						return nil
					}),
				).WithCompensationRequired(),
			}
			res, err := NewSaga(steps).Execute(ctx)
			assert.Error(t, err)
			assert.Equal(t, StageResultCompensated, res.Status)

			assert.True(t, slices.Equal([]string{"action1", "comp_hook2", "comp_hook1", "comp1"}, executed))

			assert.Equal(t, 1, len(res.Steps))

			action := res.Steps[0].Action
			assert.Equal(t, ExecutionStatusFail, action.Status)
			assert.Equal(t, 1, len(action.Errors))
			assert.ErrorIs(t, action.Errors[0], testtool.ErrExpTestA)

			compensation := res.Steps[0].Compensation
			assert.Equal(t, 1, compensation.Calls)
			assert.Equal(t, ExecutionStatusSuccess, compensation.Status)
			assert.Equal(t, ExecutionStatusSuccess, compensation.Status)
			assert.Equal(t, 0, len(compensation.Errors))

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
						NewAction(func(ctx context.Context, track Track) error {
							executed = append(executed, "action1")
							return testtool.ErrExpTestA
						}),
					).WithCompensation(
					NewCompensation(func(ctx context.Context, track Track) error {
						executed = append(executed, "comp1")
						return compErr
					}).WithAfterHook(func(ctx context.Context, track Track) error {
						executed = append(executed, "comp_hook1")

						data := track.GetData()
						assert.Equal(t, 1, data.Action.Calls)
						assert.Equal(t, 1, data.Compensation.Calls)
						assert.Equal(t, 1, len(data.Action.Errors))
						assert.ErrorIs(t, data.Action.Errors[0], testtool.ErrExpTestA)
						assert.Equal(t, 1, len(data.Compensation.Errors))
						assert.ErrorIs(t, data.Compensation.Errors[0], compErr)

						return previousHookErr
					}).WithAfterHook(func(ctx context.Context, track Track) error {
						executed = append(executed, "comp_hook2")

						data := track.GetData()
						assert.Equal(t, 1, data.Action.Calls)
						assert.Equal(t, 1, data.Compensation.Calls)
						assert.Equal(t, 1, len(data.Action.Errors))
						assert.ErrorIs(t, data.Action.Errors[0], testtool.ErrExpTestA)
						assert.Equal(t, 2, len(data.Compensation.Errors))
						assert.ErrorIs(t, data.Compensation.Errors[0], compErr)
						assert.Equal(t, 2, len(data.Compensation.Errors))
						assert.ErrorIs(t, data.Compensation.Errors[1], previousHookErr)
						return nil
					}),
				).WithCompensationRequired(),
			}

			res, err := NewSaga(steps).Execute(ctx)
			assert.Error(t, err)
			assert.Equal(t, StageResultCompensated, res.Status)

			assert.True(t, slices.Equal([]string{"action1", "comp1", "comp_hook1", "comp_hook2"}, executed))

		})
	})
}

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
		assert.Equal(t, 1, res.Steps[0].Action.Calls)
		assert.Equal(t, ExecutionStatusSuccess, res.Steps[0].Action.Status)
		assert.Equal(t, 0, res.Steps[0].Compensation.Calls)
		assert.Equal(t, ExecutionStatusUnset, res.Steps[0].Compensation.Status)

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
			).WithCompensationRequired(),
		}

		res, err := NewSaga(steps).Execute(ctx)
		assert.Error(t, err)
		assert.Equal(t, StageResultCompensated, res.Status)
		assert.Equal(t, 1, res.Steps[0].Action.Calls)
		assert.Equal(t, 1, len(res.Steps[0].Action.Errors))
		assert.Equal(t, ExecutionStatusFail, res.Steps[0].Action.Status)
		assert.Equal(t, 1, res.Steps[0].Compensation.Calls)
		assert.Equal(t, ExecutionStatusSuccess, res.Steps[0].Compensation.Status)
		assert.Equal(t, 0, len(res.Steps[0].Compensation.Errors))

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

			assert.Equal(t, 1, len(res.Steps))

			assert.Equal(t, "step1", res.Steps[0].StepName)
			assert.Equal(t, 0, res.Steps[0].StepPosition)
			assert.Equal(t, ExecutionStatusSuccess, res.Steps[0].Action.Status)
			assert.Equal(t, 1, res.Steps[0].Action.Calls)
			assert.Equal(t, 0, len(res.Steps[0].Action.Errors))

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
			assert.Equal(t, 1, len(res.Steps))
			assert.Equal(t, "step1", res.Steps[0].StepName)
			assert.Equal(t, 0, res.Steps[0].StepPosition)
			assert.Equal(t, ExecutionStatusFail, res.Steps[0].Action.Status)
			assert.Equal(t, ExecutionStatusUnset, res.Steps[0].Compensation.Status)
			assert.Equal(t, 1, res.Steps[0].Action.Calls)
			assert.Equal(t, 1, len(res.Steps[0].Action.Errors))
			assert.ErrorIs(t, res.Steps[0].Action.Errors[0], testtool.ErrExpTestA)

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
			assert.Equal(t, StageResultCompensated, res.Status)
			assert.True(t, errors.Is(err, ErrActionFailed))

			assert.Equal(t, 2, len(res.Steps))

			assert.Equal(t, "step1", res.Steps[0].StepName)
			assert.Equal(t, 0, res.Steps[0].StepPosition)
			assert.Equal(t, ExecutionStatusSuccess, res.Steps[0].Action.Status)
			assert.Equal(t, ExecutionStatusUnset, res.Steps[0].Compensation.Status)
			assert.Equal(t, false, res.Steps[0].CompensationRequired)
			assert.Equal(t, 1, res.Steps[0].Action.Calls)
			assert.Equal(t, 0, len(res.Steps[0].Action.Errors))

			assert.Equal(t, "step2", res.Steps[1].StepName)
			assert.Equal(t, 1, res.Steps[1].StepPosition)
			assert.Equal(t, ExecutionStatusFail, res.Steps[1].Action.Status)
			assert.Equal(t, ExecutionStatusUnset, res.Steps[1].Compensation.Status)
			assert.Equal(t, false, res.Steps[1].CompensationRequired)
			assert.Equal(t, 1, res.Steps[1].Action.Calls)
			assert.Equal(t, 1, len(res.Steps[1].Action.Errors))
			assert.ErrorIs(t, res.Steps[1].Action.Errors[0], testtool.ErrExpTestA)

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
					).WithCompensationRequired(),
			}

			res, err := NewSaga(steps).Execute(ctx)
			assert.Error(t, err)
			assert.Equal(t, StageResultCompensated, res.Status)
			assert.True(t, errors.Is(err, ErrActionFailed))

			assert.Equal(t, 1, len(res.Steps))

			assert.Equal(t, "step1", res.Steps[0].StepName)
			assert.Equal(t, 0, res.Steps[0].StepPosition)
			assert.Equal(t, ExecutionStatusFail, res.Steps[0].Action.Status)
			assert.Equal(t, 1, res.Steps[0].Action.Calls)
			assert.Equal(t, 1, len(res.Steps[0].Action.Errors))

			assert.Equal(t, ExecutionStatusSuccess, res.Steps[0].Compensation.Status)
			assert.Equal(t, 1, res.Steps[0].Compensation.Calls)
			assert.Equal(t, 0, len(res.Steps[0].Compensation.Errors))

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
							assert.ErrorIs(t, str.Action.Errors[0], testtool.ErrExpTestA)

							assert.Equal(t, ExecutionStatusUncalled, str.Compensation.Status)
							assert.Equal(t, 1, str.Compensation.Calls)
							assert.Equal(t, 0, len(str.Compensation.Errors))
							return nil
						}),
					).WithCompensationRequired(),
			}

			res, err := NewSaga(steps).Execute(ctx)
			assert.Error(t, err)
			assert.Equal(t, StageResultCompensated, res.Status)
			assert.True(t, errors.Is(err, ErrActionFailed))

			assert.Equal(t, 1, len(res.Steps))

			assert.Equal(t, "step1", res.Steps[0].StepName)
			assert.Equal(t, 0, res.Steps[0].StepPosition)
			assert.Equal(t, ExecutionStatusFail, res.Steps[0].Action.Status)
			assert.Equal(t, 1, res.Steps[0].Action.Calls)
			assert.Equal(t, 1, len(res.Steps[0].Action.Errors))

			assert.Equal(t, ExecutionStatusSuccess, res.Steps[0].Compensation.Status)
			assert.Equal(t, 1, res.Steps[0].Compensation.Calls)
			assert.Equal(t, 0, len(res.Steps[0].Compensation.Errors))

			testtool.TestFn(t, func() {
				t.Log(res)
			})
		})
		t.Run("compensate_v2", func(t *testing.T) {
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
					).
					WithCompensation(
						NewCompensation(func(ctx context.Context, track Track) error {
							return nil
						}),
					),
			}

			res, err := NewSaga(steps).Execute(ctx)
			assert.Error(t, err)
			assert.Equal(t, StageResultCompensated, res.Status)
			assert.True(t, errors.Is(err, ErrActionFailed))

			assert.Equal(t, 2, len(res.Steps))

			assert.Equal(t, "step1", res.Steps[0].StepName)
			assert.Equal(t, 0, res.Steps[0].StepPosition)
			assert.Equal(t, ExecutionStatusSuccess, res.Steps[0].Action.Status)
			assert.Equal(t, ExecutionStatusUnset, res.Steps[0].Compensation.Status)
			assert.Equal(t, false, res.Steps[0].CompensationRequired)
			assert.Equal(t, 1, res.Steps[0].Action.Calls)
			assert.Equal(t, 0, len(res.Steps[0].Action.Errors))

			assert.Equal(t, "step2", res.Steps[1].StepName)
			assert.Equal(t, 1, res.Steps[1].StepPosition)
			assert.Equal(t, ExecutionStatusFail, res.Steps[1].Action.Status)
			assert.Equal(t, ExecutionStatusUncalled, res.Steps[1].Compensation.Status)
			assert.Equal(t, false, res.Steps[1].CompensationRequired)
			assert.Equal(t, 1, res.Steps[1].Action.Calls)
			assert.Equal(t, 1, len(res.Steps[1].Action.Errors))
			assert.ErrorIs(t, res.Steps[1].Action.Errors[0], testtool.ErrExpTestA)

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
							assert.ErrorIs(t, str.Action.Errors[0], testtool.ErrExpTestA)

							assert.Equal(t, ExecutionStatusUncalled, str.Compensation.Status)
							assert.Equal(t, 1, str.Compensation.Calls)
							assert.Equal(t, 0, len(str.Compensation.Errors))
							return testtool.ErrExpTestB
						}),
					).WithCompensationRequired(),
			}

			res, err := NewSaga(steps).Execute(ctx)
			assert.Error(t, err)
			assert.Equal(t, StageResultFail, res.Status)
			assert.ErrorIs(t, err, ErrActionFailed)
			assert.ErrorIs(t, err, ErrCompensationFailed)

			assert.Equal(t, 1, len(res.Steps))

			assert.Equal(t, "step1", res.Steps[0].StepName)
			assert.Equal(t, 0, res.Steps[0].StepPosition)
			assert.Equal(t, ExecutionStatusFail, res.Steps[0].Action.Status)
			assert.Equal(t, 1, res.Steps[0].Action.Calls)
			assert.Equal(t, 1, len(res.Steps[0].Action.Errors))

			assert.Equal(t, ExecutionStatusFail, res.Steps[0].Compensation.Status)
			assert.Equal(t, 1, res.Steps[0].Compensation.Calls)
			assert.Equal(t, 1, len(res.Steps[0].Compensation.Errors))
			assert.ErrorIs(t, res.Steps[0].Compensation.Errors[0], testtool.ErrExpTestB)

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
		var (
			retries = uint32(4)
		)
		steps := []Step{
			NewStep("step1").
				WithAction(
					NewAction(func(ctx context.Context, _ Track) error {
						return testtool.ErrExpTestA
					}).
						WithRetry(NewBaseRetryOpt(retries, 5*time.Nanosecond)),
				).
				WithCompensation(
					NewCompensation(func(ctx context.Context, track Track) error {
						str := track.GetData()
						if str.Compensation.Calls < retries+1 {
							return fmt.Errorf("comp err [%d]: %w", len(str.Compensation.Errors), testtool.ErrExpTestA)
						}
						return nil
					}).WithRetry(NewBaseRetryOpt(retries, 5*time.Nanosecond)),
				).WithCompensationRequired(),
		}

		res, err := NewSaga(steps).Execute(ctx)
		assert.Error(t, err)
		assert.Equal(t, StageResultCompensated, res.Status)
		assert.ErrorIs(t, err, ErrActionFailed)
		assert.ErrorIsNot(t, err, ErrCompensationFailed)

		assert.Equal(t, 1, len(res.Steps))

		assert.Equal(t, "step1", res.Steps[0].StepName)
		assert.Equal(t, 0, res.Steps[0].StepPosition)
		assert.Equal(t, ExecutionStatusFail, res.Steps[0].Action.Status)
		assert.Equal(t, 5, res.Steps[0].Action.Calls)
		assert.Equal(t, 5, len(res.Steps[0].Action.Errors))

		for _, e := range res.Steps[0].Action.Errors {
			assert.ErrorIs(t, e, testtool.ErrExpTestA)
		}

		assert.Equal(t, ExecutionStatusSuccess, res.Steps[0].Compensation.Status)
		assert.Equal(t, 5, res.Steps[0].Compensation.Calls)
		assert.Equal(t, 4, len(res.Steps[0].Compensation.Errors))
		for _, e := range res.Steps[0].Compensation.Errors {
			assert.ErrorIs(t, e, testtool.ErrExpTestA)
		}

		testtool.TestFn(t, func() {
			t.Log(
				res,
				"+ error:", err,
				"\n + Action errors: ", res.Steps[0].Action.Errors,
				"\n + Compensation errors: ", res.Steps[0].Compensation.Errors,
			)
		})

	})
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
						return fmt.Errorf("comp err [%d]: %w", len(str.Compensation.Errors), testtool.ErrExpTestB)
					}).WithRetry(NewBaseRetryOpt(4, 5*time.Nanosecond)),
				).WithCompensationRequired(),
		}

		res, err := NewSaga(steps).Execute(ctx)
		assert.Error(t, err)
		assert.Equal(t, StageResultFail, res.Status)
		assert.ErrorIs(t, err, ErrActionFailed)
		assert.ErrorIs(t, err, ErrCompensationFailed)

		assert.Equal(t, 1, len(res.Steps))

		assert.Equal(t, "step1", res.Steps[0].StepName)
		assert.Equal(t, 0, res.Steps[0].StepPosition)
		assert.Equal(t, ExecutionStatusFail, res.Steps[0].Action.Status)
		assert.Equal(t, 5, res.Steps[0].Action.Calls)
		assert.Equal(t, 5, len(res.Steps[0].Action.Errors))

		for _, e := range res.Steps[0].Action.Errors {
			assert.ErrorIs(t, e, testtool.ErrExpTestA)
		}

		assert.Equal(t, ExecutionStatusFail, res.Steps[0].Compensation.Status)
		assert.Equal(t, 5, res.Steps[0].Compensation.Calls)
		assert.Equal(t, 5, len(res.Steps[0].Compensation.Errors))
		for _, e := range res.Steps[0].Compensation.Errors {
			assert.ErrorIs(t, e, testtool.ErrExpTestB)
		}

		testtool.TestFn(t, func() {
			t.Log(
				res,
				"+ error:", err,
				"\n + Action errors: ", res.Steps[0].Action.Errors,
				"\n + Compensation errors: ", res.Steps[0].Compensation.Errors,
			)
		})

	})
}
