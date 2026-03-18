package testtool

import (
	"os"
	"strings"
	"testing"
)

const (
	envTestLoggerDisabled = "TEST_FN_DISABLED"
)

var (
	disableTestLogger = false
)

func init() {
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
