package saga

import (
	"errors"
	"fmt"
	"strings"
)

// StageStatus represents the overall outcome of a saga execution.
type StageStatus string

const (
	// StageResultUnknown indicates the saga result cannot be determined.
	StageResultUnknown StageStatus = "Unknown"
	// StageResultFail indicates the saga failed and no compensation was applied
	// (or compensation also failed).
	StageResultFail StageStatus = "Fail"
	// StageResultSuccess indicates all actions completed successfully
	StageResultSuccess StageStatus = "Success"
	// StageResultCompensated indicates some actions failed and successful
	// compensations were applied.
	StageResultCompensated StageStatus = "Compensated"
)

// Result contains the complete execution report of a saga.
type Result struct {
	Steps  []StepData
	Status StageStatus
}

// String returns a human-readable representation of the Result.
func (r Result) String() string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("Status: %s\n", r.Status))
	builder.WriteString(fmt.Sprintf("Steps(%d):\n", len(r.Steps)))

	for i, track := range r.Steps {
		builder.WriteString(fmt.Sprintf("  [%d] %s\n", i+1, track.String()))
	}

	return builder.String()
}

// prepareResult analyzes execution tracks and produces a final Result.
// It evaluates the state of all steps and determines the overall saga outcome
// based on action failures, compensation requirements, and compensation outcomes.
//
// The function implements the following logic:
//   - If no actions failed -> StageResultSuccess
//   - If any compensation that was required to run failed -> StageResultFail
//   - If there were failed actions requiring compensation but all compensations
//     succeeded -> StageResultCompensated
//   - Special case: when no compensations were required, no successful steps,
//     and no successful compensations -> StageResultFail
//
// Returns:
//   - Result: aggregated execution data for all steps
//   - error: descriptive error with categorized lists of failed/compensated steps
func prepareResult(tracks []*executionTrack) (Result, error) {
	var (
		result = Result{
			Steps:  make([]StepData, 0, len(tracks)),
			Status: StageResultUnknown,
		}
		failed                          = make([]string, 0, len(tracks))
		compensated                     = make([]string, 0, len(tracks))
		compensationNotRequired         = make([]string, 0, len(tracks))
		failedWithCompensationReq       = make([]string, 0, len(tracks))
		failedWithCompensationReqFailed = make([]string, 0, len(tracks))
		hasSuccessfulStep               = false

		prepareStateStrFn = func(position uint32, name string) string {
			return fmt.Sprintf("%d#%s", position, name)
		}

		resultErrorFn = func(err error) error {
			const comma = ", "
			return fmt.Errorf(
				"state failed - failed [%s], compensated [%s], compensation not required [%s], failed requiring compensation [%s]: %w",
				strings.Join(failed, comma),
				strings.Join(compensated, comma),
				strings.Join(compensationNotRequired, comma),
				strings.Join(failedWithCompensationReq, comma),
				err,
			)
		}
	)

	for _, tr := range tracks {
		data := tr.GetData()
		result.Steps = append(result.Steps, data)

		stepID := prepareStateStrFn(data.StepPosition, data.StepName)

		if data.Action.Status == ExecutionStatusSuccess {
			hasSuccessfulStep = true
		}

		if data.Action.Status == ExecutionStatusFail {
			failed = append(failed, stepID)

			if data.CompensationRequired {
				failedWithCompensationReq = append(failedWithCompensationReq, stepID)
				if data.Compensation.Status == ExecutionStatusSuccess {
					compensated = append(compensated, stepID)
				} else {
					failedWithCompensationReqFailed = append(failedWithCompensationReqFailed, stepID)
				}
			}
			continue
		}

		if data.Action.Status == ExecutionStatusSuccess {
			switch data.Compensation.Status {
			case ExecutionStatusSuccess:
				compensated = append(compensated, stepID)
			case ExecutionStatusUnset:
				if !data.CompensationRequired {
					compensationNotRequired = append(compensationNotRequired, stepID)
				}
			}
		}
	}

	switch {
	case len(failed) == 0:
		result.Status = StageResultSuccess
		return result, nil

	case len(failedWithCompensationReqFailed) > 0:
		result.Status = StageResultFail
		return result, resultErrorFn(errors.Join(ErrActionFailed, ErrCompensationFailed))

	case len(failedWithCompensationReq) == 0 && !hasSuccessfulStep && len(compensated) == 0:
		// Edge case: no required compensations, no successful steps, no successful compensations
		// This indicates a failure scenario where no meaningful recovery occurred
		result.Status = StageResultFail
		return result, resultErrorFn(errors.Join(ErrActionFailed, ErrCompensationFailed))

	default:
		result.Status = StageResultCompensated
		return result, resultErrorFn(ErrActionFailed)
	}
}
