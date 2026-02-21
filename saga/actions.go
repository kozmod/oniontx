package saga

import (
	"context"
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
func (a ActionFunc) WithRetry(opt RetryOpt) ActionFunc {
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
// The compensation function will be retried according to the provided RetryOpt.
// The original aroseErr is preserved and passed through to each retry attempt.
// If all retry attempts fail, the behavior depends on RetryOpt.ReturnAllAroseErr.
// Returns a new CompensationFunc with retry logic enabled.
func (c CompensationFunc) WithRetry(opt RetryOpt) CompensationFunc {
	return func(ctx context.Context, aroseErr error) error {
		fn := func(ctx context.Context) error {
			return c(ctx, aroseErr)
		}
		return WithRetry(opt, fn)(ctx)
	}
}
