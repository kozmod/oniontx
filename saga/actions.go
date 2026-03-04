package saga

import (
	"context"
)

// ActionFunc represents a function that performs an action and can return an error.
// It is designed to be used in workflows, sagas, or any operation that might need
// additional behavior like panic recovery or retries.
type ActionFunc func(ctx context.Context) error

// NewAction creates a new ActionFunc.
func NewAction(afn ActionFunc) ActionFunc {
	return afn
}

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
func (a ActionFunc) WithRetry(opt RetryPolicy) ActionFunc {
	return WithRetry(opt, a)
}

// WithBeforeHook adds a before-hook to the ActionFunc.
// The hook executes before the main action and if it returns an error,
// the main action is skipped and the error is returned immediately.
// Returns a new ActionFunc with the before-hook applied.
//
// Use cases:
//   - Validation
//   - Authentication/Authorization
//   - Logging/Instrumentation
//   - Resource acquisition
//
// Example:
//
//	action := someAction.WithBeforeHook(func(ctx context.Context) error {
//	    log.Println("starting action")
//	    return validateInput(ctx)
//	})
func (a ActionFunc) WithBeforeHook(before func(ctx context.Context) error) ActionFunc {
	return func(ctx context.Context) error {
		err := before(ctx)
		if err != nil {
			return err
		}
		return a(ctx)
	}
}

// WithAfterHook adds an after-hook to the ActionFunc.
// The hook executes after the main action and receives both the context
// and the error returned by the action (nil if successful).
// The hook can inspect, log, or transform the error.
// Returns a new ActionFunc with the after-hook applied.
//
// Use cases:
//   - Logging success/failure
//   - Metrics collection
//   - Resource cleanup
//   - Error wrapping/enrichment
//
// Example:
//
//	action := someAction.WithAfterHook(func(ctx context.Context, err error) error {
//	    if err != nil {
//	        log.Printf("action failed: %v", err)
//	        return fmt.Errorf("operation failed: %w", err)
//	    }
//	    log.Println("action completed successfully")
//	    return nil
//	})
func (a ActionFunc) WithAfterHook(after func(ctx context.Context, aroseError error) error) ActionFunc {
	return func(ctx context.Context) error {
		err := a(ctx)
		err = after(ctx, err)
		if err != nil {
			return err
		}
		return err
	}
}

// WithWrapper adds a custom wrapper to the ActionFunc that can modify its behavior.
// The wrapper receives the context and the original ActionFunc, and returns an error.
// This provides maximum flexibility for cross-cutting concerns that don't fit into
// the standard before/after hook pattern.
//
// Use cases:
//   - Timing/deadline enforcement
//   - Circuit breaking
//   - Rate limiting
//   - Distributed tracing span creation
//   - Custom error handling logic
//   - Performance monitoring
//   - Feature flagging
//
// The wrapper must call the provided ActionFunc to execute the original logic,
// but can add behavior before, after, or around it.
//
// Example with timing:
//
//	action := someAction.WithWrapper(func(ctx context.Context, action ActionFunc) error {
//	    start := time.Now()
//	    err := action(ctx)
//	    duration := time.Since(start)
//	    metrics.RecordActionDuration(duration, err)
//	    return err
//	})
//
// Example with circuit breaker:
//
//	action := someAction.WithWrapper(func(ctx context.Context, action ActionFunc) error {
//	    if circuit.IsOpen() {
//	        return ErrCircuitOpen
//	    }
//	    err := action(ctx)
//	    circuit.RecordResult(err)
//	    return err
//	})
func (a ActionFunc) WithWrapper(wrapper func(ctx context.Context, action ActionFunc) error) ActionFunc {
	return func(ctx context.Context) error {
		return wrapper(ctx, a)
	}
}

// CompensationFunc represents a function that performs compensation logic
// when an error occurs in the main action.
// It receives both the context and the error that triggered the compensation,
// allowing it to make decisions based on the specific error that occurred.
type CompensationFunc func(ctx context.Context, actionErr error) error

// NewCompensation creates a new CompensationFunc.
func NewCompensation(afn CompensationFunc) CompensationFunc {
	return afn
}

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
// The compensation function will be retried according to the provided RetryPolicy.
// The original aroseErr is preserved and passed through to each retry attempt.
// If all retry attempts fail, the behavior depends on RetryPolicy.ReturnAllAroseErr.
// Returns a new CompensationFunc with retry logic enabled.
func (c CompensationFunc) WithRetry(opt RetryPolicy) CompensationFunc {
	return func(ctx context.Context, actionError error) error {
		fn := func(ctx context.Context) error {
			return c(ctx, actionError)
		}
		return WithRetry(opt, fn)(ctx)
	}
}

// WithBeforeHook adds a before-hook to the CompensationFunc.
// The hook executes before the compensation and receives the original action error.
// If the hook returns an error, the compensation is skipped and the error is returned.
// Returns a new CompensationFunc with the before-hook applied.
//
// Use cases:
//   - Check if compensation is needed based on error type
//   - Logging compensation attempts
//   - Pre-compensation validation
//
// Example:
//
//	compensation := someCompensation.WithBeforeHook(func(ctx context.Context, actionErr error) error {
//	    if errors.Is(actionErr, ErrTemporaryFailure) {
//	        return nil // Compensate for temporary failures
//	    }
//	    return ErrSkipCompensation // Skip compensation for permanent failures
//	})
func (c CompensationFunc) WithBeforeHook(before func(ctx context.Context, actionErr error) error) CompensationFunc {
	return func(ctx context.Context, actionErr error) error {
		err := before(ctx, actionErr)
		if err != nil {
			return err
		}
		return c(ctx, actionErr)
	}
}

// WithAfterHook adds an after-hook to the CompensationFunc.
// The hook executes after the compensation and receives both the original action error
// and the error returned by the compensation (nil if successful).
// The hook can inspect, log, or transform the compensation error.
// Returns a new CompensationFunc with the after-hook applied.
//
// Use cases:
//   - Logging compensation outcome
//   - Metrics collection
//   - Error enrichment
//   - Triggering alerts on compensation failures
//
// Example:
//
//	compensation := someCompensation.WithAfterHook(func(ctx context.Context, actionErr, compensationErr error) error {
//	    if compensationErr != nil {
//	        log.Printf("CRITICAL: Compensation failed for action error %v: %v", actionErr, compensationErr)
//	        monitoring.Alert(ctx, "compensation_failed", compensationErr)
//	        return fmt.Errorf("compensation error (original: %w): %w", actionErr, compensationErr)
//	    }
//	    log.Printf("Compensation successful for action error: %v", actionErr)
//	    return nil
//	})
func (c CompensationFunc) WithAfterHook(after func(ctx context.Context, actionErr, aroseErr error) error) CompensationFunc {
	return func(ctx context.Context, actionErr error) error {
		aroseErr := c(ctx, actionErr)
		err := after(ctx, actionErr, aroseErr)
		if err != nil {
			return err
		}
		return err
	}
}

// WithWrapper adds a custom wrapper to the CompensationFunc that can modify its behavior.
// The wrapper receives the context, the original action error, and the CompensationFunc,
// and returns an error. This provides maximum flexibility for cross-cutting concerns
// specific to compensation logic.
//
// Use cases:
//   - Timing/deadline enforcement for compensations
//   - Circuit breaking specifically for compensations
//   - Rate limiting compensation calls
//   - Distributed tracing with error context
//   - Conditional compensation based on error type
//   - Compensation attempt counting and alerting
//   - Dead letter queue integration for failed compensations
//
// The wrapper must call the provided CompensationFunc with the appropriate parameters
// to execute the original compensation logic.
//
// Example with error classification:
//
//	compensation := someCompensation.WithWrapper(func(ctx context.Context, actionErr error, comp CompensationFunc) error {
//	    if errors.Is(actionErr, ErrNonCritical) {
//	        log.Printf("Skipping compensation for non-critical error: %v", actionErr)
//	        return nil
//	    }
//	    return comp(ctx, actionErr)
//	})
func (c CompensationFunc) WithWrapper(wrapper func(ctx context.Context, actionErr error, comp CompensationFunc) error) CompensationFunc {
	return func(ctx context.Context, actionErr error) error {
		return wrapper(ctx, actionErr, c)
	}
}
