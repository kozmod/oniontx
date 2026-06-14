package saga

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type (
	// RetryPolicy defines the interface for retry strategy configuration.
	// It allows different retry strategies (simple, backoff, jitter) to be
	// used interchangeably with the WithRetry function.
	//
	// Implementations should provide:
	//   - Number of retry attempts after the initial call
	//   - Delay calculation based on attempt number
	RetryPolicy interface {
		Attempts() uint32
		Delay(attempt uint32) time.Duration
	}

	// Backoff defines the interface for retry delay calculation.
	Backoff interface {
		Backoff(attempts uint32, delay time.Duration) time.Duration
	}

	// Jitter defines the interface for jitter strategy calculation.
	Jitter interface {
		Jitter(base time.Duration) time.Duration
	}
)

// baseRetryPolicy provides common fields and basic implementation for retry options.
type baseRetryPolicy struct {
	attempts uint32
	delay    time.Duration
	maxDelay time.Duration
}

// Attempts returns the configured number of retry attempts after the initial call.
func (o baseRetryPolicy) Attempts() uint32 {
	return o.attempts
}

// Delay returns a constant delay duration regardless of attempt number.
func (o baseRetryPolicy) Delay(_ uint32) time.Duration {
	return o.delay
}

// BaseRetryPolicy provides fixed-interval retry configuration.
// Each retry attempt waits the same amount of time.
type BaseRetryPolicy struct {
	baseRetryPolicy
}

// NewBaseRetryOpt creates a new fixed-interval retry policy.
func NewBaseRetryOpt(attempts uint32, delay time.Duration) *BaseRetryPolicy {
	return &BaseRetryPolicy{
		baseRetryPolicy: baseRetryPolicy{
			attempts: attempts,
			delay:    delay,
			maxDelay: -1,
		},
	}
}

// AdvanceRetryPolicy provides configurable retry behavior with pluggable
// backoff and jitter strategies. This allows for flexible composition of
// different retry algorithms.
type AdvanceRetryPolicy struct {
	baseRetryPolicy
	backoff Backoff
	jitter  Jitter
}

// NewAdvanceRetryPolicy creates a new advanced retry policy with the specified
// backoff strategy.
func NewAdvanceRetryPolicy(attempts uint32, delay time.Duration, backoff Backoff) AdvanceRetryPolicy {
	return AdvanceRetryPolicy{
		baseRetryPolicy: baseRetryPolicy{
			attempts: attempts,
			delay:    delay,
			maxDelay: -1,
		},
		backoff: backoff,
	}
}

// WithJitter adds jitter to the retry policy.
func (o AdvanceRetryPolicy) WithJitter(jitter Jitter) AdvanceRetryPolicy {
	o.jitter = jitter
	return o
}

// WithMaxDelay sets an upper bound for the delay duration.
func (o AdvanceRetryPolicy) WithMaxDelay(delay time.Duration) AdvanceRetryPolicy {
	o.baseRetryPolicy.maxDelay = delay
	return o
}

// Attempts returns the configured number of retry attempts after the initial call.
func (o AdvanceRetryPolicy) Attempts() uint32 {
	return o.attempts
}

// Delay returns the backoff delay for the given retry attempt.
func (o AdvanceRetryPolicy) Delay(i uint32) time.Duration {
	var (
		backoffTime = o.backoff.Backoff(i, o.delay)
	)
	if maxDelay := o.baseRetryPolicy.maxDelay; maxDelay > 0 && backoffTime > maxDelay {
		if backoffTime > maxDelay {
			backoffTime = maxDelay
		}
	}
	if o.jitter != nil {
		backoffTime = o.jitter.Jitter(backoffTime)
		return backoffTime
	}
	return backoffTime
}

// WithRetry returns a function with retry logic for execution.
// It makes the initial call and then up to opt.Attempts() retry calls until fn succeeds.
// Between attempts, it waits for the delay determined by the RetryPolicy strategy.
// Context cancellation is checked before each retry attempt.
//
// Behavior:
//   - If attempts = 0, the function executes fn once and returns its result
//   - On first successful execution, returns nil and sets status to ExecutionStatusSuccess
//   - On failure, retries up to opt.Attempts() times with delays between attempts
//   - Context cancellation is respected before each attempt
//   - All errors from failed attempts are collected in Track.Errors
//   - Each retry attempt is wrapped with a "retry [N]" prefix for error context
//
// Return values:
//   - nil if any attempt succeeds
//   - ErrRetryContextDone if context is cancelled before a retry attempt
//   - ErrRetryFailed if all attempts fail (including the initial call)
//
// The original error from the final attempt is stored in the track and ErrRetryFailed is returned.
func WithRetry(opt RetryPolicy, fn func(ctx context.Context, track Track) error) func(context.Context, Track) error {
	return func(ctx context.Context, track Track) error {
		// first call
		var (
			attempts = opt.Attempts()
		)

		err := fn(ctx, track)
		switch {
		case err == nil:
			track.SetStatus(ExecutionStatusSuccess)
			return nil
		case attempts == 0:
			return err
		case err != nil:
			track.SetStatus(ExecutionStatusFail)
			track.AddError(err)
		}

		// retries
	stop:
		for i := uint32(0); i < attempts; i++ {
			track.SetParentError(fmt.Errorf("retry [%d]", i))
			select {
			case <-ctx.Done():
				err = ctx.Err()
				track.SetStatus(ExecutionStatusFail)
				track.AddError(
					errors.Join(ErrRetryContextDone, err),
				)

				break stop
			default:
				err = fn(ctx, track)
				if err == nil {
					track.SetStatus(ExecutionStatusSuccess)
					break stop
				}
				track.SetStatus(ExecutionStatusFail)
				track.AddError(err)
				if i < attempts-1 {
					time.Sleep(opt.Delay(i))
				}
			}
		}

		track.SetParentError(nil)
		if err != nil {
			return ErrRetryFailed
		}
		return nil
	}
}
