package saga

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/kozmod/oniontx"
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

		steps := []oniontx.Step{
			{
				Name: "step0",
				Action: func(ctx context.Context) error {
					wg.Go(func() {
						mx.Lock()
						defer mx.Unlock()
						executedActions = append(executedActions, "action0")
					})
					return nil
				},
				Compensation: func(ctx context.Context, aroseErr error) error {
					executedCompensation = append(executedCompensation, "comp0")
					return nil
				},
			},
			{
				Name: "step1",
				Action: func(ctx context.Context) error {
					wg.Go(func() {
						mx.Lock()
						defer mx.Unlock()
						executedActions = append(executedActions, "action1")
					})
					return nil
				},
				Compensation: func(ctx context.Context, aroseErr error) error {
					executedCompensation = append(executedCompensation, "comp1")
					return nil
				},
			},
			{
				Name: "step2",
				Action: func(ctx context.Context) error {
					wg.Go(func() {
						mx.Lock()
						defer mx.Unlock()
						executedActions = append(executedActions, "action2")
						errChan <- entity.ErrExpected
					})
					return nil
				},
				Compensation: func(ctx context.Context, aroseErr error) error {
					executedCompensation = append(executedCompensation, "comp2")
					return nil
				},
			},
			{
				Name: "check_async_sttep",
				Action: func(ctx context.Context) error {
					wg.Wait()
					close(errChan)

					var err error
					for e := range errChan {
						err = errors.Join(err, e)
					}
					assert.ErrorIs(t, err, entity.ErrExpected)
					return err
				},
			},
		}

		err := oniontx.NewSaga(steps).Execute(ctx)
		assert.Error(t, err)
		assert.ErrorIs(t, err, entity.ErrExpected)
		assert.ElementsMatch(t, []string{"action0", "action1", "action2"}, executedActions)
		assert.ElementsMatch(t, []string{"comp0", "comp1", "comp2"}, executedCompensation)
	})
}
