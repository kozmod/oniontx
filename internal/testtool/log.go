package testtool

import "testing"

func LogError(t testing.TB, err error) {
	t.Helper()
	t.Logf("test error output: \n{\n%v\n}", err)
}
