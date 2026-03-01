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
	//   - Maximum number of retry attempts
	//   - Delay calculation based on attempt number
	//   - Error aggregation behavior configuration
	RetryPolicy interface {
		Attempts() uint32
		Delay(attempt uint32) time.Duration

		// ReturnAllAroseErr indicates whether to return all collected errors
		// from failed attempts (true) or just the last error (false).
		ReturnAllAroseErr() bool
	}

	// Backoff defines the interface for backoff strategy calculation.
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
	attempts          uint32
	delay             time.Duration
	maxDelay          time.Duration
	returnAllAroseErr bool
}

// Attempts returns the configured maximum number of retry attempts.
func (o baseRetryPolicy) Attempts() uint32 {
	return o.attempts
}

// ReturnAllAroseErr returns the configured error aggregation behavior.
func (o baseRetryPolicy) ReturnAllAroseErr() bool {
	return o.returnAllAroseErr
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

// NewBaseRetryOpt creates a new fixed-interval retry option.
func NewBaseRetryOpt(attempts uint32, delay time.Duration) *BaseRetryPolicy {
	return &BaseRetryPolicy{
		baseRetryPolicy: baseRetryPolicy{
			attempts: attempts,
			delay:    delay,
			maxDelay: -1,
		},
	}
}

// WithReturnAllAroseErr enables returning all errors from failed attempts.
func (o BaseRetryPolicy) WithReturnAllAroseErr() BaseRetryPolicy {
	o.returnAllAroseErr = true
	return o
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

// WithReturnAllAroseErr enables returning all errors from failed attempts.
func (o AdvanceRetryPolicy) WithReturnAllAroseErr() AdvanceRetryPolicy {
	o.baseRetryPolicy.returnAllAroseErr = true
	return o
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

// Attempts returns the configured maximum number of retry attempts.
func (o AdvanceRetryPolicy) Attempts() uint32 {
	return o.attempts
}

// ReturnAllAroseErr returns the configured error aggregation behavior.
func (o AdvanceRetryPolicy) ReturnAllAroseErr() bool {
	return o.returnAllAroseErr
}

// Delay returns a constant delay duration regardless of attempt number.
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
// It makes the specified number of attempts to call fn until it succeeds.
// Between attempts, it waits for the delay determined by the RetryPolicy strategy.
// The function respects context cancellation and will return context.Err() if done.
//
// Behavior:
//   - If Attempts() = 0, the function immediately returns nil without executing fn
//   - Before each attempt, it waits for the delay provided by opt.Delay(attempt)
//   - The function stops retrying on first successful execution
//   - Context cancellation is respected between attempts (checked before each attempt)
//   - All errors from failed attempts are collected
//
// If all attempts fail, behavior depends on ReturnAllAroseErr():
//   - if true - returns all errors via errors.Join(...)
//   - if false - returns only the last error that occurred
func WithRetry(opt RetryPolicy, fn func(ctx context.Context) error) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		// first call
		var (
			attempts  = opt.Attempts()
			retryErrs []error
		)

		err := fn(ctx)
		switch {
		case err == nil:
			return nil
		case attempts == 0:
			return err
		case err != nil:
			err = fmt.Errorf("action error: %w", err)
			retryErrs = append(retryErrs, err)
		}

		// retries
	stop:
		for i := uint32(0); i < attempts; i++ {
			select {
			case <-ctx.Done():
				err = errors.Join(ErrRetryContextDone, ctx.Err())
				retryErrs = append(retryErrs, fmt.Errorf("retry [%d]: %w", i, err))
				break stop
			default:
				err = fn(ctx)
				if err == nil {
					break stop
				}
				retryErrs = append(retryErrs, fmt.Errorf("retry [%d]: %w", i, err))
				time.Sleep(opt.Delay(i))
			}
		}

		if opt.ReturnAllAroseErr() {
			return errors.Join(retryErrs...)
		}
		return err
	}
}
