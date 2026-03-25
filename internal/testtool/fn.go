package testtool

import (
	"os"
	"strings"
	"testing"
)

var (
	disableTestLogger = false
)

func init() {
	const (
		envTestLoggerDisabled = "TEST_FN_DISABLED"
	)
	dtl := os.Getenv(envTestLoggerDisabled)
	if strings.TrimSpace(strings.ToLower(dtl)) == "true" {
		disableTestLogger = true
	}
}

func TestFn(t *testing.T, fn func()) {
	t.Helper()
	if disableTestLogger {
		return
	}
	fn()
}
