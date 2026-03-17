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
	Tracks []StepTrack
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

func prepareResult(tracks []*track) (Result, error) {
	var (
		result = Result{
			Tracks: make([]StepTrack, 0, len(tracks)),
			Status: StageResultUnknown,
		}
		errs []error
	)

	for _, t := range tracks {
		tr := t.GetTrack()

		switch {
		case tr.Action.Status == ExecutionStatusFail &&
			(tr.Compensation.Status == ExecutionStatusFail || tr.Compensation.Status == ExecutionStatusUnknown):
			result.Status = StageResultFail
			errs = append(errs,
				fmt.Errorf("action failed [%d#%s]: %w", tr.StepPosition, tr.StepName,
					errors.Join(ErrActionFailed, ErrCompensationFailed),
				),
			)

		case tr.Action.Status == ExecutionStatusFail &&
			tr.Compensation.Status == ExecutionStatusSuccess:
			errs = append(errs,
				fmt.Errorf("action failed and compensated [%d#%s]: %w", tr.StepPosition, tr.StepName, ErrActionFailed),
			)
			if result.Status != StageResultFail {
				result.Status = StageResultCompensated
			}
		}

		result.Tracks = append(result.Tracks, tr)
	}

	if len(errs) <= 0 {
		result.Status = StageResultSuccess
		return result, nil
	}

	return result, fmt.Errorf("stage failed: %w", errors.Join(errs...))
}
