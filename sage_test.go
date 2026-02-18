package oniontx

import (
	"context"
	"slices"
	"testing"
)

// nolint: dupl
func TestSagaTransactor_Execute_Success(t *testing.T) {
	const (
		txKey = "TX_KEY"
	)

	var (
		ctx             = context.Background()
		contextOperator = NewContextOperator[string, *committerMock](txKey)
	)

	var (
		beginnerCalls   []string
		committerCalls  []string
		executedActions []string
	)

	steps := []Step{
		{
			Name: "step1",
			Transactor: NewTransactor[*beginnerMock[*committerMock], *committerMock](&beginnerMock[*committerMock]{
				beginFn: func(ctx context.Context) (*committerMock, error) {
					beginnerCalls = append(beginnerCalls, "begin1")
					return &committerMock{
						commitFn: func(ctx context.Context) error {
							committerCalls = append(committerCalls, "commit1")
							return nil
						},
					}, nil
				},
			}, contextOperator),
			Action: func(ctx context.Context) error {
				executedActions = append(executedActions, "action1")
				return nil
			},
			Compensation: func(ctx context.Context) error {
				executedActions = append(executedActions, "comp1")
				return nil
			},
		},
		{
			Name: "step2",
			Transactor: NewTransactor[*beginnerMock[*committerMock], *committerMock](&beginnerMock[*committerMock]{
				beginFn: func(ctx context.Context) (*committerMock, error) {
					beginnerCalls = append(beginnerCalls, "begin2")
					return &committerMock{
						commitFn: func(ctx context.Context) error {
							committerCalls = append(committerCalls, "commit2")
							return nil
						},
					}, nil
				},
			}, contextOperator),
			Action: func(ctx context.Context) error {
				executedActions = append(executedActions, "action2")
				return nil
			},
			Compensation: func(ctx context.Context) error {
				executedActions = append(executedActions, "comp2")
				return nil
			},
		},
	}

	err := NewSaga(steps).Execute(ctx)
	assertNoError(t, err)
	assertTrue(t, slices.Equal(beginnerCalls, []string{"begin1", "begin2"}))
	assertTrue(t, slices.Equal(committerCalls, []string{"commit1", "commit2"}))
	assertTrue(t, slices.Equal(executedActions, []string{"action1", "action2"}))
}
