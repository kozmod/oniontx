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
	Tracks []StepData
	Status StageStatus
}

func (r Result) String() string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("Status: %s\n", r.Status))
	builder.WriteString(fmt.Sprintf("Tracks(%d):\n", len(r.Tracks)))

	for i, track := range r.Tracks {
		builder.WriteString(fmt.Sprintf("  [%d] %s\n", i+1, track.String()))
	}

	return builder.String()
}

func prepareResult(tracks []*executionTrack) (Result, error) {
	const (
		comma = ", "
	)
	var (
		result = Result{
			Tracks: make([]StepData, 0, len(tracks)),
			Status: StageResultUnknown,
		}
		failed      = make([]string, 0, len(tracks))
		compensated = make([]string, 0, len(tracks))

		stateStrFn = func(position uint32, name string) string {
			return fmt.Sprintf("%d#%s", position, name)
		}

		resultErrorFn = func(err error) error {
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
			(data.Compensation.Status == ExecutionStatusFail ||
				data.Compensation.Status == ExecutionStatusUncalled):
			failed = append(failed,
				stateStrFn(data.StepPosition, data.StepName),
			)
		case data.Action.Status == ExecutionStatusFail &&
			data.Compensation.Status == ExecutionStatusSuccess:
			failed = append(failed,
				stateStrFn(data.StepPosition, data.StepName),
			)
			compensated = append(compensated,
				stateStrFn(data.StepPosition, data.StepName),
			)
		}

		result.Tracks = append(result.Tracks, data)
	}

	switch {
	case len(failed) > 0 && len(compensated) == 0:
		result.Status = StageResultFail
		return result, resultErrorFn(errors.Join(ErrActionFailed, ErrCompensationFailed))
		//return result, fmt.Errorf(
		//	"state failed - failed [%s], compensated [%s]: %w",
		//	strings.Join(failed, comma),
		//	strings.Join(compensated, comma),
		//	errors.Join(ErrActionFailed, ErrCompensationFailed),
		//)
	case len(compensated) > 0:
		result.Status = StageResultCompensated
		return result, resultErrorFn(ErrActionFailed)
		//return result, fmt.Errorf(
		//	"state failed - failed [%s], compensated [%s]: %w",
		//	strings.Join(failed, comma),
		//	strings.Join(compensated, comma),
		//	ErrActionFailed,
		//)
	}
	result.Status = StageResultSuccess
	return result, nil
}
