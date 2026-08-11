package mtx

import (
	"fmt"
	"testing"

	"github.com/kozmod/oniontx/internal/testtool/assert"
)

func Test_RollbackError(t *testing.T) {
	var (
		cause       = fmt.Errorf("operation failed")
		rollbackErr = fmt.Errorf("rollback failed")
		err         = NewRollbackError(cause, rollbackErr)
	)

	assert.Equal(
		t,
		"transaction rollback failed [cause='operation failed', rollback='rollback failed']",
		err.Error(),
	)

	unwrapped := err.Unwrap()
	assert.Equal(t, 3, len(unwrapped))
	assert.Equal(t, ErrRollbackFailed, unwrapped[0])
	assert.Equal(t, cause, unwrapped[1])
	assert.Equal(t, rollbackErr, unwrapped[2])
}
