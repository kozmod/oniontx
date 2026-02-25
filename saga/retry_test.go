package saga

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/kozmod/oniontx/internal/testtool"
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

func Test_jitter(t *testing.T) {
	t.Run("full_jitter_v1", func(t *testing.T) {
		var (
			baseTime   = 10 * time.Nanosecond
			fullJitter = NewFullJitter()
		)
		jitter := fullJitter.Jitter(baseTime)
		if jitter > baseTime {
			t.Fatalf("jitter is greater than base time[jitter: %v, base_time: %v]", jitter, baseTime)
		}
		if jitter < 0 {
			t.Fatalf("jitter is less than zero[jitter: %v]", jitter)
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
						func(ctx context.Context) error {
							actionCalls++
							errCounter++
							if errCounter < 3 {
								return testtool.ErrExpTest
							}
							return nil
						}),
				},
			}

			err := NewSaga(steps).Execute(ctx)
			testtool.AssertNoError(t, err)
			testtool.AssertTrue(t, actionCalls == 3)
		})
		t.Run("success_with_return_all_errors", func(t *testing.T) {
			var (
				errCounter  = 0
				actionCalls = 0

				secondExpErr = fmt.Errorf("some_err_2")
			)
			steps := []Step{
				NewStep("step0").
					WithAction(
						WithRetry(
							NewBaseRetryOpt(3, time.Nanosecond).
								WithReturnAllAroseErr(),
							func(ctx context.Context) error {
								actionCalls++
								errCounter++
								switch actionCalls {
								case 1:
									return testtool.ErrExpTest
								case 2:
									return secondExpErr
								default:
									return nil
								}
							}),
					),
			}

			err := NewSaga(steps).Execute(ctx)
			testtool.AssertError(t, err)
			testtool.AssertTrue(t, actionCalls == 3)
			testtool.AssertTrue(t, errors.Is(err, testtool.ErrExpTest))
			testtool.AssertTrue(t, errors.Is(err, secondExpErr))
			testtool.AssertTrue(t, errors.Is(err, ErrActionFailed))

			t.Logf("test error output: \n{\n%v\n}", err)
		})
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
					Action: ActionFunc(func(ctx context.Context) error {
						actionCalls++
						errCounter++
						if errCounter < 3 {
							return testtool.ErrExpTest
						}
						return nil
					}).WithRetry(
						NewBaseRetryOpt(3, time.Nanosecond),
					),
				},
			}

			err := NewSaga(steps).Execute(ctx)
			testtool.AssertNoError(t, err)
			testtool.AssertTrue(t, actionCalls == 3)
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
					Action: ActionFunc(func(ctx context.Context) error {
						actionCalls++
						return testtool.ErrExpTest
					}),
					Compensation: CompensationFunc(func(ctx context.Context, aroseErr error) error {
						compensationCalls++
						errCounter++
						if errCounter < 3 {
							return testtool.ErrExpTest
						}
						return nil
					}).WithRetry(
						NewBaseRetryOpt(3, time.Nanosecond),
					),
					CompensationOnFail: true,
				},
			}

			err := NewSaga(steps).Execute(ctx)
			testtool.AssertError(t, err)
			testtool.AssertTrue(t, errors.Is(err, ErrActionFailed))
			testtool.AssertTrue(t, errors.Is(err, ErrCompensationSuccess))
			testtool.AssertTrue(t, actionCalls == 1)
			testtool.AssertTrue(t, compensationCalls == 3)
		})
	})
}
