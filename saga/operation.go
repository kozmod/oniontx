package saga

import (
	"context"
)

var (
	dummyOperation OperationFunc = func(ctx context.Context, track Track) error {
		return nil
	}
)

func wrapCall(fn OperationFunc) OperationFunc {
	if fn == nil {
		fn = dummyOperation
	}
	return func(ctx context.Context, track Track) (err error) {
		track.Call()
		err = fn(ctx, track)
		return err
	}
}

// OperationFunc represents a function that performs an action and can return an error.
// It is designed to be used in workflows, sagas, or any operation that might need
// additional behavior like panic recovery or retries.
type OperationFunc func(ctx context.Context, track Track) error

type Operation struct {
	fn OperationFunc
}

// NewOperation creates a new OperationFunc.
func NewOperation(op OperationFunc) Operation {
	return Operation{
		fn: wrapCall(op),
	}
}

// WithPanicRecovery wraps the OperationFunc with panic recovery logic.
// If the original function panics, the panic is recovered and returned as an error
// that wraps both the original panic value and ErrPanicRecovered.
// Returns a new OperationFunc with panic recovery enabled.
func (a Operation) WithPanicRecovery() Operation {
	a.fn = WithPanicRecovery(a.fn)
	return a
}

// WithRetry wraps the OperationFunc with retry logic.
// The function will be retried according to the provided RetryOptions.
// If all retry attempts fail, the behavior depends on RetryOptions.ReturnAllAroseErr.
// Returns a new OperationFunc with retry logic enabled.
func (a Operation) WithRetry(opt RetryPolicy) Operation {
	a.fn = WithRetry(opt, a.fn)
	return a
}

// WithBeforeHook adds a before-hook to the OperationFunc.
// The hook executes before the main action and if it returns an error,
// the main action is skipped and the error is returned immediately.
// Returns a new OperationFunc with the before-hook applied.
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
func (a Operation) WithBeforeHook(before OperationFunc) Operation {
	fn := a.fn
	a.fn = func(ctx context.Context, track Track) error {
		err := before(ctx, track)
		if err != nil {
			return err
		}
		return fn(ctx, track)
	}
	return a
}

// WithAfterHook adds an after-hook to the OperationFunc.
// The hook executes after the main action and receives the execution track,
// which contains information about the action's outcome (success/failure and errors).
// The hook can inspect, log, or modify the track state.
// Returns a new OperationFunc with the after-hook applied.
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
func (a Operation) WithAfterHook(after OperationFunc) Operation {
	fn := a.fn
	a.fn = func(ctx context.Context, track Track) error {
		err := fn(ctx, track)
		if err != nil {
			return err
		}
		err = after(ctx, track)
		if err != nil {
			return err
		}
		return nil
	}
	return a
}
