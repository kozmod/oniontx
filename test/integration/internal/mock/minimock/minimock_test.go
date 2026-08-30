//go:generate minimock -pr mock_ -g
//go:generate sh ../scripts.sh update_mocks .
//go:generate git add .

package gomock

import (
	"context"
	"fmt"
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"
)

const (
	textValue = "some_text"
)

func Test_MiniMock(t *testing.T) {
	t.Run("expect", func(t *testing.T) {
		t.Run("assert_success", func(t *testing.T) {
			var (
				ctx = context.Background()
				mc  = minimock.NewController(t)
			)

			repositoryMockA := NewRepositoryMock(mc).
				InsertMock.
				Expect(ctx, textValue).
				Return(nil)
			repositoryMockB := NewRepositoryMock(mc).
				InsertMock.
				Expect(ctx, textValue).
				Return(nil)
			transactorMock := NewTransactorMock(mc).
				WithinTxMock.
				Inspect(func(ctx context.Context, fn func(ctx context.Context) error) {
					err := fn(ctx)
					require.NoError(t, err)
				}).Return(nil)

			useCase := UseCase{
				transactor: transactorMock,
				textRepoA:  repositoryMockA,
				textRepoB:  repositoryMockB,
			}

			err := useCase.CreateTextRecords(ctx, textValue)
			require.NoError(t, err)
		})
		t.Run("assert_error", func(t *testing.T) {
			var (
				ctx           = context.Background()
				mc            = minimock.NewController(t)
				expError      = fmt.Errorf("some_error")
				transactorErr = fmt.Errorf("transactor_error")
			)

			repositoryMockA := NewRepositoryMock(mc).
				InsertMock.
				Expect(ctx, textValue).
				Return(nil)
			repositoryMockB := NewRepositoryMock(mc).
				InsertMock.
				Expect(ctx, textValue).
				Return(expError)
			transactorMock := NewTransactorMock(mc).
				WithinTxMock.
				Inspect(func(ctx context.Context, fn func(ctx context.Context) error) {
					err := fn(ctx)
					require.Error(t, err)
					require.ErrorIsf(t, err, expError, "repositoryMockB does not return err [%v]", expError)
				}).Return(transactorErr)

			useCase := UseCase{
				transactor: transactorMock,
				textRepoA:  repositoryMockA,
				textRepoB:  repositoryMockB,
			}

			err := useCase.CreateTextRecords(ctx, textValue)
			require.Error(t, err)
			require.ErrorIs(t, err, transactorErr)
		})
	})
	t.Run("set", func(t *testing.T) {
		t.Run("assert_success", func(t *testing.T) {
			var (
				ctx = context.Background()
				mc  = minimock.NewController(t)
			)

			repositoryMockA := NewRepositoryMock(mc).
				InsertMock.
				Set(func(ctx context.Context, val string) (err error) {
					return nil
				})
			repositoryMockB := NewRepositoryMock(mc).
				InsertMock.
				Set(func(ctx context.Context, val string) (err error) {
					return nil
				})
			transactorMock := NewTransactorMock(mc).
				WithinTxMock.
				Set(func(ctx context.Context, fn func(ctx context.Context) error) (err error) {
					require.NoError(t, repositoryMockA.Insert(ctx, textValue))
					require.NoError(t, repositoryMockB.Insert(ctx, textValue))
					return nil
				})

			useCase := UseCase{
				transactor: transactorMock,
				textRepoA:  repositoryMockA,
				textRepoB:  repositoryMockB,
			}

			err := useCase.CreateTextRecords(ctx, textValue)
			require.NoError(t, err)
		})
		t.Run("assert_error", func(t *testing.T) {
			var (
				ctx           = context.Background()
				mc            = minimock.NewController(t)
				expError      = fmt.Errorf("some_error")
				transactorErr = fmt.Errorf("transactor_error")
			)

			repositoryMockA := NewRepositoryMock(mc).
				InsertMock.
				Set(func(ctx context.Context, val string) (err error) {
					return nil
				})
			repositoryMockB := NewRepositoryMock(mc).
				InsertMock.
				Set(func(ctx context.Context, val string) (err error) {
					return expError
				})
			transactorMock := NewTransactorMock(mc).
				WithinTxMock.
				Set(func(ctx context.Context, fn func(ctx context.Context) error) (err error) {
					require.NoError(t, repositoryMockA.Insert(ctx, textValue))
					require.Error(t, repositoryMockB.Insert(ctx, textValue))
					return transactorErr
				})

			useCase := UseCase{
				transactor: transactorMock,
				textRepoA:  repositoryMockA,
				textRepoB:  repositoryMockB,
			}

			err := useCase.CreateTextRecords(ctx, textValue)
			require.Error(t, err)
			require.ErrorIs(t, err, transactorErr)
		})
	})
}
