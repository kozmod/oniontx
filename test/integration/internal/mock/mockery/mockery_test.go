package mockery

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	textValue = "some_text"

	repositoryMethodInsert   = "Insert"
	transactorMethodWithinTx = "WithinTx"
)

func Test_mockery(t *testing.T) {
	t.Run("assert_success", func(t *testing.T) {
		ctx := context.Background()
		transactorMock := new(mockTransactor)
		transactorMock.On(transactorMethodWithinTx,
			ctx,
			mock.MatchedBy(func(i any) bool {
				fn, ok := i.(func(context.Context) error)
				assert.True(t, ok)
				return assert.NoError(t, fn(ctx))
			})).Return(nil)

		repositoryMockA := new(mockRepository)
		repositoryMockA.On(repositoryMethodInsert, ctx, textValue).Return(nil)

		repositoryMockB := new(mockRepository)
		repositoryMockB.On(repositoryMethodInsert, ctx, textValue).Return(nil)

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
			expError      = fmt.Errorf("some_error")
			trsnsactorErr = fmt.Errorf("transactor_error")
		)

		transactorMock := new(mockTransactor)
		transactorMock.On(transactorMethodWithinTx,
			ctx,
			mock.MatchedBy(func(i any) bool {
				fn, ok := i.(func(context.Context) error)
				assert.True(t, ok)
				return assert.Error(t, fn(ctx))
			}),
		).Return(trsnsactorErr)

		repositoryMockA := new(mockRepository)
		repositoryMockA.On(repositoryMethodInsert, ctx, textValue).Return(nil)

		repositoryMockB := new(mockRepository)
		repositoryMockB.On(repositoryMethodInsert, ctx, textValue).Return(expError)

		useCase := UseCase{
			transactor: transactorMock,
			textRepoA:  repositoryMockA,
			textRepoB:  repositoryMockB,
		}

		err := useCase.CreateTextRecords(ctx, textValue)
		assert.Error(t, err)
		assert.ErrorIs(t, err, trsnsactorErr)
	})
}
