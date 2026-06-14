package saga

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/kozmod/oniontx/saga"
	"github.com/kozmod/oniontx/test/integration/internal/entity"
	"github.com/stretchr/testify/assert"
)

func Test_Concurrent(t *testing.T) {
	var (
		ctx = context.Background()
	)
	t.Run("async_compensation_v1", func(t *testing.T) {
		var (
			// Mutex for tests.
			mx                   sync.Mutex
			executedActions      []string
			executedCompensation []string

			// WaitGroup amd errors channel for simple async logic.
			wg      sync.WaitGroup
			errChan = make(chan error, 1)
		)

		steps := []saga.Step{
			saga.NewStep("step0").
				WithAction(saga.NewOperation(func(ctx context.Context, _ saga.Track) error {
					wg.Go(func() {
						mx.Lock()
						defer mx.Unlock()
						executedActions = append(executedActions, "action0")
					})
					return nil
				})).
				WithCompensation(saga.NewOperation(func(ctx context.Context, _ saga.Track) error {
					executedCompensation = append(executedCompensation, "comp0")
					return nil
				})),
			saga.NewStep("step1").
				WithAction(saga.NewOperation(func(ctx context.Context, _ saga.Track) error {
					wg.Go(func() {
						mx.Lock()
						defer mx.Unlock()
						executedActions = append(executedActions, "action1")
					})
					return nil
				})).
				WithCompensation(saga.NewOperation(func(ctx context.Context, _ saga.Track) error {
					executedCompensation = append(executedCompensation, "comp1")
					return nil
				})),
			saga.NewStep("step2").
				WithAction(saga.NewOperation(func(ctx context.Context, _ saga.Track) error {
					wg.Go(func() {
						mx.Lock()
						defer mx.Unlock()
						executedActions = append(executedActions, "action2")
						errChan <- entity.ErrExpected
					})
					return nil
				})).
				WithCompensation(saga.NewOperation(func(ctx context.Context, _ saga.Track) error {
					executedCompensation = append(executedCompensation, "comp2")
					return nil
				})),
			saga.NewStep("check_async_sttep").
				WithAction(saga.NewOperation(func(ctx context.Context, _ saga.Track) error {
					wg.Wait()
					close(errChan)

					var err error
					for e := range errChan {
						err = errors.Join(e, err)
					}
					assert.ErrorIs(t, err, entity.ErrExpected)
					return err
				})),
		}

		res, err := saga.NewSaga(steps).Execute(ctx)
		assert.Error(t, err)
		assert.Equal(t, saga.StageResultCompensated, res.Status)
		assert.ElementsMatch(t, []string{"action0", "action1", "action2"}, executedActions)
		assert.ElementsMatch(t, []string{"comp0", "comp1", "comp2"}, executedCompensation)
	})
}
