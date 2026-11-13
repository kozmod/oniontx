package saga

import (
	"fmt"
	"slices"
	"strings"

	"github.com/kozmod/oniontx/internal/errors"
)

// StageStatus represents the overall outcome of a saga execution.
type StageStatus string

const (
	// StageResultUnknown indicates the saga result cannot be determined.
	StageResultUnknown StageStatus = "Unknown"
	// StageResultFail indicates the saga failed and compensation did not fully recover it.
	StageResultFail StageStatus = "Fail"
	// StageResultSuccess indicates all actions completed successfully.
	StageResultSuccess StageStatus = "Success"
	// StageResultCompensated indicates execution failed or was interrupted, and
	// at least one compensation succeeded with no required compensation incomplete.
	StageResultCompensated StageStatus = "Compensated"
)

// Result contains the complete execution report of a saga.
type Result struct {
	Steps  []StepData
	Status StageStatus
	errs   []error

	// executionErr is an execution-level error.
	executionErr error

	// errorsInTrackDataString controls whether String includes collected TrackData errors.
	errorsInTrackDataString bool
}

// Errors returns a copy of all operation errors and errors that interrupted saga execution.
func (r Result) Errors() []error {
	return slices.Clone(r.errs)
}

// ExecutionError returns an execution-level  error.
func (r Result) ExecutionError() error {
	return r.executionErr
}

// WithErrorsInTrackDataString returns a copy of Result whose String output
// includes the collected errors for every TrackData value.
func (r Result) WithErrorsInTrackDataString() Result {
	r.errorsInTrackDataString = true
	return r
}

// String returns a human-readable representation of the Result.
func (r Result) String() string {
	var builder strings.Builder

	_, _ = fmt.Fprintf(&builder, "Status: %s\n", r.Status)
	if r.executionErr != nil {
		_, _ = fmt.Fprintf(&builder, "Execution error: %v\n", r.executionErr)
	}
	_, _ = fmt.Fprintf(&builder, "Steps(%d):\n", len(r.Steps))

	for i, track := range r.Steps {
		_, _ = fmt.Fprintf(&builder, "  [%d] %s\n", i+1, track.string(r.errorsInTrackDataString))
	}

	return builder.String()
}

// prepareResult analyzes execution tracks and produces a final Result.
func prepareResult(track sagaTrack) (Result, error) {
	var (
		tracks = track.tracks
		result = Result{
			Steps:        make([]StepData, 0, len(tracks)),
			Status:       StageResultUnknown,
			executionErr: track.err,
		}
		failed                             int
		compensated                        int
		failedCompensations                int
		compensationNotRequired            int
		failedActionsRequiringCompensation int
		requiredCompensationNotSucceeded   int
		executionErrors                    = make([]error, 0, len(tracks))
		failureCauses                      []error

		resultErrorFn = func(err error) error {
			return fmt.Errorf(
				"state failed - failed [%d], compensated [%d], failed compensations [%d], compensation not required [%d], failed actions requiring compensation [%d], required compensations not succeeded [%d]: %w",
				failed,
				compensated,
				failedCompensations,
				compensationNotRequired,
				failedActionsRequiringCompensation,
				requiredCompensationNotSucceeded,
				err,
			)
		}
	)

	for _, tr := range tracks {
		data := tr.GetStepData()
		result.Steps = append(result.Steps, data)
		executionErrors = append(executionErrors, data.Action.Errors...)
		executionErrors = append(executionErrors, data.Compensation.Errors...)

		if data.Compensation.Status == ExecutionStatusFail {
			failedCompensations++
			if n := len(data.Compensation.Errors); n > 0 {
				failureCauses = append(failureCauses, data.Compensation.Errors[n-1])
			}
		}

		switch data.Action.Status {
		case ExecutionStatusFail:
			failed++
			if n := len(data.Action.Errors); n > 0 {
				failureCauses = append(failureCauses, data.Action.Errors[n-1])
			}

			if data.CompensationOnActionFailure {
				failedActionsRequiringCompensation++
				if data.Compensation.Status == ExecutionStatusSuccess {
					compensated++
				} else {
					requiredCompensationNotSucceeded++
				}
			}

		case ExecutionStatusSuccess:
			switch data.Compensation.Status {
			case ExecutionStatusSuccess:
				compensated++
			case ExecutionStatusUncalled:
				if track.compensating {
					requiredCompensationNotSucceeded++
				}
			case ExecutionStatusUnset:
				compensationNotRequired++
			}
		}
	}

	if track.err != nil {
		executionErrors = append(executionErrors, track.err)
	}
	result.errs = executionErrors

	var actionErr error
	if failed > 0 {
		actionErr = ErrActionFailed
	}
	failureErr := errors.Join(actionErr, track.err, errors.Join(failureCauses...))

	switch {
	case failed == 0 && failedCompensations == 0 && requiredCompensationNotSucceeded == 0 && track.err == nil:
		result.Status = StageResultSuccess
		return result, nil

	case failedCompensations > 0 || requiredCompensationNotSucceeded > 0:
		result.Status = StageResultFail
		return result, resultErrorFn(errors.Join(failureErr, ErrCompensationFailed))

	case compensated == 0:
		// No compensation succeeded, so the failed workflow was not compensated.
		result.Status = StageResultFail
		return result, resultErrorFn(failureErr)

	default:
		result.Status = StageResultCompensated
		return result, resultErrorFn(failureErr)
	}
}
