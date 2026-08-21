package saga

import (
	"fmt"
	"testing"
)

var (
	benchmarkPrepareResultResult Result
	benchmarkPrepareResultErr    error
)

func Benchmark_PrepareResult(b *testing.B) {
	scenarios := []struct {
		name               string
		compensated        bool
		compensationFailed bool
	}{
		{name: "success"},
		{name: "compensated", compensated: true},
		{name: "compensation_failed", compensated: true, compensationFailed: true},
	}

	for _, scenario := range scenarios {
		for _, steps := range []int{1, 10, 100, 1_000, 10_000} {
			tracks := prepareResultBenchmarkTracks(steps, scenario.compensated, scenario.compensationFailed)

			b.Run(fmt.Sprintf("%s/steps_%d", scenario.name, steps), func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					benchmarkPrepareResultResult, benchmarkPrepareResultErr = prepareResult(tracks)
				}
			})
		}
	}

	_ = benchmarkPrepareResultResult
	_ = benchmarkPrepareResultErr
}

func prepareResultBenchmarkTracks(steps int, compensated, compensationFailed bool) []*simpleTracker {
	tracks := make([]*simpleTracker, steps)
	for i := range tracks {
		step := NewStep(fmt.Sprintf("step-%d", i)).
			WithAction(
				NewOperation(dummyOperation),
			)
		if compensated {
			step = step.
				WithCompensation(
					NewOperation(dummyOperation),
				)
			if i == steps-1 {
				step = step.WithCompensationOnActionFailure()
			}
		}

		track := newInMemoryTrack(uint32(i), step)
		track.action.apply(
			newTrackSucceededAct(),
		)
		if compensated {
			track.compensation.apply(
				newTrackSucceededAct(),
			)
		}
		tracks[i] = track
	}

	if compensated {
		last := tracks[len(tracks)-1]
		last.action.apply(
			newTrackFailedAct(ErrActionFailed),
		)
		if compensationFailed {
			last.compensation.apply(
				newTrackFailedAct(ErrCompensationFailed),
			)
		}
	}

	return tracks
}
