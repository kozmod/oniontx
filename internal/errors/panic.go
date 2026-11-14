package errors

import (
	"fmt"
)

// WrapPanic transforms panic to error.
func WrapPanic(p any) error {
	if err, ok := p.(error); ok {
		return fmt.Errorf("panic [%w]", err)
	}
	return fmt.Errorf("panic [%v]", p)
}
