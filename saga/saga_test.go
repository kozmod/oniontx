//nolint:dupl,goconst
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
	"github.com/kozmod/oniontx/internal/testtool/require"
)

var _ Track = readOnlyTrack{}

type readOnlyTrack struct{}

func (readOnlyTrack) GetStepData() StepData { return StepData{} }

func (readOnlyTrack) GetTrackData() TrackData { return TrackData{} }

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
			NewStep("step0").
				WithAction(NewOperation(func(ctx context.Context, _ Track) error {
					executedActions = append(executedActions, "action1")
					return nil
				})).
				WithCompensation(NewOperation(func(ctx context.Context, _ Track) error {
					executedCompensation = append(executedCompensation, "comp1")
					t.Fatalf("should not have been called")
					return nil
				})),
			NewStep("step1").
				WithAction(NewOperation(func(ctx context.Context, _ Track) error {
					executedActions = append(executedActions, "action2")
					return nil
				})).
				WithCompensation(NewOperation(func(ctx context.Context, _ Track) error {
					executedCompensation = append(executedCompensation, "comp2")
					t.Fatalf("should not have been called")
					return nil
				})),
		}

		res, err := NewSaga(steps).Execute(ctx)
		require.NoError(t, err)
		require.Equal(t, StageResultSuccess, res.Status)
		require.Equal(t, 2, len(res.Steps))

		require.Equal(t, "step0", res.Steps[0].StepName)
		require.Equal(t, 0, res.Steps[0].StepPosition)
		require.Equal(t, 1, res.Steps[0].Action.Calls)
		require.Equal(t, ExecutionStatusSuccess, res.Steps[0].Action.Status)
		require.Equal(t, 0, res.Steps[0].Compensation.Calls)

		require.Equal(t, "step1", res.Steps[1].StepName)
		require.Equal(t, 1, res.Steps[1].StepPosition)
		require.Equal(t, 1, res.Steps[1].Action.Calls)
		require.Equal(t, ExecutionStatusSuccess, res.Steps[1].Action.Status)
		require.Equal(t, 0, res.Steps[1].Compensation.Calls)

		require.True(t, slices.Equal([]string{"action1", "action2"}, executedActions))
		require.True(t, len(executedCompensation) == 0)
	})

	t.Run("success_compensation_on_step1", func(t *testing.T) {
		var (
			executedActions      []string
			executedCompensation []string
		)

		steps := []Step{
			NewStep("step0").
				WithAction(NewOperation(func(ctx context.Context, _ Track) error {
					executedActions = append(executedActions, "action1")
					return nil
				})).
				WithCompensation(NewOperation(func(ctx context.Context, _ Track) error {
					executedCompensation = append(executedCompensation, "comp1")
					return nil
				})),
			NewStep("step1").
				WithAction(NewOperation(func(ctx context.Context, _ Track) error {
					executedActions = append(executedActions, "action2")
					return testtool.ErrExpTestA
				})).
				WithCompensation(NewOperation(func(ctx context.Context, _ Track) error {
					executedCompensation = append(executedCompensation, "comp2")
					t.Fatalf("should not have been called")
					return nil
				})),
		}

		res, err := NewSaga(steps).Execute(ctx)
		require.Error(t, err)
		require.Equal(t, StageResultCompensated, res.Status)
		require.Equal(t, 2, len(res.Steps))

		require.Equal(t, "step0", res.Steps[0].StepName)
		require.Equal(t, 0, res.Steps[0].StepPosition)
		require.Equal(t, 1, res.Steps[0].Action.Calls)
		require.Equal(t, 1, res.Steps[0].Compensation.Calls)
		require.Equal(t, ExecutionStatusSuccess, res.Steps[0].Compensation.Status)

		require.Equal(t, "step1", res.Steps[1].StepName)
		require.Equal(t, 1, res.Steps[1].StepPosition)
		require.Equal(t, 1, res.Steps[1].Action.Calls)
		require.Equal(t, 0, res.Steps[1].Compensation.Calls)

		require.True(t, slices.Equal([]string{"action1", "action2"}, executedActions))
		require.True(t, slices.Equal([]string{"comp1"}, executedCompensation))
	})

	t.Run("compensation_on_fail", func(t *testing.T) {
		t.Run("failed_compensation_for_completed_step", func(t *testing.T) {
			steps := []Step{
				NewStep("completed_step").
					WithAction(NewOperation(func(context.Context, Track) error {
						return nil
					})).
					WithCompensation(NewOperation(func(context.Context, Track) error {
						return testtool.ErrExpTestB
					})),
				NewStep("failed_step").
					WithAction(NewOperation(func(context.Context, Track) error {
						return testtool.ErrExpTestA
					})),
			}

			res, err := NewSaga(steps).Execute(ctx)

			require.Error(t, err)
			require.ErrorIs(t, err, ErrActionFailed)
			require.ErrorIs(t, err, ErrCompensationFailed)
			require.Equal(t, StageResultFail, res.Status)
			require.Equal(t, ExecutionStatusSuccess, res.Steps[0].Action.Status)
			require.Equal(t, ExecutionStatusFail, res.Steps[0].Compensation.Status)
			require.ErrorIs(t, res.Steps[0].Compensation.Errors[0], testtool.ErrExpTestB)
			require.Equal(t, ExecutionStatusFail, res.Steps[1].Action.Status)
		})

		t.Run("skipped", func(t *testing.T) {
			var (
				executedActions      []string
				executedCompensation []string
			)

			steps := []Step{
				NewStep("step0").
					WithAction(NewOperation(func(ctx context.Context, _ Track) error {
						executedActions = append(executedActions, "action1")
						return testtool.ErrExpTestA
					})).
					WithCompensation(NewOperation(func(ctx context.Context, _ Track) error {
						executedCompensation = append(executedCompensation, "comp1")
						t.Fatalf("should not have been called")
						return nil
					})),
			}

			res, err := NewSaga(steps).Execute(ctx)
			require.Error(t, err)
			require.Equal(t, StageResultFail, res.Status)
			require.Equal(t, 1, len(res.Steps))

			require.Equal(t, "step0", res.Steps[0].StepName)
			require.Equal(t, 0, res.Steps[0].StepPosition)
			require.Equal(t, 1, res.Steps[0].Action.Calls)
			require.Equal(t, 1, len(res.Steps[0].Action.Errors))
			require.ErrorIs(t, res.Steps[0].Action.Errors[0], testtool.ErrExpTestA)
			require.Equal(t, ExecutionStatusFail, res.Steps[0].Action.Status)
			require.Equal(t, 0, res.Steps[0].Compensation.Calls)
			require.Equal(t, ExecutionStatusUncalled, res.Steps[0].Compensation.Status)

			require.True(t, slices.Equal([]string{"action1"}, executedActions))
			require.True(t, len(executedCompensation) == 0)
		})
		t.Run("added", func(t *testing.T) {
			var (
				executedActions      []string
				executedCompensation []string
			)

			steps := []Step{
				NewStep("step0").
					WithAction(NewOperation(func(ctx context.Context, _ Track) error {
						executedActions = append(executedActions, "action1")
						return testtool.ErrExpTestA
					})).
					WithCompensation(NewOperation(func(ctx context.Context, _ Track) error {
						executedCompensation = append(executedCompensation, "comp1")
						return nil
					})).
					WithCompensationOnActionFailure(),
			}

			res, err := NewSaga(steps).Execute(ctx)
			require.Error(t, err)
			require.Equal(t, StageResultCompensated, res.Status)
			require.Equal(t, 1, len(res.Steps))

			require.Equal(t, "step0", res.Steps[0].StepName)
			require.Equal(t, 0, res.Steps[0].StepPosition)
			require.Equal(t, 1, res.Steps[0].Action.Calls)
			require.Equal(t, 1, len(res.Steps[0].Action.Errors))
			require.ErrorIs(t, res.Steps[0].Action.Errors[0], testtool.ErrExpTestA)
			require.Equal(t, ExecutionStatusFail, res.Steps[0].Action.Status)
			require.Equal(t, 1, res.Steps[0].Compensation.Calls)
			require.Equal(t, ExecutionStatusSuccess, res.Steps[0].Compensation.Status)

			require.True(t, slices.Equal([]string{"action1"}, executedActions))
			require.True(t, slices.Equal([]string{"comp1"}, executedCompensation))
		})

		t.Run("reverse_order", func(t *testing.T) {
			var executedCompensation []string

			steps := []Step{
				NewStep("step0").
					WithAction(NewOperation(func(ctx context.Context, _ Track) error {
						return nil
					})).
					WithCompensation(NewOperation(func(ctx context.Context, _ Track) error {
						executedCompensation = append(executedCompensation, "comp0")
						return nil
					})),
				NewStep("step1").
					WithAction(NewOperation(func(ctx context.Context, _ Track) error {
						return nil
					})).
					WithCompensation(NewOperation(func(ctx context.Context, _ Track) error {
						executedCompensation = append(executedCompensation, "comp1")
						return nil
					})),
				NewStep("step2").
					WithAction(NewOperation(func(ctx context.Context, _ Track) error {
						return nil
					})).
					WithCompensation(NewOperation(func(ctx context.Context, _ Track) error {
						executedCompensation = append(executedCompensation, "comp2")
						return nil
					})),
				NewStep("step3").
					WithAction(NewOperation(func(ctx context.Context, _ Track) error {
						return testtool.ErrExpTestA
					})),
			}

			res, err := NewSaga(steps).Execute(ctx)
			require.Error(t, err)
			require.Equal(t, StageResultCompensated, res.Status)
			require.True(t, slices.Equal([]string{"comp2", "comp1", "comp0"}, executedCompensation))
		})

		t.Run("required_without_compensation", func(t *testing.T) {
			steps := []Step{
				NewStep("step0").
					WithAction(NewOperation(func(ctx context.Context, _ Track) error {
						return testtool.ErrExpTestA
					})).
					WithCompensationOnActionFailure(),
			}

			res, err := NewSaga(steps).Execute(ctx)
			require.Error(t, err)
			require.ErrorIs(t, err, ErrActionFailed)
			require.ErrorIs(t, err, ErrCompensationFailed)
			require.Equal(t, StageResultFail, res.Status)
			require.Equal(t, 1, len(res.Steps[0].Compensation.Errors))
			require.ErrorIs(t, res.Steps[0].Compensation.Errors[0], ErrCompensationNotConfigured)
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
				NewStep("step0").
					WithAction(NewOperation(func(ctx context.Context, _ Track) error {
						panic("panic_v1!")
					}).WithPanicRecovery()),
			}

			res, err := NewSaga(steps).Execute(ctx)
			require.Error(t, err)
			require.Equal(t, StageResultFail, res.Status)
			require.Equal(t, 1, len(res.Steps))
			require.Equal(t, ExecutionStatusFail, res.Steps[0].Action.Status)
			require.Equal(t, 1, len(res.Steps[0].Action.Errors))
			require.ErrorIs(t, res.Steps[0].Action.Errors[0], ErrPanicRecovered)
			require.Equal(t, ExecutionStatusUnset, res.Steps[0].Compensation.Status)
			require.Equal(t, 0, res.Steps[0].Compensation.Calls)
			require.Equal(t, 0, len(res.Steps[0].Compensation.Errors))

		})
	})
	t.Run("builder_stile", func(t *testing.T) {
		t.Run("success_Operation", func(t *testing.T) {
			steps := []Step{
				NewStep("step0").
					WithAction(
						NewOperation(func(ctx context.Context, _ Track) error {
							panic("panic_v2!")
						}).WithPanicRecovery(),
					),
			}

			res, err := NewSaga(steps).Execute(ctx)
			require.Error(t, err)
			require.Equal(t, StageResultFail, res.Status)
			require.Equal(t, 1, len(res.Steps))
			require.Equal(t, ExecutionStatusFail, res.Steps[0].Action.Status)
			require.Equal(t, 1, len(res.Steps[0].Action.Errors))
			require.ErrorIs(t, res.Steps[0].Action.Errors[0], ErrPanicRecovered)
			require.Equal(t, ExecutionStatusUnset, res.Steps[0].Compensation.Status)
			require.Equal(t, 0, res.Steps[0].Compensation.Calls)
			require.Equal(t, 0, len(res.Steps[0].Compensation.Errors))
		})

		t.Run("success_OperationFunc", func(t *testing.T) {
			steps := []Step{
				NewStep("step0").
					WithAction(NewOperation(func(ctx context.Context, _ Track) error {
						return testtool.ErrExpTestA
					})).
					WithCompensation(NewOperation(func(ctx context.Context, track Track) error {
						str := track.GetStepData()
						require.Equal(t, 1, len(str.Action.Errors))
						require.Equal(t, 1, str.Action.Calls)
						require.Equal(t, ExecutionStatusFail, str.Action.Status)
						require.Error(t, str.Action.Errors[0])
						require.ErrorIs(t, str.Action.Errors[0], testtool.ErrExpTestA)

						panic("panic_v3!")
					}).WithPanicRecovery()).
					WithCompensationOnActionFailure(),
			}

			res, err := NewSaga(steps).Execute(ctx)
			require.Error(t, err)
			require.Equal(t, StageResultFail, res.Status)
			require.Equal(t, 1, len(res.Steps))
			require.Equal(t, ExecutionStatusFail, res.Steps[0].Action.Status)
			require.Equal(t, 1, len(res.Steps[0].Action.Errors))
			require.ErrorIs(t, res.Steps[0].Action.Errors[0], testtool.ErrExpTestA)
			require.Equal(t, ExecutionStatusFail, res.Steps[0].Compensation.Status)
			require.Equal(t, 1, res.Steps[0].Compensation.Calls)
			require.Equal(t, 1, len(res.Steps[0].Compensation.Errors))
			require.ErrorIs(t, res.Steps[0].Compensation.Errors[0], ErrPanicRecovered)
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
				WithAction(NewOperation(func(ctx context.Context, _ Track) error {
					return nil
				})).
				WithCompensation(NewOperation(func(ctx context.Context, _ Track) error {
					t.Fatalf("should not have been called")
					return nil
				})),
			NewStep("step1").
				WithAction(NewOperation(func(ctx context.Context, _ Track) error {
					return nil
				})).
				WithCompensation(NewOperation(func(ctx context.Context, _ Track) error {
					t.Fatalf("should not have been called")
					return nil
				})),
		}

		res, err := NewSaga(steps).Execute(ctx)
		require.NoError(t, err)
		require.Equal(t, StageResultSuccess, res.Status)
		require.Equal(t, 2, len(res.Steps))
		require.Equal(t, ExecutionStatusSuccess, res.Steps[0].Action.Status)
		require.Equal(t, 1, res.Steps[0].Action.Calls)
		require.Equal(t, 0, len(res.Steps[0].Action.Errors))
		require.Equal(t, ExecutionStatusUncalled, res.Steps[0].Compensation.Status)
		require.Equal(t, 0, res.Steps[0].Compensation.Calls)
		require.Equal(t, 0, len(res.Steps[0].Compensation.Errors))

		require.Equal(t, ExecutionStatusSuccess, res.Steps[1].Action.Status)
		require.Equal(t, 1, res.Steps[1].Action.Calls)
		require.Equal(t, 0, len(res.Steps[1].Action.Errors))
		require.Equal(t, ExecutionStatusUncalled, res.Steps[1].Compensation.Status)
		require.Equal(t, 0, res.Steps[1].Compensation.Calls)
		require.Equal(t, 0, len(res.Steps[1].Compensation.Errors))
	})
}
func Test_execute_context(t *testing.T) {
	t.Run("action_ctx_cancel", func(t *testing.T) {
		var (
			ctx, cancel = context.WithCancel(context.Background())
			Calls       = make([]string, 0, 2)
		)

		steps := []Step{
			NewStep("step0").
				WithAction(Operation{}),
			NewStep("step1").
				WithAction(NewOperation(func(ctx context.Context, _ Track) error {
					Calls = append(Calls, "action1")
					return nil
				})),
			NewStep("step2").
				WithAction(NewOperation(func(ctx context.Context, _ Track) error {
					Calls = append(Calls, "action2")
					cancel() // cancel context for test
					return nil
				})),
			NewStep("step3").
				WithAction(NewOperation(func(ctx context.Context, _ Track) error {
					Calls = append(Calls, "action3")
					t.Fatalf("should not have been called")
					return nil
				})),
		}

		res, err := NewSaga(steps).Execute(ctx)
		require.Error(t, err)

		require.Equal(t, StageResultFail, res.Status)
		require.Equal(t, 4, len(res.Steps))
		require.Equal(t, "step0", res.Steps[0].StepName)
		require.Equal(t, 0, res.Steps[0].StepPosition)
		require.Equal(t, ExecutionStatusUnset, res.Steps[0].Action.Status)
		require.Equal(t, ExecutionStatusUnset, res.Steps[0].Compensation.Status)
		require.Equal(t, false, res.Steps[0].CompensationOnActionFailure)

		require.Equal(t, "step1", res.Steps[1].StepName)
		require.Equal(t, 1, res.Steps[1].StepPosition)
		require.Equal(t, ExecutionStatusSuccess, res.Steps[1].Action.Status)
		require.Equal(t, ExecutionStatusUnset, res.Steps[1].Compensation.Status)
		require.Equal(t, false, res.Steps[1].CompensationOnActionFailure)

		require.Equal(t, "step2", res.Steps[2].StepName)
		require.Equal(t, 2, res.Steps[2].StepPosition)
		require.Equal(t, ExecutionStatusSuccess, res.Steps[2].Action.Status)
		require.Equal(t, ExecutionStatusUnset, res.Steps[2].Compensation.Status)
		require.Equal(t, false, res.Steps[2].CompensationOnActionFailure)

		require.Equal(t, "step3", res.Steps[3].StepName)
		require.Equal(t, 3, res.Steps[3].StepPosition)
		require.Equal(t, ExecutionStatusFail, res.Steps[3].Action.Status)
		require.Equal(t, 1, len(res.Steps[3].Action.Errors))
		require.ErrorIs(t, res.Steps[3].Action.Errors[0], ErrExecuteActionsContextDone)
		require.Equal(t, ExecutionStatusUnset, res.Steps[3].Compensation.Status)
		require.Equal(t, 0, len(res.Steps[3].Compensation.Errors))
		require.Equal(t, false, res.Steps[3].CompensationOnActionFailure)

		require.True(t, slices.Equal([]string{"action1", "action2"}, Calls))
	})
	t.Run("retry_ctx_cancel", func(t *testing.T) {
		var (
			ctx, cancel = context.WithCancel(context.Background())
		)

		steps := []Step{
			NewStep("step0").
				WithAction(
					NewOperation(func(ctx context.Context, track Track) error {
						data := track.GetStepData()
						if data.Action.Calls >= 2 {
							cancel() // cancel context for test
						}
						return testtool.ErrExpTestA
					}).WithRetry(NewBaseRetryPolicy(10, 1*time.Nanosecond)),
				),
		}
		res, err := NewSaga(steps).Execute(ctx)
		require.Error(t, err)

		require.Equal(t, StageResultFail, res.Status)
		require.Equal(t, 1, len(res.Steps))
		require.Equal(t, "step0", res.Steps[0].StepName)
		require.Equal(t, 0, res.Steps[0].StepPosition)
		require.Equal(t, ExecutionStatusFail, res.Steps[0].Action.Status)
		require.Equal(t, ExecutionStatusUnset, res.Steps[0].Compensation.Status)
		require.Equal(t, 4, len(res.Steps[0].Action.Errors))
		require.ErrorIs(t, res.Steps[0].Action.Errors[0], testtool.ErrExpTestA)
		require.ErrorIs(t, res.Steps[0].Action.Errors[1], testtool.ErrExpTestA)
		require.ErrorIs(t, res.Steps[0].Action.Errors[2], ErrRetryContextDone)
		require.ErrorIs(t, res.Steps[0].Action.Errors[3], ErrRetryFailed)
	})
	t.Run("compensation_ctx_cancel", func(t *testing.T) {
		var (
			ctx, cancel = context.WithCancel(context.Background())
		)

		steps := []Step{
			NewStep("step0").
				WithAction(
					NewOperation(func(ctx context.Context, _ Track) error {
						cancel() // cancel context for test
						return testtool.ErrExpTestA
					}),
				).WithCompensation(
				NewOperation(func(ctx context.Context, _ Track) error {
					t.Fatalf("should not have been called")
					return nil
				}),
			).WithCompensationOnActionFailure(),
		}
		res, err := NewSaga(steps).Execute(ctx)
		require.Error(t, err)
		require.Equal(t, StageResultFail, res.Status)
		require.Equal(t, 1, len(res.Steps))
		require.Equal(t, "step0", res.Steps[0].StepName)
		require.Equal(t, 0, res.Steps[0].StepPosition)
		require.Equal(t, ExecutionStatusFail, res.Steps[0].Action.Status)
		require.Equal(t, 1, res.Steps[0].Action.Calls)
		require.Equal(t, 1, len(res.Steps[0].Action.Errors))
		require.ErrorIs(t, res.Steps[0].Action.Errors[0], testtool.ErrExpTestA)
		require.Equal(t, ExecutionStatusFail, res.Steps[0].Compensation.Status)
		require.Equal(t, 0, res.Steps[0].Compensation.Calls)
		require.Equal(t, 1, len(res.Steps[0].Compensation.Errors))
		require.ErrorIs(t, res.Steps[0].Compensation.Errors[0], ErrExecuteCompensationContextDone)
	})

	t.Run("with_compensation_context", func(t *testing.T) {
		t.Run("operation_context_is_derived_from_compensation_context", func(t *testing.T) {
			type contextKey string
			const (
				compensationKey contextKey = "compensation"
				operationKey    contextKey = "operation"
			)

			var operationFactoryCalled bool
			steps := []Step{
				NewStep("step0").
					WithAction(NewOperation(func(context.Context, Track) error {
						return testtool.ErrExpTestA
					})).
					WithCompensation(
						NewOperation(func(ctx context.Context, _ Track) error {
							require.Equal(t, "saga", ctx.Value(compensationKey))
							require.Equal(t, "operation", ctx.Value(operationKey))
							return nil
						}).WithContext(func(ctx context.Context) context.Context {
							require.Equal(t, "saga", ctx.Value(compensationKey))
							operationFactoryCalled = true
							return context.WithValue(ctx, operationKey, "operation")
						}),
					).
					WithCompensationOnActionFailure(),
			}

			res, err := NewSaga(steps).
				WithCompensationContext(func(ctx context.Context) context.Context {
					return context.WithValue(ctx, compensationKey, "saga")
				}).
				Execute(context.Background())

			require.Error(t, err)
			require.Equal(t, StageResultCompensated, res.Status)
			require.True(t, operationFactoryCalled)
			require.Equal(t, ExecutionStatusSuccess, res.Steps[0].Compensation.Status)
			require.Equal(t, 1, res.Steps[0].Compensation.Calls)
		})

		t.Run("nil_operation_context_factory_is_ignored", func(t *testing.T) {
			var compensationCalled bool
			steps := []Step{
				NewStep("step0").
					WithAction(NewOperation(func(context.Context, Track) error {
						return testtool.ErrExpTestA
					})).
					WithCompensation(
						NewOperation(func(context.Context, Track) error {
							compensationCalled = true
							return nil
						}).WithContext(nil),
					).
					WithCompensationOnActionFailure(),
			}

			res, err := NewSaga(steps).Execute(context.Background())

			require.Error(t, err)
			require.Equal(t, StageResultCompensated, res.Status)
			require.True(t, compensationCalled)
			require.Equal(t, ExecutionStatusSuccess, res.Steps[0].Compensation.Status)
		})

		t.Run("compensation_uses_own_context_after_action_ctx_cancel", func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			var compensationCalled bool

			steps := []Step{
				NewStep("step0").
					WithAction(
						NewOperation(func(context.Context, Track) error {
							cancel()
							return testtool.ErrExpTestA
						}),
					).
					WithCompensation(
						NewOperation(func(ctx context.Context, _ Track) error {
							require.NoError(t, ctx.Err())
							compensationCalled = true
							return nil
						}),
					).
					WithCompensationOnActionFailure(),
			}

			res, err := NewSaga(steps).
				WithCompensationContext(func(ctx context.Context) context.Context {
					require.ErrorIs(t, ctx.Err(), context.Canceled)
					return context.WithoutCancel(ctx)
				}).
				Execute(ctx)

			require.Error(t, err)
			require.Equal(t, StageResultCompensated, res.Status)
			require.True(t, compensationCalled)
			require.Equal(t, ExecutionStatusSuccess, res.Steps[0].Compensation.Status)
			require.Equal(t, 1, res.Steps[0].Compensation.Calls)
		})
		t.Run("completed_steps_are_compensated_when_action_ctx_is_done", func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			var compensationCalled bool

			steps := []Step{
				NewStep("step0").
					WithAction(NewOperation(func(context.Context, Track) error {
						cancel()
						return nil
					})).
					WithCompensation(NewOperation(func(ctx context.Context, _ Track) error {
						require.NoError(t, ctx.Err())
						compensationCalled = true
						return nil
					})),
				NewStep("step1").
					WithAction(NewOperation(func(context.Context, Track) error {
						t.Fatal("should not have been called")
						return nil
					})),
			}

			res, err := NewSaga(steps).
				WithCompensationContext(func(ctx context.Context) context.Context {
					return context.WithoutCancel(ctx)
				}).
				Execute(ctx)

			require.Error(t, err)
			require.Equal(t, StageResultCompensated, res.Status)
			require.True(t, compensationCalled)
			require.Equal(t, ExecutionStatusSuccess, res.Steps[0].Compensation.Status)
			require.ErrorIs(t, res.Steps[1].Action.Errors[0], ErrExecuteActionsContextDone)
		})
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
						NewOperation(func(ctx context.Context, _ Track) error {
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
			require.Error(t, err)

			require.Equal(t, StageResultFail, res.Status)
			require.Equal(t, 1, len(res.Steps))
			require.Equal(t, 1, res.Steps[0].Action.Calls)
			require.Equal(t, ExecutionStatusFail, res.Steps[0].Action.Status)
			require.Equal(t, 1, len(res.Steps[0].Action.Errors))
			require.ErrorIs(t, res.Steps[0].Action.Errors[0], testtool.ErrExpTestA)

			require.True(t, slices.Equal([]string{"hook2", "hook1", "action1"}, executed))
		})
		t.Run("before_with_retry", func(t *testing.T) {
			var (
				ctx      = context.Background()
				executed = make([]string, 0, 8)
			)

			steps := []Step{
				NewStep("step1").
					WithAction(
						NewOperation(func(ctx context.Context, _ Track) error {
							executed = append(executed, "action1")
							return testtool.ErrExpTestA
						}).WithBeforeHook(func(ctx context.Context, _ Track) error {
							executed = append(executed, "hook1")
							return nil
						}).WithBeforeHook(func(ctx context.Context, _ Track) error {
							executed = append(executed, "hook2")
							return nil
						}).WithRetry(NewBaseRetryPolicy(1, 1*time.Nanosecond)).
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
			require.Error(t, err)
			require.Equal(t, StageResultFail, res.Status)
			require.Equal(t, 1, len(res.Steps))
			require.Equal(t, 2, res.Steps[0].Action.Calls)
			require.Equal(t, ExecutionStatusFail, res.Steps[0].Action.Status)
			require.Equal(t, 3, len(res.Steps[0].Action.Errors))
			require.ErrorIs(t, res.Steps[0].Action.Errors[0], testtool.ErrExpTestA)
			require.ErrorIs(t, res.Steps[0].Action.Errors[1], testtool.ErrExpTestA)
			require.ErrorIs(t, res.Steps[0].Action.Errors[2], ErrRetryFailed)
			require.True(t,
				slices.Equal(
					[]string{
						"retry_hook2", "retry_hook1", // retry hooks
						"hook2", "hook1", "action1", // call
						"hook2", "hook1", "action1", // first retry
					},
					executed),
			)
		})

		t.Run("after", func(t *testing.T) {
			var (
				ctx      = context.Background()
				executed = make([]string, 0, 3)
			)

			steps := []Step{
				NewStep("step1").
					WithAction(
						NewOperation(func(ctx context.Context, track Track) error {
							executed = append(executed, "action1")
							return testtool.ErrExpTestA
						}).WithAfterHook(func(ctx context.Context, track Track) error {
							t.Fatalf("should not have been called")
							return nil
						}).WithAfterHook(func(ctx context.Context, track Track) error {
							t.Fatalf("should not have been called")
							return nil
						}),
					),
			}
			res, err := NewSaga(steps).Execute(ctx)
			require.Error(t, err)
			require.Equal(t, StageResultFail, res.Status)
			require.Equal(t, 1, len(res.Steps))
			require.Equal(t, 1, res.Steps[0].Action.Calls)
			require.Equal(t, ExecutionStatusFail, res.Steps[0].Action.Status)
			require.Equal(t, 1, len(res.Steps[0].Action.Errors))
			require.ErrorIs(t, res.Steps[0].Action.Errors[0], testtool.ErrExpTestA)

			require.True(t, slices.Equal([]string{"action1"}, executed))

		})
		t.Run("after_on_success", func(t *testing.T) {
			var (
				ctx      = context.Background()
				executed []string
			)

			steps := []Step{
				NewStep("step1").
					WithAction(
						NewOperation(func(ctx context.Context, track Track) error {
							executed = append(executed, "action1")
							return nil
						}).WithAfterHook(func(ctx context.Context, track Track) error {
							executed = append(executed, "hook1")
							data := track.GetStepData()
							require.Equal(t, 1, data.Action.Calls)
							return nil
						}).WithAfterHook(func(ctx context.Context, track Track) error {
							executed = append(executed, "hook2")
							data := track.GetStepData()
							require.Equal(t, 1, data.Action.Calls)
							return nil
						}),
					),
			}

			res, err := NewSaga(steps).Execute(ctx)
			require.NoError(t, err)
			require.Equal(t, StageResultSuccess, res.Status)
			require.Equal(t, 1, res.Steps[0].Action.Calls)
			require.Equal(t, ExecutionStatusSuccess, res.Steps[0].Action.Status)
			require.True(t, slices.Equal([]string{"action1", "hook1", "hook2"}, executed))
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
						NewOperation(func(ctx context.Context, track Track) error {
							executed = append(executed, "action1")

							data := track.GetStepData()
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
							t.Fatalf("should not have been called")
							return nil
						}).WithAfterHook(func(ctx context.Context, track Track) error {
							t.Fatalf("should not have been called")
							return nil
						}).WithRetry(NewBaseRetryPolicy(2, 1*time.Nanosecond)).
							WithAfterHook(func(ctx context.Context, track Track) error {
								t.Fatalf("should not have been called")
								return nil
							}).
							WithAfterHook(func(ctx context.Context, track Track) error {
								t.Fatalf("should not have been called")
								return nil
							}),
					),
			}
			res, err := NewSaga(steps).Execute(ctx)
			require.Error(t, err)
			require.Equal(t, StageResultFail, res.Status)

			testtool.TestFn(t, func() {
				t.Logf("\nresult:\n%v", res)
				t.Logf("\nexecution error: %v", err)
				step := res.Steps[0]
				t.Logf("\nstep [%d#%s] action errors:", step.StepPosition, step.StepName)
				for i, e := range res.Steps[0].Action.Errors {
					fmt.Printf("%d: %v\n", i, e)
				}
			})

			require.True(t,
				slices.Equal(
					[]string{
						"action1", // call
						"action1", // first retry
						"action1", // second retry
					},
					executed),
			)

			require.Equal(t, 1, len(res.Steps))

			action := res.Steps[0].Action
			require.Equal(t, 3, res.Steps[0].Action.Calls)
			require.Equal(t, ExecutionStatusFail, action.Status)
			require.Equal(t, 4, len(action.Errors))
			require.ErrorIs(t, action.Errors[0], testtool.ErrExpTestA)
			require.ErrorIs(t, action.Errors[1], testtool.ErrExpTestB)
			require.True(t, checkRetryStr(0, action.Errors[1]))
			require.ErrorIs(t, action.Errors[2], testtool.ErrExpTestC)
			require.True(t, checkRetryStr(1, action.Errors[2]))
			require.ErrorIs(t, action.Errors[3], ErrRetryFailed)

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
						NewOperation(func(ctx context.Context, _ Track) error {
							executed = append(executed, "action1")
							return testtool.ErrExpTestA
						}),
					).WithCompensation(
					NewOperation(func(ctx context.Context, track Track) error {
						executed = append(executed, "comp1")

						data := track.GetStepData()
						require.Equal(t, 1, data.Action.Calls)
						require.Equal(t, 1, data.Compensation.Calls)
						require.Equal(t, 1, len(data.Action.Errors))
						require.ErrorIs(t, data.Action.Errors[0], testtool.ErrExpTestA)

						return nil
					}).WithBeforeHook(func(ctx context.Context, track Track) error {
						executed = append(executed, "comp_hook1")

						data := track.GetStepData()
						require.Equal(t, 1, data.Action.Calls)
						require.Equal(t, 0, data.Compensation.Calls)
						require.Equal(t, 1, len(data.Action.Errors))
						require.ErrorIs(t, data.Action.Errors[0], testtool.ErrExpTestA)
						return nil
					}).WithBeforeHook(func(ctx context.Context, track Track) error {
						executed = append(executed, "comp_hook2")

						data := track.GetStepData()
						require.Equal(t, 1, data.Action.Calls)
						require.Equal(t, 0, data.Compensation.Calls)
						require.Equal(t, 1, len(data.Action.Errors))
						require.ErrorIs(t, data.Action.Errors[0], testtool.ErrExpTestA)
						return nil
					}),
				).WithCompensationOnActionFailure(),
			}
			res, err := NewSaga(steps).Execute(ctx)
			require.Error(t, err)
			require.Equal(t, StageResultCompensated, res.Status)

			require.True(t, slices.Equal([]string{"action1", "comp_hook2", "comp_hook1", "comp1"}, executed))

			require.Equal(t, 1, len(res.Steps))

			action := res.Steps[0].Action
			require.Equal(t, ExecutionStatusFail, action.Status)
			require.Equal(t, 1, len(action.Errors))
			require.ErrorIs(t, action.Errors[0], testtool.ErrExpTestA)

			compensation := res.Steps[0].Compensation
			require.Equal(t, 1, compensation.Calls)
			require.Equal(t, ExecutionStatusSuccess, compensation.Status)
			require.Equal(t, ExecutionStatusSuccess, compensation.Status)
			require.Equal(t, 0, len(compensation.Errors))

		})
		t.Run("after", func(t *testing.T) {
			var (
				ctx      = context.Background()
				compErr  = fmt.Errorf("comp_error_1")
				executed []string
			)

			steps := []Step{
				NewStep("step1").
					WithAction(
						NewOperation(func(ctx context.Context, track Track) error {
							executed = append(executed, "action1")
							return testtool.ErrExpTestA
						}),
					).WithCompensation(
					NewOperation(func(ctx context.Context, track Track) error {
						executed = append(executed, "comp1")
						return compErr
					}).WithAfterHook(func(ctx context.Context, track Track) error {
						t.Fatalf("should not have been called")
						return nil
					}).WithAfterHook(func(ctx context.Context, track Track) error {
						t.Fatalf("should not have been called")
						return nil
					}),
				).WithCompensationOnActionFailure(),
			}

			res, err := NewSaga(steps).Execute(ctx)
			require.Error(t, err)
			require.Equal(t, StageResultFail, res.Status)

			require.True(t, slices.Equal([]string{"action1", "comp1"}, executed))

		})
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
						NewOperation(func(ctx context.Context, _ Track) error {
							return nil
						}),
					),
			}

			res, err := NewSaga(steps).Execute(ctx)
			require.NoError(t, err)
			require.Equal(t, StageResultSuccess, res.Status)

			require.Equal(t, 1, len(res.Steps))

			require.Equal(t, "step1", res.Steps[0].StepName)
			require.Equal(t, 0, res.Steps[0].StepPosition)
			require.Equal(t, ExecutionStatusSuccess, res.Steps[0].Action.Status)
			require.Equal(t, 1, res.Steps[0].Action.Calls)
			require.Equal(t, 0, len(res.Steps[0].Action.Errors))

			testtool.TestFn(t, func() {
				t.Log(res)
			})
		})
		t.Run("fail_v1", func(t *testing.T) {
			steps := []Step{
				NewStep("step1").
					WithAction(
						NewOperation(func(ctx context.Context, _ Track) error {
							return testtool.ErrExpTestA
						}),
					),
			}

			res, err := NewSaga(steps).Execute(ctx)
			require.Error(t, err)
			require.Equal(t, StageResultFail, res.Status)
			require.True(t, errors.Is(err, ErrActionFailed))
			require.Equal(t, 1, len(res.Steps))
			require.Equal(t, "step1", res.Steps[0].StepName)
			require.Equal(t, 0, res.Steps[0].StepPosition)
			require.Equal(t, ExecutionStatusFail, res.Steps[0].Action.Status)
			require.Equal(t, ExecutionStatusUnset, res.Steps[0].Compensation.Status)
			require.Equal(t, 1, res.Steps[0].Action.Calls)
			require.Equal(t, 1, len(res.Steps[0].Action.Errors))
			require.ErrorIs(t, res.Steps[0].Action.Errors[0], testtool.ErrExpTestA)

			testtool.TestFn(t, func() {
				t.Log(res)
			})
		})
		t.Run("fail_v2", func(t *testing.T) {
			steps := []Step{
				NewStep("step1").
					WithAction(
						NewOperation(func(ctx context.Context, _ Track) error {
							return nil
						}),
					),
				NewStep("step2").
					WithAction(
						NewOperation(func(ctx context.Context, _ Track) error {
							return testtool.ErrExpTestA
						}),
					),
			}

			res, err := NewSaga(steps).Execute(ctx)
			require.Error(t, err)
			require.Equal(t, StageResultFail, res.Status)
			require.True(t, errors.Is(err, ErrActionFailed))

			require.Equal(t, 2, len(res.Steps))

			require.Equal(t, "step1", res.Steps[0].StepName)
			require.Equal(t, 0, res.Steps[0].StepPosition)
			require.Equal(t, ExecutionStatusSuccess, res.Steps[0].Action.Status)
			require.Equal(t, ExecutionStatusUnset, res.Steps[0].Compensation.Status)
			require.Equal(t, false, res.Steps[0].CompensationOnActionFailure)
			require.Equal(t, 1, res.Steps[0].Action.Calls)
			require.Equal(t, 0, len(res.Steps[0].Action.Errors))

			require.Equal(t, "step2", res.Steps[1].StepName)
			require.Equal(t, 1, res.Steps[1].StepPosition)
			require.Equal(t, ExecutionStatusFail, res.Steps[1].Action.Status)
			require.Equal(t, ExecutionStatusUnset, res.Steps[1].Compensation.Status)
			require.Equal(t, false, res.Steps[1].CompensationOnActionFailure)
			require.Equal(t, 1, res.Steps[1].Action.Calls)
			require.Equal(t, 1, len(res.Steps[1].Action.Errors))
			require.ErrorIs(t, res.Steps[1].Action.Errors[0], testtool.ErrExpTestA)

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
						NewOperation(func(ctx context.Context, _ Track) error {
							return testtool.ErrExpTestA
						}),
					).
					WithCompensation(
						NewOperation(func(ctx context.Context, track Track) error {
							str := track.GetStepData()
							require.Equal(t, "step1", str.StepName)
							require.Equal(t, 0, str.StepPosition)
							require.Equal(t, ExecutionStatusFail, str.Action.Status)
							require.Equal(t, 1, str.Action.Calls)
							require.Equal(t, 1, len(str.Action.Errors))

							require.Equal(t, ExecutionStatusUncalled, str.Compensation.Status)
							require.Equal(t, 1, str.Compensation.Calls)
							require.Equal(t, 0, len(str.Compensation.Errors))
							return nil
						}),
					).WithCompensationOnActionFailure(),
			}

			res, err := NewSaga(steps).Execute(ctx)
			require.Error(t, err)
			require.Equal(t, StageResultCompensated, res.Status)
			require.True(t, errors.Is(err, ErrActionFailed))

			require.Equal(t, 1, len(res.Steps))

			require.Equal(t, "step1", res.Steps[0].StepName)
			require.Equal(t, 0, res.Steps[0].StepPosition)
			require.Equal(t, ExecutionStatusFail, res.Steps[0].Action.Status)
			require.Equal(t, 1, res.Steps[0].Action.Calls)
			require.Equal(t, 1, len(res.Steps[0].Action.Errors))

			require.Equal(t, ExecutionStatusSuccess, res.Steps[0].Compensation.Status)
			require.Equal(t, 1, res.Steps[0].Compensation.Calls)
			require.Equal(t, 0, len(res.Steps[0].Compensation.Errors))

			testtool.TestFn(t, func() {
				t.Log(res)
			})
		})
		t.Run("compensate_v1", func(t *testing.T) {
			steps := []Step{
				NewStep("step1").
					WithAction(
						NewOperation(func(ctx context.Context, _ Track) error {
							return testtool.ErrExpTestA
						}),
					).
					WithCompensation(
						NewOperation(func(ctx context.Context, track Track) error {
							str := track.GetStepData()
							require.Equal(t, "step1", str.StepName)
							require.Equal(t, 0, str.StepPosition)
							require.Equal(t, ExecutionStatusFail, str.Action.Status)
							require.Equal(t, 1, str.Action.Calls)
							require.Equal(t, 1, len(str.Action.Errors))
							require.ErrorIs(t, str.Action.Errors[0], testtool.ErrExpTestA)

							require.Equal(t, ExecutionStatusUncalled, str.Compensation.Status)
							require.Equal(t, 1, str.Compensation.Calls)
							require.Equal(t, 0, len(str.Compensation.Errors))
							return nil
						}),
					).WithCompensationOnActionFailure(),
			}

			res, err := NewSaga(steps).Execute(ctx)
			require.Error(t, err)
			require.Equal(t, StageResultCompensated, res.Status)
			require.True(t, errors.Is(err, ErrActionFailed))

			require.Equal(t, 1, len(res.Steps))

			require.Equal(t, "step1", res.Steps[0].StepName)
			require.Equal(t, 0, res.Steps[0].StepPosition)
			require.Equal(t, ExecutionStatusFail, res.Steps[0].Action.Status)
			require.Equal(t, 1, res.Steps[0].Action.Calls)
			require.Equal(t, 1, len(res.Steps[0].Action.Errors))

			require.Equal(t, ExecutionStatusSuccess, res.Steps[0].Compensation.Status)
			require.Equal(t, 1, res.Steps[0].Compensation.Calls)
			require.Equal(t, 0, len(res.Steps[0].Compensation.Errors))

			testtool.TestFn(t, func() {
				t.Log(res)
			})
		})
		t.Run("compensate_v2", func(t *testing.T) {
			steps := []Step{
				NewStep("step1").
					WithAction(
						NewOperation(func(ctx context.Context, _ Track) error {
							return nil
						}),
					),
				NewStep("step2").
					WithAction(
						NewOperation(func(ctx context.Context, _ Track) error {
							return testtool.ErrExpTestA
						}),
					).
					WithCompensation(
						NewOperation(func(ctx context.Context, track Track) error {
							return nil
						}),
					),
			}

			res, err := NewSaga(steps).Execute(ctx)
			require.Error(t, err)
			require.Equal(t, StageResultFail, res.Status)
			require.True(t, errors.Is(err, ErrActionFailed))

			require.Equal(t, 2, len(res.Steps))

			require.Equal(t, "step1", res.Steps[0].StepName)
			require.Equal(t, 0, res.Steps[0].StepPosition)
			require.Equal(t, ExecutionStatusSuccess, res.Steps[0].Action.Status)
			require.Equal(t, ExecutionStatusUnset, res.Steps[0].Compensation.Status)
			require.Equal(t, false, res.Steps[0].CompensationOnActionFailure)
			require.Equal(t, 1, res.Steps[0].Action.Calls)
			require.Equal(t, 0, len(res.Steps[0].Action.Errors))

			require.Equal(t, "step2", res.Steps[1].StepName)
			require.Equal(t, 1, res.Steps[1].StepPosition)
			require.Equal(t, ExecutionStatusFail, res.Steps[1].Action.Status)
			require.Equal(t, ExecutionStatusUncalled, res.Steps[1].Compensation.Status)
			require.Equal(t, false, res.Steps[1].CompensationOnActionFailure)
			require.Equal(t, 1, res.Steps[1].Action.Calls)
			require.Equal(t, 1, len(res.Steps[1].Action.Errors))
			require.ErrorIs(t, res.Steps[1].Action.Errors[0], testtool.ErrExpTestA)

			testtool.TestFn(t, func() {
				t.Log(res)
			})
		})
		t.Run("fail_v1", func(t *testing.T) {
			steps := []Step{
				NewStep("step1").
					WithAction(
						NewOperation(func(ctx context.Context, _ Track) error {
							return testtool.ErrExpTestA
						}),
					).
					WithCompensation(
						NewOperation(func(ctx context.Context, track Track) error {
							str := track.GetStepData()
							require.Equal(t, "step1", str.StepName)
							require.Equal(t, 0, str.StepPosition)
							require.Equal(t, ExecutionStatusFail, str.Action.Status)
							require.Equal(t, 1, str.Action.Calls)
							require.Equal(t, 1, len(str.Action.Errors))
							require.ErrorIs(t, str.Action.Errors[0], testtool.ErrExpTestA)

							require.Equal(t, ExecutionStatusUncalled, str.Compensation.Status)
							require.Equal(t, 1, str.Compensation.Calls)
							require.Equal(t, 0, len(str.Compensation.Errors))
							return testtool.ErrExpTestB
						}),
					).WithCompensationOnActionFailure(),
			}

			res, err := NewSaga(steps).Execute(ctx)
			require.Error(t, err)
			require.Equal(t, StageResultFail, res.Status)
			require.ErrorIs(t, err, ErrActionFailed)
			require.ErrorIs(t, err, ErrCompensationFailed)

			require.Equal(t, 1, len(res.Steps))

			require.Equal(t, "step1", res.Steps[0].StepName)
			require.Equal(t, 0, res.Steps[0].StepPosition)
			require.Equal(t, ExecutionStatusFail, res.Steps[0].Action.Status)
			require.Equal(t, 1, res.Steps[0].Action.Calls)
			require.Equal(t, 1, len(res.Steps[0].Action.Errors))

			require.Equal(t, ExecutionStatusFail, res.Steps[0].Compensation.Status)
			require.Equal(t, 1, res.Steps[0].Compensation.Calls)
			require.Equal(t, 1, len(res.Steps[0].Compensation.Errors))
			require.ErrorIs(t, res.Steps[0].Compensation.Errors[0], testtool.ErrExpTestB)

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
					NewOperation(func(ctx context.Context, _ Track) error {
						return testtool.ErrExpTestA
					}).
						WithRetry(NewBaseRetryPolicy(retries, 5*time.Nanosecond)),
				).
				WithCompensation(
					NewOperation(func(ctx context.Context, track Track) error {
						str := track.GetStepData()
						if str.Compensation.Calls < retries+1 {
							return fmt.Errorf("comp err [%d]: %w", len(str.Compensation.Errors), testtool.ErrExpTestA)
						}
						return nil
					}).WithRetry(NewBaseRetryPolicy(retries, 5*time.Nanosecond)),
				).WithCompensationOnActionFailure(),
		}

		res, err := NewSaga(steps).Execute(ctx)
		require.Error(t, err)
		require.Equal(t, StageResultCompensated, res.Status)
		require.ErrorIs(t, err, ErrActionFailed)
		require.ErrorIsNot(t, err, ErrCompensationFailed)

		require.Equal(t, 1, len(res.Steps))

		require.Equal(t, "step1", res.Steps[0].StepName)
		require.Equal(t, 0, res.Steps[0].StepPosition)
		require.Equal(t, ExecutionStatusFail, res.Steps[0].Action.Status)
		require.Equal(t, 5, res.Steps[0].Action.Calls)
		require.Equal(t, 6, len(res.Steps[0].Action.Errors))

		// check all errors except on [ErrRetryFailed]
		for i := 0; i < len(res.Steps[0].Action.Errors)-1; i++ {
			e := res.Steps[0].Action.Errors[i]
			require.ErrorIs(t, e, testtool.ErrExpTestA)
		}
		// check last [ErrRetryFailed] error
		require.ErrorIs(t, res.Steps[0].Action.Errors[len(res.Steps[0].Action.Errors)-1], ErrRetryFailed)

		require.Equal(t, ExecutionStatusSuccess, res.Steps[0].Compensation.Status)
		require.Equal(t, 5, res.Steps[0].Compensation.Calls)
		require.Equal(t, 4, len(res.Steps[0].Compensation.Errors))
		for _, e := range res.Steps[0].Compensation.Errors {
			require.ErrorIs(t, e, testtool.ErrExpTestA)
		}

		testtool.TestFn(t, func() {
			t.Log(
				res,
				"+ error:", err,
				"\n + Action errors: ", testtool.JoinAsString(", ", res.Steps[0].Action.Errors),
				"\n + Compensation errors: ", testtool.JoinAsString(", ", res.Steps[0].Compensation.Errors),
			)
		})

	})
	t.Run("compensation", func(t *testing.T) {
		steps := []Step{
			NewStep("step1").
				WithAction(
					NewOperation(func(ctx context.Context, _ Track) error {
						return testtool.ErrExpTestA
					}).
						WithRetry(NewBaseRetryPolicy(4, 5*time.Nanosecond)),
				).
				WithCompensation(
					NewOperation(func(ctx context.Context, track Track) error {
						str := track.GetStepData()
						return fmt.Errorf("comp err [%d]: %w", len(str.Compensation.Errors), testtool.ErrExpTestB)
					}).WithRetry(NewBaseRetryPolicy(4, 5*time.Nanosecond)),
				).WithCompensationOnActionFailure(),
		}

		res, err := NewSaga(steps).Execute(ctx)
		require.Error(t, err)
		require.Equal(t, StageResultFail, res.Status)
		require.ErrorIs(t, err, ErrActionFailed)
		require.ErrorIs(t, err, ErrCompensationFailed)

		require.Equal(t, 1, len(res.Steps))

		require.Equal(t, "step1", res.Steps[0].StepName)
		require.Equal(t, 0, res.Steps[0].StepPosition)
		require.Equal(t, ExecutionStatusFail, res.Steps[0].Action.Status)
		require.Equal(t, 5, res.Steps[0].Action.Calls)
		require.Equal(t, 6, len(res.Steps[0].Action.Errors))

		// check all errors except on [ErrRetryFailed]
		for i := 0; i < len(res.Steps[0].Action.Errors)-1; i++ {
			e := res.Steps[0].Action.Errors[i]
			require.ErrorIs(t, e, testtool.ErrExpTestA)
		}
		// check last [ErrRetryFailed] error
		require.ErrorIs(t, res.Steps[0].Action.Errors[len(res.Steps[0].Action.Errors)-1], ErrRetryFailed)

		require.Equal(t, ExecutionStatusFail, res.Steps[0].Compensation.Status)
		require.Equal(t, 5, res.Steps[0].Compensation.Calls)
		require.Equal(t, 6, len(res.Steps[0].Compensation.Errors))

		// check all errors except on [ErrRetryFailed]
		for i := 0; i < len(res.Steps[0].Compensation.Errors)-1; i++ {
			e := res.Steps[0].Compensation.Errors[i]
			require.ErrorIs(t, e, testtool.ErrExpTestB)
		}
		// check last [ErrRetryFailed] error
		require.ErrorIs(t, res.Steps[0].Compensation.Errors[len(res.Steps[0].Compensation.Errors)-1], ErrRetryFailed)

		testtool.TestFn(t, func() {
			t.Log(
				res,
				"+ error:", err,
				"\n + Action errors: ", testtool.JoinAsString(", ", res.Steps[0].Action.Errors...),
				"\n + Compensation errors: ", testtool.JoinAsString(", ", res.Steps[0].Compensation.Errors...),
			)
		})

	})
}
