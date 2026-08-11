package mtx

import "fmt"

var (
	// ErrNilTxBeginner indicates that the provided TxBeginner is nil.
	// This error is returned when trying to use a Transactor with an uninitialized beginner.
	ErrNilTxBeginner = fmt.Errorf("tx beginner is nil")

	// ErrNilTxOperator indicates that the provided CtxOperator is nil.
	// This error is returned when trying to use a Transactor with an uninitialized operator.
	ErrNilTxOperator = fmt.Errorf("tx operator is nil")

	// ErrBeginTx indicates that starting a new transaction has failed.
	// This error wraps the underlying error from the database driver.
	ErrBeginTx = fmt.Errorf("begin tx")

	// ErrCommitFailed indicates that committing a transaction has failed.
	// This error wraps the underlying error from the database driver during commit
	ErrCommitFailed = fmt.Errorf("commit failed")

	// ErrRollbackFailed indicates that rolling back a transaction has failed.
	ErrRollbackFailed = fmt.Errorf("transaction rollback failed")

	// ErrPanicRecovered indicates that a panic was recovered and converted to an error.
	// It wraps the original panic value to provide context about what caused the panic.
	ErrPanicRecovered = fmt.Errorf("panic recovered")
)

// RollbackError reports an operation failure followed by a failed rollback.
//
// It unwraps to ErrRollbackFailed, Cause, and RollbackErr so callers can use
// errors.Is and errors.As to inspect every part of the failure.
type RollbackError struct {
	Cause       error
	RollbackErr error
}

// NewRollbackError returns an error that joins the operation cause with the
// error returned while rolling back the transaction.
func NewRollbackError(cause, rollbackErr error) *RollbackError {
	return &RollbackError{
		Cause:       cause,
		RollbackErr: rollbackErr,
	}
}

func (e *RollbackError) Error() string {
	if e == nil {
		return "<nil>"
	}
	return fmt.Sprintf(
		"%s [cause='%v', rollback='%v']",
		ErrRollbackFailed,
		e.Cause,
		e.RollbackErr,
	)
}

func (e *RollbackError) Unwrap() []error {
	if e == nil {
		return nil
	}

	errs := make([]error, 0, 3)
	errs = append(errs, ErrRollbackFailed)
	if e.Cause != nil {
		errs = append(errs, e.Cause)
	}
	if e.RollbackErr != nil {
		errs = append(errs, e.RollbackErr)
	}

	return errs
}
