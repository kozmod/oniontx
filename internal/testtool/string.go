package testtool

import (
	"fmt"
	"strings"
)

func JoinAsString[T any](sep string, vals ...T) string {
	strVals := make([]string, len(vals))
	for i, val := range vals {
		strVals[i] = fmt.Sprintf("%v", val)
	}
	return strings.Join(strVals, sep)
}
