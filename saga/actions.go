package saga

import (
	"context"
)

// ActionFunc represents a function that performs an action and can return an error.
// It is designed to be used in workflows, sagas, or any operation that might need
// additional behavior like panic recovery or retries.
type ActionFunc func(ctx context.Context, track Track) error

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
//	action := someAction.WithBeforeHook(func(ctx context.Context, _ Track) error {
//	    log.Println("starting action")
//	    return validateInput(ctx)
//	})
func (a ActionFunc) WithBeforeHook(before func(ctx context.Context, track Track) error) ActionFunc {
	return func(ctx context.Context, track Track) error {
		err := before(ctx, track)
		if err != nil {
			return err
		}
		return a(ctx, track)
	}
}

// WithAfterHook adds an after-hook to the ActionFunc.
// The hook executes after the main action and receives the execution track,
// which contains information about the action's outcome (success/failure and errors).
// The hook can inspect, log, or modify the track state.
// Returns a new ActionFunc with the after-hook applied.
//
// Use cases:
//   - Logging success/failure
//   - Metrics collection
//   - Resource cleanup
//   - Error enrichment via track.AddError() or track.SetFailedOnError()
//
// Example:
//
//	action := someAction.WithAfterHook(func(ctx context.Context, track Track) error {
//	    data := track.GetData()
//	    if data.Action.Status == ExecutionStatusFail {
//	        log.Printf("action failed with errors: %v", data.Action.Errors)
//	        return nil
//	    }
//	    log.Println("action completed successfully")
//	    return nil
//	})
func (a ActionFunc) WithAfterHook(after func(ctx context.Context, track Track) error) ActionFunc {
	return func(ctx context.Context, track Track) error {
		err := a(ctx, track)
		if err != nil {
			track.SetFailedOnError(err)
		}
		err = after(ctx, track)
		if err != nil {
			return err
		}
		return nil
	}
}

// WithWrapper adds a custom wrapper to the ActionFunc that can modify its behavior.
// The wrapper receives the context, the track, and the original ActionFunc, and returns an error.
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
// Example:
//
//	action := someAction.WithWrapper(func(ctx context.Context, track Track, action ActionFunc) error {
//	    start := time.Now()
//	    err := action(ctx, track)
//	    duration := time.Since(start)
//	    metrics.RecordActionDuration(duration, err)
//	    return err
//	})
func (a ActionFunc) WithWrapper(wrapper func(ctx context.Context, track Track, action ActionFunc) error) ActionFunc {
	return func(ctx context.Context, track Track) error {
		return wrapper(ctx, track, a)
	}
}

// CompensationFunc represents a function that performs compensation logic
// when an error occurs in the main action.
// It receives both the context and the error that triggered the compensation,
// allowing it to make decisions based on the specific error that occurred.
type CompensationFunc func(ctx context.Context, track Track) error

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
	return func(ctx context.Context, track Track) error {
		fn := func(ctx context.Context, track Track) error {
			return c(ctx, track)
		}
		return WithPanicRecovery(fn)(ctx, track)
	}
}

// WithRetry wraps the CompensationFunc with retry logic.
// The compensation function will be retried according to the provided RetryPolicy.
// The original aroseErr is preserved and passed through to each retry attempt.
// If all retry attempts fail, the behavior depends on RetryPolicy.ReturnAllAroseErr.
// Returns a new CompensationFunc with retry logic enabled.
func (c CompensationFunc) WithRetry(opt RetryPolicy) CompensationFunc {
	return func(ctx context.Context, track Track) error {
		fn := func(ctx context.Context, track Track) error {
			return c(ctx, track)
		}
		return WithRetry(opt, fn)(ctx, track)
	}
}

// WithBeforeHook adds a before-hook to the CompensationFunc.
// The hook executes before the compensation and receives the track,
// which contains information about the original action's failure.
// If the hook returns an error, the compensation is skipped and the error is returned.
// Returns a new CompensationFunc with the before-hook applied.
//
// Use cases:
//   - Check if compensation is needed based on error type
//   - Logging compensation attempts
//   - Pre-compensation validation
//   - Conditional compensation based on step state
//
// Example:
//
//	compensation := someCompensation.WithBeforeHook(func(ctx context.Context, track Track) error {
//	    data := track.GetData()
//	    if data.Action.Status == ExecutionStatusFail {
//	        // Only compensate if the action actually failed
//	        return nil
//	    }
//	    return ErrSkipCompensation
//	})
func (c CompensationFunc) WithBeforeHook(before func(ctx context.Context, track Track) error) CompensationFunc {
	return func(ctx context.Context, track Track) error {
		err := before(ctx, track)
		if err != nil {
			return err
		}
		return c(ctx, track)
	}
}

// WithAfterHook adds an after-hook to the CompensationFunc.
// The hook executes after the compensation and receives the track,
// which contains information about both the original action and the compensation outcome.
// The hook can inspect, log, or modify the compensation error via track.
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
//	compensation := someCompensation.WithAfterHook(func(ctx context.Context, track Track) error {
//	    data := track.GetData()
//	    if data.Compensation.Status == ExecutionStatusFail {
//	        log.Printf("CRITICAL: Compensation failed: %v", data.Compensation.Errors)
//	        monitoring.Alert(ctx, "compensation_failed", data.Compensation.Errors)
//	        return fmt.Errorf("compensation failed: %w", data.Compensation.Errors[0])
//	    }
//	    log.Printf("Compensation successful for action: %s", data.StepName)
//	    return nil
//	})
func (c CompensationFunc) WithAfterHook(after func(ctx context.Context, track Track) error) CompensationFunc {
	return func(ctx context.Context, track Track) error {
		err := c(ctx, track)
		if err != nil {
			track.SetFailedOnError(err)
		}
		err = after(ctx, track)
		if err != nil {
			return err
		}

		return nil
	}
}

// WithWrapper adds a custom wrapper to the CompensationFunc that can modify its behavior.
// The wrapper receives the context, the track, and the CompensationFunc,
// and returns an error. This provides maximum flexibility for cross-cutting concerns
// specific to compensation logic.
//
// Use cases:
//   - Timing/deadline enforcement for compensations
//   - Circuit breaking specifically for compensations
//   - Rate limiting compensation calls
//   - Distributed tracing with error context
//   - Conditional compensation based on step state
//   - Compensation attempt counting and alerting
//   - Dead letter queue integration for failed compensations
//
// The wrapper must call the provided CompensationFunc to execute the original compensation logic.
//
// Example with error classification:
//
//	compensation := someCompensation.WithWrapper(func(ctx context.Context, track Track, comp CompensationFunc) error {
//	    data := track.GetData()
//	    if errors.Is(data.Action.Errors[0], ErrNonCritical) {
//	        log.Printf("Skipping compensation for non-critical error")
//	        return nil
//	    }
//	    return comp(ctx, track)
//	})
func (c CompensationFunc) WithWrapper(wrapper func(ctx context.Context, track Track, comp CompensationFunc) error) CompensationFunc {
	return func(ctx context.Context, track Track) error {
		return wrapper(ctx, track, c)
	}
}
