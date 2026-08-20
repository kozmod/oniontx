package saga

import (
	"context"
	"time"

	"github.com/kozmod/oniontx/internal/errors"
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

// NewBaseRetryPolicy creates a new fixed-interval retry policy.
func NewBaseRetryPolicy(attempts uint32, delay time.Duration) *BaseRetryPolicy {
	return &BaseRetryPolicy{
		baseRetryPolicy: baseRetryPolicy{
			attempts: attempts,
			delay:    delay,
			maxDelay: -1,
		},
	}
}

// AdvancedRetryPolicy provides configurable retry behavior with pluggable
// backoff and jitter strategies. This allows for flexible composition of
// different retry algorithms.
type AdvancedRetryPolicy struct {
	baseRetryPolicy
	backoff Backoff
	jitter  Jitter
}

// NewAdvancedRetryPolicy creates a new advanced retry policy with the specified
// backoff strategy. A nil backoff uses ExponentialBackoff.
func NewAdvancedRetryPolicy(attempts uint32, delay time.Duration, backoff Backoff) AdvancedRetryPolicy {
	if backoff == nil {
		backoff = NewExponentialBackoff()
	}

	return AdvancedRetryPolicy{
		baseRetryPolicy: baseRetryPolicy{
			attempts: attempts,
			delay:    delay,
			maxDelay: -1,
		},
		backoff: backoff,
	}
}

// WithJitter adds jitter to the retry policy.
func (o AdvancedRetryPolicy) WithJitter(jitter Jitter) AdvancedRetryPolicy {
	o.jitter = jitter
	return o
}

// WithMaxDelay sets an upper bound for the delay duration.
func (o AdvancedRetryPolicy) WithMaxDelay(delay time.Duration) AdvancedRetryPolicy {
	o.maxDelay = delay
	return o
}

// Attempts returns the configured number of retry attempts after the initial call.
func (o AdvancedRetryPolicy) Attempts() uint32 {
	return o.attempts
}

// Delay returns the backoff delay for the given retry attempt.
func (o AdvancedRetryPolicy) Delay(i uint32) time.Duration {
	backoffTime := o.limitDelay(o.backoff.Backoff(i, o.delay))
	if o.jitter != nil {
		backoffTime = o.limitDelay(o.jitter.Jitter(backoffTime))
	}
	return backoffTime
}

func (o AdvancedRetryPolicy) limitDelay(delay time.Duration) time.Duration {
	if o.maxDelay > 0 && delay > o.maxDelay {
		return o.maxDelay
	}
	return delay
}

// WithRetry returns a function with retry logic for execution.
// It makes the initial call and then up to opt.Attempts() retry calls until fn succeeds.
// Between attempts, it waits for the delay determined by the RetryPolicy strategy.
// Context cancellation is checked before each retry attempt and while waiting
// between attempts.
//
// Behavior:
//   - If attempts = 0, the function executes fn once and returns its result
//   - On first successful execution, returns nil and sets status to ExecutionStatusSuccess
//   - On failure, retries up to opt.Attempts() times with delays between attempts
//   - Context cancellation is respected before each attempt and during retry delays
//   - All errors from failed attempts are collected in Track.Errors
//   - Each retry attempt is wrapped with a "retry [N]" prefix for error context
//
// Return values:
//   - nil if any attempt succeeds
//   - ErrRetryContextDone and the context error if context is cancelled before a retry attempt or during a retry delay
//   - ErrRetryFailed if all attempts fail (including the initial call)
//
// A failure returns ErrRetryFailed joined with the cause of the final failed
// attempt. This preserves context cancellation and lets callers use errors.Is.
func WithRetry(opt RetryPolicy, fn func(ctx context.Context, track Track) error) func(context.Context, Track) error {
	return func(ctx context.Context, track Track) error {
		// first call
		var (
			attempts = opt.Attempts()
		)

		err := fn(ctx, track)
		switch {
		case err == nil:
			applyTrackAct(track, newTrackSucceededAct())
			return nil
		case attempts == 0:
			return err
		case err != nil:
			applyTrackAct(track, newTrackFailedAct(err))
		}

		// retries
	stop:
		for i := range attempts {
			retryTrack := newExecutionRetryTrack(track, i)
			select {
			case <-ctx.Done():
				err = errors.Join(ErrRetryContextDone, ctx.Err())
				retryTrack.apply(newTrackFailedAct(err))

				break stop
			default:
				err = fn(ctx, retryTrack)
				if err == nil {
					retryTrack.apply(newTrackSucceededAct())
					break stop
				}
				retryTrack.apply(newTrackFailedAct(err))
				if i < attempts-1 {
					select {
					case <-ctx.Done():
						err = errors.Join(ErrRetryContextDone, ctx.Err())
						retryTrack.apply(newTrackFailedAct(err))
						break stop
					case <-time.After(opt.Delay(i)):
					}
				}
			}
		}

		if err != nil {
			return errors.Join(ErrRetryFailed, err)
		}
		return nil
	}
}
