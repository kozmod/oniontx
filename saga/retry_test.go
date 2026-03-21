package saga

import (
	"context"
	"testing"
	"time"

	"github.com/kozmod/oniontx/internal/testtool"
	"github.com/kozmod/oniontx/internal/testtool/assert"
)

func Test_backoff(t *testing.T) {
	t.Run("exponential_v1", func(t *testing.T) {
		var (
			baseTime = time.Second
		)
		backoff := NewExponentialBackoff()
		delay := backoff.Backoff(1, baseTime)
		if delay <= baseTime {
			t.Fail()
		}
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
				{
					Name: "step0",
					Action: WithRetry(
						NewBaseRetryOpt(3, time.Nanosecond),
						func(ctx context.Context, _ Track) error {
							actionCalls++
							errCounter++
							if errCounter < 3 {
								return testtool.ErrExpTestA
							}
							return nil
						}),
				},
			}

			resp, err := NewSaga(steps).Execute(ctx)
			assert.NoError(t, err)
			assert.Equal(t, StageResultSuccess, resp.Status)
			assert.Equal(t, 3, actionCalls)
			assert.Equal(t, ExecutionStatusSuccess, resp.Sterps[0].Action.Status)
			assert.Equal(t, 3, resp.Sterps[0].Action.Calls)
			assert.Equal(t, 2, len(resp.Sterps[0].Action.Errors))
			for _, e := range resp.Sterps[0].Action.Errors {
				assert.ErrorIs(t, e, testtool.ErrExpTestA)
			}
		})
		t.Run("builders", func(t *testing.T) {
			t.Run("success_ActionFunc", func(t *testing.T) {
				var (
					errCounter  = 0
					actionCalls = 0
				)
				steps := []Step{
					{
						Name: "step0",
						Action: ActionFunc(func(ctx context.Context, _ Track) error {
							actionCalls++
							errCounter++
							if errCounter < 3 {
								return testtool.ErrExpTestA
							}
							return nil
						}).WithRetry(
							NewBaseRetryOpt(3, time.Nanosecond),
						),
					},
				}

				resp, err := NewSaga(steps).Execute(ctx)
				assert.NoError(t, err)
				assert.Equal(t, StageResultSuccess, resp.Status)
				assert.Equal(t, 3, actionCalls)
				assert.Equal(t, ExecutionStatusSuccess, resp.Sterps[0].Action.Status)
				assert.Equal(t, 3, resp.Sterps[0].Action.Calls)
				assert.Equal(t, 2, len(resp.Sterps[0].Action.Errors))
				for _, e := range resp.Sterps[0].Action.Errors {
					assert.ErrorIs(t, e, testtool.ErrExpTestA)
				}
			})
			t.Run("success_CompensationFunc", func(t *testing.T) {
				var (
					errCounter        = 0
					actionCalls       = 0
					compensationCalls = 0
				)
				steps := []Step{
					{
						Name: "step0",
						Action: ActionFunc(func(ctx context.Context, track Track) error {
							actionCalls++
							return testtool.ErrExpTestA
						}),
						Compensation: CompensationFunc(func(ctx context.Context, track Track) error {
							compensationCalls++
							errCounter++
							if errCounter < 3 {
								return testtool.ErrExpTestA
							}
							return nil
						}).WithRetry(
							NewBaseRetryOpt(3, time.Nanosecond),
						),
						CompensationOnFail: true,
					},
				}

				resp, err := NewSaga(steps).Execute(ctx)
				assert.Error(t, err)
				assert.Equal(t, StageResultCompensated, resp.Status)
				assert.Equal(t, 1, actionCalls)
				assert.Equal(t, 3, compensationCalls)
			})
		})
	})
}
