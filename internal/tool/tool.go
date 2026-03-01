package tool

import "fmt"

// WrapPanic transforms panic to error.
func WrapPanic(p any) error {
	return fmt.Errorf("panic [%v]", p)
}
