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

type (
	// OperationFunc represents a function that performs an action and can return an error.
	// It is designed to be used as either the main step action or its compensation.
	OperationFunc func(ctx context.Context, track Track) error
)

// CtxFactory derives an operation or compensation context from its parent.
type CtxFactory func(ctx context.Context) context.Context

// Apply transforms ctx using the factory.
// If the factory or the context returned by it is nil, Apply returns the
// original context unchanged.
func (f CtxFactory) Apply(ctx context.Context) context.Context {
	if f == nil {
		return ctx
	}

	newCtx := f(ctx)
	if newCtx == nil {
		return ctx
	}

	return newCtx
}

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
func (o Operation) WithPanicRecovery() Operation {
	o.fn = WithPanicRecovery(o.fn)
	return o
}

// WithRetry wraps the Operation with retry logic.
// The function will be retried according to the provided RetryOptions.
// Returns a new Operation with retry logic enabled.
func (o Operation) WithRetry(opt RetryPolicy) Operation {
	if opt == nil {
		return o
	}

	o.fn = WithRetry(opt, o.fn)
	return o
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
func (o Operation) WithBeforeHook(before OperationFunc) Operation {
	if before == nil {
		return o
	}

	fn := o.fn
	o.fn = func(ctx context.Context, track Track) error {
		err := before(ctx, track)
		if err != nil {
			return err
		}
		return fn(ctx, track)
	}
	return o
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
func (o Operation) WithAfterHook(after OperationFunc) Operation {
	if after == nil {
		return o
	}

	fn := o.fn
	o.fn = func(ctx context.Context, track Track) error {
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
	return o
}

// WithContext wraps the Operation with a context transformation.
// The factory receives the context passed to the operation, and its result is
// passed to the wrapped function. If the factory or its result is nil, the
// original context is used unchanged.
//
// This can be used to add operation-specific values or derive a context with
// different cancellation behavior. A nil factory leaves the Operation unchanged.
//
// When used for an action, WithContext does not bypass Saga cancellation checks:
// if the parent context is already done before the action starts, the action and
// its context factory are not called. Using context.WithoutCancel detaches a
// running action from subsequent parent cancellation, so the action may continue
// after the Saga context is done. Use this behavior deliberately.
//
// WithContext does not manage resources owned by the derived context. In
// particular, a factory must not discard a CancelFunc returned by
// context.WithCancel, context.WithDeadline, or context.WithTimeout.
//
// Decorators are applied in call order. When WithContext is applied after
// WithRetry, the context is created once for the entire retry sequence. When it
// is applied before WithRetry, the factory is called for every retry attempt.
//
// Example:
//
//	op := NewOperation(fn).WithContext(func(ctx context.Context) context.Context {
//		return context.WithValue(ctx, operationKey{}, operationID)
//	})
func (o Operation) WithContext(ctxFactory CtxFactory) Operation {
	if ctxFactory == nil {
		return o
	}

	fn := o.fn
	o.fn = func(ctx context.Context, track Track) error {
		return fn(ctxFactory.Apply(ctx), track)
	}
	return o
}
