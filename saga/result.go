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
	// StageResultCompensated indicates at least one action failed and configured
	// compensations completed successfully.
	StageResultCompensated StageStatus = "Compensated"
)

// Result contains the complete execution report of a saga.
type Result struct {
	Steps  []StepData
	Status StageStatus
	errs   []error

	// errorsInTrackDataString controls whether String includes collected TrackData errors.
	errorsInTrackDataString bool
}

// Errors returns a copy of all errors recorded while executing actions and compensations.
func (r Result) Errors() []error {
	return slices.Clone(r.errs)
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
	_, _ = fmt.Fprintf(&builder, "Steps(%d):\n", len(r.Steps))

	for i, track := range r.Steps {
		_, _ = fmt.Fprintf(&builder, "  [%d] %s\n", i+1, track.string(r.errorsInTrackDataString))
	}

	return builder.String()
}

// prepareResult analyzes execution tracks and produces a final Result.
func prepareResult(tracks []*simpleTracker) (Result, error) {
	var (
		result = Result{
			Steps:  make([]StepData, 0, len(tracks)),
			Status: StageResultUnknown,
		}
		failed                             int32
		compensated                        int32
		failedCompensations                int32
		compensationNotRequired            int32
		failedActionsRequiringCompensation int32
		requiredCompensationNotSucceeded   int32
		executionErrors                    = make([]error, 0, len(tracks))

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
		}

		switch data.Action.Status {
		case ExecutionStatusFail:
			failed++

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
			case ExecutionStatusUnset:
				if !data.CompensationOnActionFailure {
					compensationNotRequired++
				}
			}
		}
	}

	result.errs = executionErrors

	switch {
	case failed == 0:
		result.Status = StageResultSuccess
		return result, nil

	case failedCompensations > 0 || requiredCompensationNotSucceeded > 0:
		result.Status = StageResultFail
		return result, resultErrorFn(errors.Join(ErrActionFailed, ErrCompensationFailed))

	case compensated == 0:
		// No compensation succeeded, so the failed workflow was not compensated.
		result.Status = StageResultFail
		return result, resultErrorFn(ErrActionFailed)

	default:
		result.Status = StageResultCompensated
		return result, resultErrorFn(ErrActionFailed)
	}
}
