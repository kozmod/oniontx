package saga

import (
	"context"
	"errors"
	"fmt"
)

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
