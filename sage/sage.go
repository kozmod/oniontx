package sage

import (
	"context"
	"log"
)

type StageFn func(ctx context.Context) error

type Step struct {
	apply    StageFn
	rollback StageFn
}

type Stage struct {
	stages []Step
}

func (s *Stage) Exec(ctx context.Context) {
	for i, stage := range s.stages {
		err := stage(ctx)
		if err != nil {
			log.Printf("error executing stage [%d]: %v", i, err)
		}
	}

}
