package gorm

import (
	"context"
	"database/sql"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pkg/errors"

	"github.com/stretchr/testify/require"

	"github.com/kozmod/oniontx/test/integration/internal/entity"
)

const (
	textRecord = "text_A"
)

func Test_UseCase_CreateTextRecords(t *testing.T) {
	var (
		db        = ConnectDB(t)
		cleanupFn = func() {
			err := ClearDB(db)
			require.NoError(t, err)
		}
	)

	cleanupFn()

	t.Run("success_create", func(t *testing.T) {
		t.Cleanup(cleanupFn)

		var (
			ctx        = context.Background()
			transactor = NewTransactor(
				NewDB(db, sql.TxOptions{}),
			)
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
				require.Equal(t, Text{Val: textRecord}, record)
			}
		}
	})
	t.Run("error_and_rollback", func(t *testing.T) {
		t.Cleanup(cleanupFn)

		var (
			ctx        = context.Background()
			transactor = NewTransactor(
				NewDB(db, sql.TxOptions{}),
			)
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
	t.Run("error_and_rollback_when_ReadOnly_is_true", func(t *testing.T) {
		t.Cleanup(cleanupFn)

		var (
			ctx        = context.Background()
			transactor = NewTransactor(
				NewDB(db, sql.TxOptions{ReadOnly: true}),
			)
			repositoryA = NewTextRepository(transactor, false)
			repositoryB = NewTextRepository(transactor, false)
			useCase     = NewUseCase(repositoryA, repositoryB, transactor)
		)

		err := useCase.CreateTextRecords(ctx, textRecord)
		require.Error(t, err)

		var pgErr *pgconn.PgError
		require.True(t, errors.As(err, &pgErr))
		require.Equal(t, `25006`, pgErr.Code)
	})
	t.Run("ctx_canceled_error_and_rollback", func(t *testing.T) {
		t.Cleanup(cleanupFn)

		var (
			ctx, cancel = context.WithCancel(context.Background())
			transactor  = NewTransactor(
				NewDB(db, sql.TxOptions{}),
			)
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

func Test_UseCase_CreateText(t *testing.T) {
	var (
		db        = ConnectDB(t)
		cleanupFn = func() {
			err := ClearDB(db)
			require.NoError(t, err)
		}

		text = Text{
			Val: textRecord,
		}
	)

	cleanupFn()

	t.Run("success_create", func(t *testing.T) {
		t.Cleanup(cleanupFn)

		var (
			ctx        = context.Background()
			transactor = NewTransactor(
				NewDB(db, sql.TxOptions{}),
			)
			repositoryA = NewTextRepository(transactor, false)
			repositoryB = NewTextRepository(transactor, false)
			useCase     = NewUseCase(repositoryA, repositoryB, transactor)
		)

		err := useCase.CreateText(ctx, text)
		require.NoError(t, err)

		{
			records, err := GetTextRecords(db)
			require.NoError(t, err)
			require.Len(t, records, 2)
			for _, record := range records {
				require.Equal(t, Text{Val: textRecord}, record)
			}
		}
	})
	t.Run("error_and_rollback", func(t *testing.T) {
		t.Cleanup(cleanupFn)

		var (
			ctx        = context.Background()
			transactor = NewTransactor(
				NewDB(db, sql.TxOptions{}),
			)
			repositoryA = NewTextRepository(transactor, false)
			repositoryB = NewTextRepository(transactor, true)
			useCase     = NewUseCase(repositoryA, repositoryB, transactor)
		)

		err := useCase.CreateText(ctx, text)
		require.Error(t, err)
		require.ErrorIs(t, err, entity.ErrExpected)

		{
			records, err := GetTextRecords(db)
			require.NoError(t, err)
			require.Len(t, records, 0)

		}
	})
}

func Test_UseCases(t *testing.T) {
	var (
		db        = ConnectDB(t)
		cleanupFn = func() {
			err := ClearDB(db)
			require.NoError(t, err)
		}
	)

	cleanupFn()

	t.Run("single_repository", func(t *testing.T) {
		t.Run("success_create", func(t *testing.T) {
			t.Cleanup(cleanupFn)

			var (
				ctx        = context.Background()
				transactor = NewTransactor(
					NewDB(db, sql.TxOptions{}),
				)
				repositoryA = NewTextRepository(transactor, false)
				repositoryB = NewTextRepository(transactor, false)
				useCases    = NewUseCases(
					NewUseCase(repositoryA, repositoryB, transactor),
					NewUseCase(repositoryA, repositoryB, transactor),
					transactor,
				)
			)

			err := useCases.CreateTextRecords(ctx, textRecord)
			require.NoError(t, err)

			{
				records, err := GetTextRecords(db)
				require.NoError(t, err)
				require.Len(t, records, 4)
				for _, record := range records {
					require.Equal(t, Text{Val: textRecord}, record)
				}
			}
		})
		t.Run("error_and_rollback", func(t *testing.T) {
			t.Cleanup(cleanupFn)

			var (
				ctx        = context.Background()
				transactor = NewTransactor(
					NewDB(db, sql.TxOptions{}),
				)
				repositoryA = NewTextRepository(transactor, false)
				repositoryB = NewTextRepository(transactor, true)
				useCases    = NewUseCases(
					NewUseCase(repositoryA, repositoryB, transactor),
					NewUseCase(repositoryA, repositoryB, transactor),
					transactor,
				)
			)

			err := useCases.CreateTextRecords(ctx, textRecord)
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
