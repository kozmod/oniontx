package sage

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// ActionFunc represents a function that performs an action and can return an error.
// It is designed to be used in workflows, sagas, or any operation that might need
// additional behavior like panic recovery or retries.
type ActionFunc func(ctx context.Context) error

// WithPanicRecovery wraps the ActionFunc with panic recovery logic.
// If the original function panics, the panic is recovered and returned as an error
// that wraps both the original panic value and ErrPanicRecovered.
// Returns a new ActionFunc with panic recovery enabled.
func (a ActionFunc) WithPanicRecovery() ActionFunc {
	return WithPanicRecovery(a)
}

// WithRetry wraps the ActionFunc with retry logic.
// The function will be retried according to the provided RetryOptions.
// If all retry attempts fail, the behavior depends on RetryOptions.ReturnAllAroseErr.
// Returns a new ActionFunc with retry logic enabled.
func (a ActionFunc) WithRetry(opt RetryOptions) ActionFunc {
	return WithRetry(opt, a)
}

// CompensationFunc represents a function that performs compensation logic
// when an error occurs in the main action.
// It receives both the context and the error that triggered the compensation,
// allowing it to make decisions based on the specific error that occurred.
type CompensationFunc func(ctx context.Context, aroseErr error) error

// WithPanicRecovery wraps the CompensationFunc with panic recovery logic.
// If the compensation function panics, the panic is recovered and returned as an error
// that wraps both the original panic value and ErrPanicRecovered.
// The original aroseErr is preserved and passed through to the compensation function.
// Returns a new CompensationFunc with panic recovery enabled.
func (c CompensationFunc) WithPanicRecovery() CompensationFunc {
	return func(ctx context.Context, aroseErr error) error {
		fn := func(ctx context.Context) error {
			return c(ctx, aroseErr)
		}
		return WithPanicRecovery(fn)(ctx)
	}
}

// WithRetry wraps the CompensationFunc with retry logic.
// The compensation function will be retried according to the provided RetryOptions.
// The original aroseErr is preserved and passed through to each retry attempt.
// If all retry attempts fail, the behavior depends on RetryOptions.ReturnAllAroseErr.
// Returns a new CompensationFunc with retry logic enabled.
func (c CompensationFunc) WithRetry(opt RetryOptions) CompensationFunc {
	return func(ctx context.Context, aroseErr error) error {
		fn := func(ctx context.Context) error {
			return c(ctx, aroseErr)
		}
		return WithRetry(opt, fn)(ctx)
	}
}

// WithPanicRecovery returns a wrapped function that includes panic recovery logic.
// The returned function will recover from any panic that occurs during execution of fn
// and convert it to an error that includes both the original panic value and ErrPanicRecovered.
// This is particularly useful for long-running goroutines or when calling code that
// might panic and you want to handle it gracefully.
//
// Example:
//
//	fn := func(ctx context.Context) error {
//	    panic("something went wrong")
//	}
//	safeFn := WithPanicRecovery(fn)
//	err := safeFn(ctx)
//
// err will wrap "panic [something went wrong]" and ErrPanicRecovered
func WithPanicRecovery(fn func(ctx context.Context) error) func(ctx context.Context) error {
	return func(ctx context.Context) (err error) {
		defer func() {
			if p := recover(); p != nil {
				err = errors.Join(fmt.Errorf("panic [%v]", p), ErrPanicRecovered)
			}
		}()
		err = fn(ctx)
		return err
	}
}

// RetryOptions defines parameters for function execution retries.
type RetryOptions struct {
	// Attempts is a number of execution attempts (0 - skip execution).
	Attempts uint32

	// Delay before the first attempt
	Delay time.Duration

	// ReturnAllAroseErr return all errors arose through all retries if value is `true`.
	// Return only the last one if value is `false`.
	ReturnAllAroseErr bool
}

// WithRetry returns a function with retry logic for execution.
// It makes the specified number of attempts to call fn until it succeeds.
// Between attempts, it waits for the specified Delay using a ticker.
// The function respects context cancellation and will return context.Err() if done.
//
// Behavior:
//   - If Attempts = 0, the function immediately returns nil without executing fn
//   - Before each attempt, it waits for the Delay period
//   - The function stops retrying on first successful execution
//   - Context cancellation is respected between attempts
//   - All errors from failed attempts are collected
//
// If all attempts fail, behavior depends on ReturnAllAroseErr:
//   - if true - returns all errors via errors.Join(...)
//   - if false - returns only the last error that occurred
//
// Example:
//
//	opts := RetryOptions{
//	    Attempts: 3,
//	    Delay:    time.Second,
//	    ReturnAllAroseErr: true,
//	}
//
//	retryableFn := WithRetry(opts, func(ctx context.Context) error {
//	    return someOperation(ctx)
//	})
//
//	err := retryableFn(ctx) // Will try up to 3 times with 1s delays
func WithRetry(opt RetryOptions, fn func(ctx context.Context) error) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		if opt.Attempts == 0 {
			return nil
		}

		var (
			err      error
			retryErr []error
			ticker   = time.NewTicker(opt.Delay)
		)
		defer ticker.Stop()

		for i := uint32(0); i < opt.Attempts; i++ {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-ticker.C:
				err = fn(ctx)
				if err == nil {
					break
				}
				retryErr = append(retryErr, fmt.Errorf("retry [%d]: %w", i, err))
			}
		}

		if opt.ReturnAllAroseErr {
			return errors.Join(retryErr...)
		}
		return err
	}
}
