//go:generate mockgen -source=use_case.go -destination=mocks.go -package=gomock

package gomock

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

const (
	textValue = "some_text"
)

func Test_GoMock(t *testing.T) {
	t.Run("assert_success", func(t *testing.T) {
		var (
			ctx  = context.Background()
			ctrl = gomock.NewController(t)
		)
		t.Cleanup(ctrl.Finish)

		var (
			transactorMock  = NewMocktransactor(ctrl)
			repositoryMockA = NewMockrepository(ctrl)
			repositoryMockB = NewMockrepository(ctrl)
		)
		gomock.InOrder(
			transactorMock.EXPECT().
				WithinTx(
					ctx,
					gomock.Any(),
				).
				DoAndReturn(func(ctx context.Context, cb func(context.Context) error) error {
					err := cb(ctx)
					assert.NoError(t, err)
					return nil
				}).
				Times(1),
			repositoryMockA.
				EXPECT().
				Insert(ctx, textValue).
				Return(nil).
				Times(1),
			repositoryMockB.
				EXPECT().
				Insert(ctx, textValue).
				Return(nil).
				Times(1),
		)

		useCase := UseCase{
			transactor: transactorMock,
			textRepoA:  repositoryMockA,
			textRepoB:  repositoryMockB,
		}

		err := useCase.CreateTextRecords(ctx, textValue)
		assert.NoError(t, err)
	})
	t.Run("assert_error", func(t *testing.T) {
		var (
			ctx           = context.Background()
			ctrl          = gomock.NewController(t)
			expError      = fmt.Errorf("some_error")
			transactorErr = fmt.Errorf("transactor_error")
		)
		t.Cleanup(ctrl.Finish)

		var (
			transactorMock  = NewMocktransactor(ctrl)
			repositoryMockA = NewMockrepository(ctrl)
			repositoryMockB = NewMockrepository(ctrl)
		)
		gomock.InOrder(
			transactorMock.EXPECT().
				WithinTx(
					ctx,
					gomock.Any(),
				).
				DoAndReturn(func(ctx context.Context, cb func(context.Context) error) error {
					err := cb(ctx)
					assert.Error(t, err)
					return transactorErr
				}),
			repositoryMockA.
				EXPECT().
				Insert(ctx, textValue).
				Return(nil),

			repositoryMockB.
				EXPECT().
				Insert(ctx, textValue).
				Return(expError),
		)

		useCase := UseCase{
			transactor: transactorMock,
			textRepoA:  repositoryMockA,
			textRepoB:  repositoryMockB,
		}

		err := useCase.CreateTextRecords(ctx, textValue)
		assert.Error(t, err)
		assert.ErrorIs(t, err, transactorErr)
	})
}
