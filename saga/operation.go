package saga

import (
	"context"
)

var (
	dummyOperation OperationFunc = func(ctx context.Context, track Track) error {
		return nil
	}
)

// wrapCall records an operation call before invoking fn.
// A nil fn is treated as a no-op operation.
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
// It is designed to be used as either the main step action or its compensation.
type OperationFunc func(ctx context.Context, track Track) error

// Operation wraps an OperationFunc with optional behavior such as retry,
// panic recovery, and hooks.
type Operation struct {
	fn OperationFunc
}

// NewOperation creates a new Operation.
// The operation increments the provided Track call counter before invoking op.
// Passing nil creates a no-op operation.
func NewOperation(op OperationFunc) Operation {
	return Operation{
		fn: wrapCall(op),
	}
}

// WithPanicRecovery wraps the Operation with panic recovery logic.
// If the original function panics, the panic is recovered and returned as an error
// that wraps both the original panic value and ErrPanicRecovered.
// Returns a new Operation with panic recovery enabled.
func (a Operation) WithPanicRecovery() Operation {
	a.fn = WithPanicRecovery(a.fn)
	return a
}

// WithRetry wraps the Operation with retry logic.
// The function will be retried according to the provided RetryOptions.
// Returns a new Operation with retry logic enabled.
func (a Operation) WithRetry(opt RetryPolicy) Operation {
	a.fn = WithRetry(opt, a.fn)
	return a
}

// WithBeforeHook adds a before-hook to the Operation.
// The hook executes before the wrapped operation. If the hook returns an error,
// the wrapped operation is skipped and the hook error becomes the operation error.
// Returns a new Operation with the before-hook applied.
//
// Use cases:
//   - Validation
//   - Authentication/Authorization
//   - Resource acquisition
//   - Conditional short-circuiting
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

// WithAfterHook adds an after-hook to the Operation.
// The hook executes only after the wrapped operation succeeds. If the wrapped
// operation returns an error, the after-hook is skipped and that error is returned.
// If the after-hook returns an error, that error becomes the operation error.
// Returns a new Operation with the after-hook applied.
//
// Use cases:
//   - Logging success
//   - Metrics collection
//   - Resource cleanup
//   - Post-operation validation
//
// Example:
//
//	action := someAction.WithAfterHook(func(ctx context.Context, track Track) error {
//	    data := track.GetStepData()
//	    log.Printf("action calls: %d", data.Action.Calls)
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
