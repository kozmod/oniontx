package require

import (
	"errors"
	"reflect"
	"testing"
)

type (
	Integer interface {
		Signed | Unsigned
	}

	Signed interface {
		~int | ~int8 | ~int16 | ~int32 | ~int64
	}

	Unsigned interface {
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
	}
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

// Equal asserts that two objects are equal.
func Equal[T comparable](t *testing.T, expected, target T) {
	t.Helper()
	if expected != target {
		t.Fatalf("%v != %v", expected, target)
	}
}

// ErrorIs asserts that the error chain contains the target error.
// This is a wrapper for errors.Is.
func ErrorIs(t *testing.T, err, target error) {
	t.Helper()
	if !errors.Is(err, target) {
		t.Fatalf("[%v] is not [%v]", err, target)
	}
}

// ErrorIsNot asserts that the error chain does NOT contain the target error.
func ErrorIsNot(t *testing.T, err, target error) {
	t.Helper()
	if errors.Is(err, target) {
		t.Fatalf("[%v] is [%v]", err, target)
	}
}

// NotNil asserts that value is not nil.
func NotNil[T any](t *testing.T, value T) {
	t.Helper()
	if any(value) == nil {
		t.Fatalf("value is nil")
	}
}

// Len asserts that the specified object has specific length.
// Len also fails if the object has a type that len() not accept.
//
//	require.Len(t, mySlice, 3)
func Len[T Integer](t *testing.T, object any, length T) {
	t.Helper()

	l, ok := getLen[T](t, object)
	if !ok {
		t.Fatalf("[%v] could not be applied builtin len()", object)
	}

	if l != length {
		t.Fatalf("[%v] should have %d item(s), but has %d", object, length, l)
	}
}

// getLen tries to get the length of an object.
// It returns (0, false) if impossible.
func getLen[T Integer](t *testing.T, x any) (length T, ok bool) {
	t.Helper()
	v := reflect.ValueOf(x)
	defer func() {
		ok = recover() == nil
	}()
	return T(v.Len()), true
}
