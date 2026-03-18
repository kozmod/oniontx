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

// NoError was added to avoid to use external dependencies for mocking
func NoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("error arose: %v", err)
	}
}

// Error was added to avoid to use external dependencies for mocking
func Error(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("error expected")
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
		t.Fatal()
	}
}

func ErrorIsNot(t *testing.T, err, target error) {
	t.Helper()
	if errors.Is(err, target) {
		t.Fatal()
	}
}
