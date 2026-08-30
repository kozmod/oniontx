package testtool

import (
	"os"
	"strings"
	"sync"
	"testing"
)

var disableTestFn = sync.OnceValue[bool](func() bool {
	const (
		envTestFnDisabled = "TEST_FN_DISABLED"
	)
	var (
		dtl = os.Getenv(envTestFnDisabled)
		val = strings.TrimSpace(strings.ToLower(dtl))
	)
	return val == "true"
})

func TestFn(t *testing.T, fn func()) {
	t.Helper()
	if disableTestFn() {
		return
	}
	fn()
}
