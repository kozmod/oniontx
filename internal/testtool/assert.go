package testtool

import "testing"

// AssertTrue was added to avoid to use external dependencies for mocking
func AssertTrue(t *testing.T, val bool) {
	t.Helper()
	if !val {
		t.Fatalf("expected true [current value: %v]", val)
	}
}

// AssertFalse was added to avoid to use external dependencies for mocking
func AssertFalse(t *testing.T, val bool) {
	t.Helper()
	if val {
		t.Fatalf("expected false [current value: %v]", val)
	}
}

// AssertNoError was added to avoid to use external dependencies for mocking
func AssertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("error arose: %v", err)
	}
}

// AssertError was added to avoid to use external dependencies for mocking
func AssertError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("error expected")
	}
}
