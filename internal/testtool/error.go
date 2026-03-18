package testtool

import "fmt"

var (
	// ErrExpTestA is an error for tests.
	ErrExpTestA = fmt.Errorf("exp_test_error_A")
	// ErrExpTestB is an errors for tests.
	ErrExpTestB = fmt.Errorf("exp_test_error_B")
)
