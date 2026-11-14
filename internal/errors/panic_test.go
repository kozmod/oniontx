package errors

import (
	stderrors "errors"
	"fmt"
	"testing"
)

func Test_WrapPanic(t *testing.T) {
	t.Run("error_fmt", func(t *testing.T) {
		var (
			panicErr = fmt.Errorf("some_panic")
			err      = WrapPanic(panicErr)
		)

		if !stderrors.Is(err, panicErr) {
			t.Fatalf("expected wrapped panic error to preserve cause: %v", err)
		}
	})
	t.Run("error_new", func(t *testing.T) {
		var (
			panicErr = stderrors.New("some_panic")
			err      = WrapPanic(panicErr)
		)

		if !stderrors.Is(err, panicErr) {
			t.Fatalf("expected wrapped panic error to preserve cause: %v", err)
		}
	})
}
