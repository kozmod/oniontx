package saga

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/kozmod/oniontx/internal/testtool"
	"github.com/kozmod/oniontx/internal/testtool/require"
)

func Test_Saga_Action_Panic_Triggers_Compensation(t *testing.T) {
	for _, compensationCase := range []struct {
		name                   string
		compensateFailedAction bool
	}{
		{
			name:                   "completed_steps",
			compensateFailedAction: false,
		},
		{
			name:                   "including_failed_action",
			compensateFailedAction: true,
		},
	} {
		t.Run(compensationCase.name, func(t *testing.T) {
			var calls []string
			compensation := func(name string) Operation {
				return NewOperation(func(context.Context, Track) error {
					calls = append(calls, name)
					return nil
				})
			}
			failed := NewStep("failed").
				WithAction(NewOperation(func(context.Context, Track) error {
					panic("action panic")
				})).
				WithCompensation(compensation("failed"))
			switch {
			case compensationCase.compensateFailedAction:
				failed = failed.WithCompensationOnActionFailure()
			}
			result, err := NewSaga([]Step{
				NewStep("completed").
					WithAction(NewOperation(nil)).
					WithCompensation(compensation("completed")),
				failed,
				NewStep("future").
					WithAction(NewOperation(func(context.Context, Track) error {
						t.Fatal("future action must not run")
						return nil
					})),
			}).
				WithPanicRecovery(true).
				Execute(context.Background())
			require.Equal(t, StageResultCompensated, result.Status)
			require.ErrorIs(t, err, ErrActionFailed)
			require.ErrorIs(t, err, ErrPanicRecovered)
			require.ErrorIsNot(t, err, ErrCompensationFailed)
			require.True(t, strings.Contains(err.Error(), "action panic"))
			require.Equal(t, ExecutionStatusFail, result.Steps[1].Action.Status)
			require.Equal(t, 1, result.Steps[1].Action.Calls)
			require.ErrorIs(t, result.Steps[1].Action.Errors[0], ErrPanicRecovered)
			require.Equal(t, ExecutionStatusUncalled, result.Steps[2].Action.Status)
			switch {
			case compensationCase.compensateFailedAction:
				require.True(t, slices.Equal([]string{"failed", "completed"}, calls))
			default:
				require.True(t, slices.Equal([]string{"completed"}, calls))
			}
		})
	}
}

func Test_Saga_Compensation_Panic_Does_Not_Stop_Rollback(t *testing.T) {
	var calls []string
	result, err := NewSaga([]Step{
		NewStep("first").
			WithAction(NewOperation(nil)).
			WithCompensation(NewOperation(func(context.Context, Track) error {
				calls = append(calls, "first")
				return nil
			})),
		NewStep("second").
			WithAction(NewOperation(nil)).
			WithCompensation(NewOperation(func(context.Context, Track) error {
				calls = append(calls, "second")
				panic("compensation panic")
			})),
		NewStep("failed").
			WithAction(NewOperation(func(context.Context, Track) error {
				return testtool.ErrExpTestA
			})),
	}).
		WithPanicRecovery(true).
		Execute(context.Background())
	require.Equal(t, StageResultFail, result.Status)
	require.True(t, slices.Equal([]string{"second", "first"}, calls))
	require.ErrorIs(t, err, ErrActionFailed)
	require.ErrorIs(t, err, ErrCompensationFailed)
	require.ErrorIs(t, err, ErrPanicRecovered)
	require.ErrorIs(t, err, testtool.ErrExpTestA)
	require.Equal(t, ExecutionStatusSuccess, result.Steps[0].Compensation.Status)
	require.Equal(t, ExecutionStatusFail, result.Steps[1].Compensation.Status)
	require.ErrorIs(t, result.Steps[1].Compensation.Errors[0], ErrPanicRecovered)
}

func Test_Saga_Panic_Retry_Requires_Explicit_Recovery(t *testing.T) {
	for _, recoveryCase := range []struct {
		name     string
		explicit bool
	}{
		{
			name:     "automatic_recovery",
			explicit: false,
		},
		{
			name:     "explicit_recovery_inside_retry",
			explicit: true,
		},
	} {
		t.Run(recoveryCase.name, func(t *testing.T) {
			calls := 0
			operation := NewOperation(func(context.Context, Track) error {
				calls++
				switch calls {
				case 1:
					panic("first attempt")
				default:
					return nil
				}
			})
			switch {
			case recoveryCase.explicit:
				operation = operation.WithPanicRecovery()
			}
			result, err := NewSaga([]Step{
				NewStep("step").
					WithAction(operation.WithRetry(NewBaseRetryPolicy(1, 0))),
			}).
				WithPanicRecovery(true).
				Execute(context.Background())
			switch {
			case recoveryCase.explicit:
				require.NoError(t, err)
				require.Equal(t, StageResultSuccess, result.Status)
				require.Equal(t, 2, calls)
				require.ErrorIs(t, result.Errors()[0], ErrPanicRecovered)
			default:
				require.ErrorIs(t, err, ErrPanicRecovered)
				require.Equal(t, StageResultFail, result.Status)
				require.Equal(t, 1, calls)
			}
		})
	}
}

func Test_Saga_Recovers_Hook_Panics(t *testing.T) {
	for _, hookCase := range []struct {
		name   string
		before bool
	}{
		{
			name:   "before",
			before: true,
		},
		{
			name:   "after",
			before: false,
		},
	} {
		t.Run(hookCase.name, func(t *testing.T) {
			compensationCalled := false
			operation := NewOperation(nil)
			hook := func(context.Context, Track) error { panic("hook panic") }
			switch {
			case hookCase.before:
				operation = operation.WithBeforeHook(hook)
			default:
				operation = operation.WithAfterHook(hook)
			}
			result, err := NewSaga([]Step{
				NewStep("step").
					WithAction(operation).
					WithCompensation(NewOperation(func(context.Context, Track) error {
						compensationCalled = true
						return nil
					})).
					WithCompensationOnActionFailure(),
			}).
				WithPanicRecovery(true).
				Execute(context.Background())
			require.Equal(t, StageResultCompensated, result.Status)
			require.ErrorIs(t, err, ErrPanicRecovered)
			require.True(t, compensationCalled)
			switch {
			case hookCase.before:
				require.Equal(t, 0, result.Steps[0].Action.Calls)
			default:
				require.Equal(t, 1, result.Steps[0].Action.Calls)
			}
		})
	}
}

func Test_Saga_Panic_Recovery_Disabled_By_Default(t *testing.T) {
	for _, recoveryCase := range []struct {
		name              string
		disableExplicitly bool
	}{
		{
			name:              "default",
			disableExplicitly: false,
		},
		{
			name:              "explicitly_disabled",
			disableExplicitly: true,
		},
	} {
		t.Run(recoveryCase.name, func(t *testing.T) {
			compensationCalled := false
			original := NewSaga([]Step{
				NewStep("first").
					WithAction(NewOperation(nil)).
					WithCompensation(NewOperation(func(context.Context, Track) error {
						compensationCalled = true
						return nil
					})),
				NewStep("panic").
					WithAction(NewOperation(func(context.Context, Track) error {
						panic("original panic")
					})),
			})
			configured := original.WithPanicRecovery(true)
			require.False(t, original.recoverPanic)
			require.True(t, configured.Clone().recoverPanic)
			switch {
			case recoveryCase.disableExplicitly:
				original = configured.WithPanicRecovery(false)
				require.True(t, configured.recoverPanic)
			}
			defer func() {
				require.Equal(t, "original panic", recover())
				require.False(t, compensationCalled)
			}()
			_, _ = original.Execute(context.Background())
			t.Fatal("panic must escape Execute")
		})
	}
}
