package stdlib

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kozmod/oniontx/test/integration/internal/entity"
)

const (
	textRecord = "text_A"
)

func Test_UseCase(t *testing.T) {
	var (
		db        = ConnectDB(t)
		cleanupFn = func() {
			err := ClearDB(db)
			require.NoError(t, err)
		}
	)
	defer func() {
		err := db.Close()
		require.NoError(t, err)
	}()

	cleanupFn()

	t.Run("success_create", func(t *testing.T) {
		t.Cleanup(cleanupFn)

		var (
			ctx         = context.Background()
			transactor  = NewTransactor(db)
			repositoryA = NewTextRepository(transactor, false)
			repositoryB = NewTextRepository(transactor, false)
			useCase     = NewUseCase(repositoryA, repositoryB, transactor)
		)

		err := useCase.CreateTextRecords(ctx, textRecord)
		require.NoError(t, err)

		{
			records, err := GetTextRecords(db)
			require.NoError(t, err)
			require.Len(t, records, 2)
			for _, record := range records {
				require.Equal(t, textRecord, record)
			}
		}
	})
	t.Run("error_and_rollback", func(t *testing.T) {
		t.Cleanup(cleanupFn)

		var (
			ctx         = context.Background()
			transactor  = NewTransactor(db)
			repositoryA = NewTextRepository(transactor, false)
			repositoryB = NewTextRepository(transactor, true)
			useCase     = NewUseCase(repositoryA, repositoryB, transactor)
		)

		err := useCase.CreateTextRecords(ctx, textRecord)
		require.Error(t, err)
		require.ErrorIs(t, err, entity.ErrExpected)

		{
			records, err := GetTextRecords(db)
			require.NoError(t, err)
			require.Len(t, records, 0)
		}
	})
	t.Run("ctx_canceled_error_and_rollback", func(t *testing.T) {
		t.Cleanup(cleanupFn)

		var (
			ctx, cancel = context.WithCancel(context.Background())
			transactor  = NewTransactor(db)
			repositoryA = NewTextRepository(transactor, false)
			repositoryB = NewTextRepository(transactor, false)
			useCase     = NewUseCase(repositoryA, repositoryB, transactor)
		)

		cancel()
		err := useCase.CreateTextRecords(ctx, textRecord)
		require.Error(t, err)
		require.ErrorIs(t, err, context.Canceled)

		{
			records, err := GetTextRecords(db)
			require.NoError(t, err)
			require.Len(t, records, 0)
		}
	})
}

func Test_UseCasesFacade(t *testing.T) {
	var (
		db        = ConnectDB(t)
		cleanupFn = func() {
			err := ClearDB(db)
			require.NoError(t, err)
		}
	)
	defer func() {
		err := db.Close()
		require.NoError(t, err)
	}()

	cleanupFn()

	t.Run("single_repository", func(t *testing.T) {
		t.Run("success_create", func(t *testing.T) {
			t.Cleanup(cleanupFn)

			var (
				ctx            = context.Background()
				transactor     = NewTransactor(db)
				repositoryA    = NewTextRepository(transactor, false)
				repositoryB    = NewTextRepository(transactor, false)
				useCasesFacade = NewUseCasesFacade(
					NewUseCase(repositoryA, repositoryB, transactor),
					NewUseCase(repositoryA, repositoryB, transactor),
					transactor,
				)
			)

			err := useCasesFacade.CreateTextRecords(ctx, textRecord)
			require.NoError(t, err)

			{
				records, err := GetTextRecords(db)
				require.NoError(t, err)
				require.Len(t, records, 4)
				for _, record := range records {
					require.Equal(t, textRecord, record)
				}
			}
		})
		t.Run("error_and_rollback", func(t *testing.T) {
			t.Cleanup(cleanupFn)

			var (
				ctx            = context.Background()
				transactor     = NewTransactor(db)
				repositoryA    = NewTextRepository(transactor, false)
				repositoryB    = NewTextRepository(transactor, true)
				useCasesFacade = NewUseCasesFacade(
					NewUseCase(repositoryA, repositoryB, transactor),
					NewUseCase(repositoryA, repositoryB, transactor),
					transactor,
				)
			)

			err := useCasesFacade.CreateTextRecords(ctx, textRecord)
			require.Error(t, err)
			require.ErrorIs(t, err, entity.ErrExpected)

			{
				records, err := GetTextRecords(db)
				require.NoError(t, err)
				require.Len(t, records, 0)
			}
		})
	})
}
