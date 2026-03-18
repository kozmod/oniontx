package assert

import (
	"errors"
	"testing"
)

// True was added to avoid to use external dependencies for mocking
func True(t *testing.T, val bool) {
	t.Helper()
	if !val {
		t.Fatalf("expected true [current value: %v]", val)
	}
}

// False was added to avoid to use external dependencies for mocking
func False(t *testing.T, val bool) {
	t.Helper()
	if val {
		t.Fatalf("expected false [current value: %v]", val)
	}
}

// Error was added to avoid to use external dependencies for mocking
func Error(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("error expected")
	}
}

// NoError was added to avoid to use external dependencies for mocking
func NoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("error arose: %v", err)
	}
}

func Equal[T comparable](t *testing.T, expected, target T) {
	t.Helper()
	if expected != target {
		t.Fatalf("%v != %v", expected, target)
	}
}

// ErrorIs asserts that at least one of the errors in err's chain matches target.
// This is a wrapper for errors.Is.
func ErrorIs(t *testing.T, err, target error) {
	t.Helper()
	if !errors.Is(err, target) {
		t.Fatalf("[%v] is not [%v]", err, target)
	}
}

func ErrorIsNot(t *testing.T, err, target error) {
	t.Helper()
	if errors.Is(err, target) {
		t.Fatalf("[%v] is [%v]", err, target)
	}
}
