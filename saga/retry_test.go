//nolint:dupl
package saga

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kozmod/oniontx/internal/testtool"
	"github.com/kozmod/oniontx/internal/testtool/assert"
)

type jitterFunc func(time.Duration) time.Duration

func (f jitterFunc) Jitter(base time.Duration) time.Duration { return f(base) }

func Test_backoff(t *testing.T) {
	t.Run("exponential", func(t *testing.T) {
		t.Run("v1", func(t *testing.T) {
			var (
				baseTime = time.Second
			)
			backoff := NewExponentialBackoff()
			delay := backoff.Backoff(1, baseTime)
			if delay <= baseTime {
				t.Fail()
			}
		})

		t.Run("v2", func(t *testing.T) {
			backoff := NewExponentialBackoff()

			assert.Equal(t, 8*time.Second, backoff.Backoff(3, time.Second))
		})

		t.Run("nil_backoff_uses_exponential_default", func(t *testing.T) {
			policy := NewAdvancedRetryPolicy(1, time.Second, nil)

			assert.Equal(t, 2*time.Second, policy.Delay(1))
		})

		t.Run("on_overflow", func(t *testing.T) {
			backoff := NewExponentialBackoff()

			assert.Equal(t, maxDuration, backoff.Backoff(64, time.Second))
		})

		t.Run("max_delay_applies_after_backoff_saturates", func(t *testing.T) {
			policy := NewAdvancedRetryPolicy(64, time.Second, NewExponentialBackoff()).
				WithMaxDelay(10 * time.Second)

			assert.Equal(t, 10*time.Second, policy.Delay(64))
		})

		t.Run("max_delay_applies_after_custom_jitter", func(t *testing.T) {
			policy := NewAdvancedRetryPolicy(1, time.Second, NewExponentialBackoff()).
				WithJitter(
					jitterFunc(func(time.Duration) time.Duration {
						return time.Hour
					}),
				).
				WithMaxDelay(10 * time.Second)

			assert.Equal(t, 10*time.Second, policy.Delay(1))
		})
	})
}

func Test_Saga_retry(t *testing.T) {
	var (
		ctx = context.Background()
	)
	t.Run("static_func", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			var (
				errCounter  = 0
				actionCalls = 0
			)
			steps := []Step{
				NewStep("step0").
					WithAction(
						NewOperation(func(ctx context.Context, _ Track) error {
							actionCalls++
							errCounter++
							if errCounter < 3 {
								return testtool.ErrExpTestA
							}
							return nil
						}).WithRetry(NewBaseRetryPolicy(3, time.Nanosecond)),
					),
			}

			resp, err := NewSaga(steps).Execute(ctx)
			assert.NoError(t, err)
			assert.Equal(t, StageResultSuccess, resp.Status)
			assert.Equal(t, 3, actionCalls)
			assert.Equal(t, ExecutionStatusSuccess, resp.Steps[0].Action.Status)
			assert.Equal(t, 3, resp.Steps[0].Action.Calls)
			assert.Equal(t, 2, len(resp.Steps[0].Action.Errors))
			for _, e := range resp.Steps[0].Action.Errors {
				assert.ErrorIs(t, e, testtool.ErrExpTestA)
			}
		})
		t.Run("builders", func(t *testing.T) {
			t.Run("success_Operation", func(t *testing.T) {
				var (
					errCounter  = 0
					actionCalls = 0
				)
				steps := []Step{
					NewStep("step0").
						WithAction(
							NewOperation(func(ctx context.Context, _ Track) error {
								actionCalls++
								errCounter++
								if errCounter < 3 {
									return testtool.ErrExpTestA
								}
								return nil
							}).WithRetry(
								NewBaseRetryPolicy(3, time.Nanosecond),
							),
						),
				}

				resp, err := NewSaga(steps).Execute(ctx)
				assert.NoError(t, err)
				assert.Equal(t, StageResultSuccess, resp.Status)
				assert.Equal(t, 3, actionCalls)
				assert.Equal(t, ExecutionStatusSuccess, resp.Steps[0].Action.Status)
				assert.Equal(t, 3, resp.Steps[0].Action.Calls)
				assert.Equal(t, 2, len(resp.Steps[0].Action.Errors))
				for _, e := range resp.Steps[0].Action.Errors {
					assert.ErrorIs(t, e, testtool.ErrExpTestA)
				}
			})
			t.Run("success_OperationFunc", func(t *testing.T) {
				var (
					errCounter        = 0
					actionCalls       = 0
					compensationCalls = 0
				)
				steps := []Step{
					NewStep("step0").
						WithAction(
							NewOperation(func(ctx context.Context, track Track) error {
								actionCalls++
								return testtool.ErrExpTestA
							}),
						).
						WithCompensation(
							NewOperation(func(ctx context.Context, track Track) error {
								compensationCalls++
								errCounter++
								if errCounter < 3 {
									return testtool.ErrExpTestA
								}
								return nil
							}).WithRetry(
								NewBaseRetryPolicy(3, time.Nanosecond),
							),
						).
						WithCompensationOnActionFailure(),
				}

				resp, err := NewSaga(steps).Execute(ctx)
				assert.Error(t, err)
				assert.Equal(t, StageResultCompensated, resp.Status)
				assert.Equal(t, 1, actionCalls)
				assert.Equal(t, 3, compensationCalls)
			})
		})
	})
	t.Run("context_cancel_during_delay", func(t *testing.T) {
		var (
			ctx, cancel = context.WithCancel(context.Background())
			actionCalls = 0
		)

		steps := []Step{
			NewStep("step0").
				WithAction(
					NewOperation(func(ctx context.Context, _ Track) error {
						actionCalls++
						if actionCalls == 1 {
							cancel()
						}
						return testtool.ErrExpTestA
					}).WithRetry(NewBaseRetryPolicy(3, time.Hour)),
				),
		}

		resp, err := NewSaga(steps).Execute(ctx)
		assert.Error(t, err)
		assert.Equal(t, StageResultFail, resp.Status)
		assert.Equal(t, 1, actionCalls)
		assert.Equal(t, 3, len(resp.Steps[0].Action.Errors))
		assert.ErrorIs(t, resp.Steps[0].Action.Errors[0], testtool.ErrExpTestA)
		assert.ErrorIs(t, resp.Steps[0].Action.Errors[1], ErrRetryContextDone)
		assert.ErrorIs(t, resp.Steps[0].Action.Errors[2], ErrRetryFailed)
	})
	t.Run("context_cancellation_is_returned", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		track := newInMemoryTrack(0, NewStep("step"))

		err := WithRetry(NewBaseRetryPolicy(1, time.Hour), func(context.Context, Track) error {
			cancel()
			return testtool.ErrExpTestA
		})(ctx, track.action)

		assert.True(t, errors.Is(err, ErrRetryFailed))
		assert.True(t, errors.Is(err, ErrRetryContextDone))
		assert.True(t, errors.Is(err, context.Canceled))
	})
}

func Test_WithRetry_nil_arguments(t *testing.T) {
	policy := NewBaseRetryPolicy(1, time.Nanosecond)

	t.Run("nil_policy", func(t *testing.T) {
		called := false
		fn := func(context.Context, Track) error {
			called = true
			return nil
		}

		err := WithRetry(nil, fn)(context.Background(), nil)

		assert.ErrorIs(t, err, ErrNilRetryPolicy)
		assert.False(t, called)
	})

	t.Run("nil_operation_function", func(t *testing.T) {
		err := WithRetry(policy, nil)(context.Background(), nil)

		assert.ErrorIs(t, err, ErrNilRetryFunc)
	})
}
