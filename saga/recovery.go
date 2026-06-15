package saga

import (
	"context"
	"errors"

	"github.com/kozmod/oniontx/internal/tool"
)

// WithPanicRecovery returns an OperationFunc with panic recovery logic.
// The returned function recovers from panics raised by fn and converts them to
// an error that includes both ErrPanicRecovered and the original panic value.
//
// Example:
//
//	fn := func(ctx context.Context, track Track) error {
//	    panic("something went wrong")
//	}
//	safeFn := WithPanicRecovery(fn)
//	err := safeFn(ctx, track)
//
// err will wrap ErrPanicRecovered and the panic value.
func WithPanicRecovery(fn func(ctx context.Context, track Track) error) OperationFunc {
	return func(ctx context.Context, track Track) (err error) {
		defer func() {
			if p := recover(); p != nil {
				err = errors.Join(ErrPanicRecovered, tool.WrapPanic(p))
			}
		}()
		err = fn(ctx, track)
		return err
	}
}
