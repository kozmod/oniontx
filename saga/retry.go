package saga

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"time"
)

// RetryOpt defines the interface for retry strategy configuration.
// It allows different retry strategies (simple, backoff, jitter) to be
// used interchangeably with the WithRetry function
type RetryOpt interface {
	Attempts() uint32
	Delay(attempt uint32) time.Duration

	// ReturnAllAroseErr indicates whether to return all collected errors
	// from failed attempts (true) or just the last error (false).
	ReturnAllAroseErr() bool
}

// baseRetryOpt provides common fields and basic implementation for retry options.
type baseRetryOpt struct {
	attempts          uint32
	delay             time.Duration
	maxDelay          time.Duration
	returnAllAroseErr bool
}

// Attempts returns the configured maximum number of retry attempts.
func (o baseRetryOpt) Attempts() uint32 {
	return o.attempts
}

// ReturnAllAroseErr returns the configured error aggregation behavior.
func (o baseRetryOpt) ReturnAllAroseErr() bool {
	return o.returnAllAroseErr
}

// Delay returns a constant delay duration regardless of attempt number.
func (o baseRetryOpt) Delay(_ uint32) time.Duration {
	return o.delay
}

// BaseRetryOpt provides fixed-interval retry configuration.
// Each retry attempt waits the same amount of time.
type BaseRetryOpt struct {
	baseRetryOpt
}

// NewBaseRetryOpt creates a new fixed-interval retry option.
func NewBaseRetryOpt(attempts uint32, delay time.Duration) *BaseRetryOpt {
	return &BaseRetryOpt{
		baseRetryOpt: baseRetryOpt{
			attempts: attempts,
			delay:    delay,
			maxDelay: -1,
		},
	}
}

// WithReturnAllAroseErr enables returning all errors from failed attempts.
func (o BaseRetryOpt) WithReturnAllAroseErr() BaseRetryOpt {
	o.returnAllAroseErr = true
	return o
}

// FullJitterRetryOpt provides exponential backoff with full jitter strategy.
type FullJitterRetryOpt struct {
	baseRetryOpt
}

// NewFullJitterRetryOpt creates a new full jitter retry option.
func NewFullJitterRetryOpt(attempts uint32, delay time.Duration) FullJitterRetryOpt {
	return FullJitterRetryOpt{
		baseRetryOpt: baseRetryOpt{
			attempts: attempts,
			delay:    delay,
			maxDelay: -1,
		},
	}
}

// WithReturnAllAroseErr enables returning all errors from failed attempts.
func (o FullJitterRetryOpt) WithReturnAllAroseErr() FullJitterRetryOpt {
	o.baseRetryOpt.returnAllAroseErr = true
	return o
}

// WithMaxDelay sets an upper bound for the delay duration.
func (o FullJitterRetryOpt) WithMaxDelay(delay time.Duration) FullJitterRetryOpt {
	o.baseRetryOpt.maxDelay = delay
	return o
}

// Delay calculates the wait time before the specified attempt using full jitter strategy.
func (o FullJitterRetryOpt) Delay(attempt uint32) time.Duration {
	var (
		r           = rand.New(rand.NewSource(time.Now().UnixNano()))
		backoffTime = o.baseRetryOpt.delay * time.Duration(math.Pow(2, float64(attempt)))
	)
	if maxDelay := o.baseRetryOpt.maxDelay; maxDelay > 0 && backoffTime > maxDelay {
		if backoffTime > maxDelay {
			backoffTime = maxDelay
		}
	}
	return time.Duration(r.Int63n(int64(backoffTime)))
}

// BackoffRetryOpt provides exponential backoff without jitter strategy.
type BackoffRetryOpt struct {
	baseRetryOpt
}

// NewBackoffRetryOpt creates a new exponential backoff retry option.
func NewBackoffRetryOpt(attempts uint32, delay time.Duration) BackoffRetryOpt {
	return BackoffRetryOpt{
		baseRetryOpt: baseRetryOpt{
			attempts: attempts,
			delay:    delay,
			maxDelay: -1,
		},
	}
}

// WithReturnAllAroseErr enables returning all errors from failed attempts.
func (o BackoffRetryOpt) WithReturnAllAroseErr() BackoffRetryOpt {
	o.baseRetryOpt.returnAllAroseErr = true
	return o
}

// WithMaxDelay sets an upper bound for the delay duration.
func (o BackoffRetryOpt) WithMaxDelay(delay time.Duration) BackoffRetryOpt {
	o.baseRetryOpt.maxDelay = delay
	return o
}

// Delay calculates the wait time before the specified attempt using exponential backoff.
func (o BackoffRetryOpt) Delay(attempt uint32) time.Duration {
	backoffTime := o.baseRetryOpt.delay * time.Duration(math.Pow(2, float64(attempt)))
	if maxDelay := o.baseRetryOpt.maxDelay; maxDelay > 0 && backoffTime > maxDelay {
		return maxDelay
	}
	return backoffTime
}

// WithRetry returns a function with retry logic for execution.
// It makes the specified number of attempts to call fn until it succeeds.
// Between attempts, it waits for the delay determined by the RetryOpt strategy.
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
func WithRetry(opt RetryOpt, fn func(ctx context.Context) error) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		attempts := opt.Attempts()
		if attempts == 0 {
			return nil
		}

		var (
			err      error
			retryErr []error
		)

		for i := uint32(0); i < attempts; i++ {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				err = fn(ctx)
				if err == nil {
					break
				}
				retryErr = append(retryErr, fmt.Errorf("retry [%d]: %w", i, err))
				time.Sleep(opt.Delay(i))
			}
		}

		if opt.ReturnAllAroseErr() {
			return errors.Join(retryErr...)
		}
		return err
	}
}
