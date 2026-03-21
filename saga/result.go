package saga

import (
	"errors"
	"fmt"
	"strings"
)

type StageStatus string

const (
	StageResultUnknown     StageStatus = "Unknown"
	StageResultFail        StageStatus = "Fail"
	StageResultSuccess     StageStatus = "Success"
	StageResultCompensated StageStatus = "Compensated"
)

type Result struct {
	Sterps []StepData
	Status StageStatus
}

func (r Result) String() string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("Status: %s\n", r.Status))
	builder.WriteString(fmt.Sprintf("Sterps(%d):\n", len(r.Sterps)))

	for i, track := range r.Sterps {
		builder.WriteString(fmt.Sprintf("  [%d] %s\n", i+1, track.String()))
	}

	return builder.String()
}

func prepareResult(tracks []*executionTrack) (Result, error) {
	var (
		result = Result{
			Sterps: make([]StepData, 0, len(tracks)),
			Status: StageResultUnknown,
		}
		failed      = make([]string, 0, len(tracks))
		compensated = make([]string, 0, len(tracks))

		prepareStateStrFn = func(position uint32, name string) string {
			return fmt.Sprintf("%d#%s", position, name)
		}

		resultErrorFn = func(err error) error {
			const (
				comma = ", "
			)
			return fmt.Errorf(
				"state failed - failed [%s], compensated [%s]: %w",
				strings.Join(failed, comma),
				strings.Join(compensated, comma),
				err,
			)
		}
	)

	for _, tr := range tracks {
		data := tr.GetData()

		switch {
		case data.Action.Status == ExecutionStatusFail &&
			data.Compensation.Status == ExecutionStatusUncalled:
			failed = append(failed,
				prepareStateStrFn(data.StepPosition, data.StepName),
			)
		case data.Action.Status == ExecutionStatusFail &&
			data.Compensation.Status == ExecutionStatusFail:
			failed = append(failed,
				prepareStateStrFn(data.StepPosition, data.StepName),
			)
		case data.Action.Status == ExecutionStatusFail &&
			data.Compensation.Status == ExecutionStatusSuccess:
			compensated = append(compensated,
				prepareStateStrFn(data.StepPosition, data.StepName),
			)
		case data.Action.Status == ExecutionStatusSuccess &&
			data.Compensation.Status == ExecutionStatusSuccess:
			compensated = append(compensated,
				prepareStateStrFn(data.StepPosition, data.StepName),
			)
		}

		result.Sterps = append(result.Sterps, data)
	}

	switch {
	case len(failed) > 0 && len(failed) != len(compensated):
		result.Status = StageResultFail
		return result, resultErrorFn(errors.Join(ErrActionFailed, ErrCompensationFailed))
	case len(compensated) > 0:
		result.Status = StageResultCompensated
		return result, resultErrorFn(ErrActionFailed)
	default:
		result.Status = StageResultSuccess
	}
	return result, nil
}
