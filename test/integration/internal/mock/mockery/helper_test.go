package mockery

import (
	"reflect"
	"runtime"
	"strings"
	"testing"
)

// FN returns name of a function.
func FN(t *testing.T, i any) string {
	t.Helper()
	const (
		fmSuffix = "-fm"
		empty    = ""
	)
	value := reflect.ValueOf(i)
	if value.Kind() != reflect.Func {
		t.FailNow()
	}
	fullName := runtime.FuncForPC(value.Pointer()).Name()
	fullNameSlice := strings.Split(fullName, ".")
	name := fullNameSlice[len(fullNameSlice)-1]
	name = strings.Replace(name, fmSuffix, empty, -1)
	return name
}
