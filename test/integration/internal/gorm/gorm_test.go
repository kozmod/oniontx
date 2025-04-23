package gorm

import (
	"context"
	"database/sql"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pkg/errors"
	"testing"

	"github.com/stretchr/testify/assert"

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
			assert.NoError(t, err)
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
		assert.NoError(t, err)

		{
			records, err := GetTextRecords(db)
			assert.NoError(t, err)
			assert.Len(t, records, 2)
			for _, record := range records {
				assert.Equal(t, Text{Val: textRecord}, record)
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
		assert.Error(t, err)
		assert.ErrorIs(t, err, entity.ErrExpected)

		{
			records, err := GetTextRecords(db)
			assert.NoError(t, err)
			assert.Len(t, records, 0)

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
		assert.Error(t, err)

		var pgErr *pgconn.PgError
		assert.True(t, errors.As(err, &pgErr))
		assert.Equal(t, `25006`, pgErr.Code)
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
		assert.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)

		{
			records, err := GetTextRecords(db)
			assert.NoError(t, err)
			assert.Len(t, records, 0)
		}
	})
}

func Test_UseCase_CreateText(t *testing.T) {
	var (
		db        = ConnectDB(t)
		cleanupFn = func() {
			err := ClearDB(db)
			assert.NoError(t, err)
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
		assert.NoError(t, err)

		{
			records, err := GetTextRecords(db)
			assert.NoError(t, err)
			assert.Len(t, records, 2)
			for _, record := range records {
				assert.Equal(t, Text{Val: textRecord}, record)
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
		assert.Error(t, err)
		assert.ErrorIs(t, err, entity.ErrExpected)

		{
			records, err := GetTextRecords(db)
			assert.NoError(t, err)
			assert.Len(t, records, 0)

		}
	})
}

func Test_UseCases(t *testing.T) {
	var (
		db        = ConnectDB(t)
		cleanupFn = func() {
			err := ClearDB(db)
			assert.NoError(t, err)
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
			assert.NoError(t, err)

			{
				records, err := GetTextRecords(db)
				assert.NoError(t, err)
				assert.Len(t, records, 4)
				for _, record := range records {
					assert.Equal(t, Text{Val: textRecord}, record)
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
			assert.Error(t, err)
			assert.ErrorIs(t, err, entity.ErrExpected)

			{
				records, err := GetTextRecords(db)
				assert.NoError(t, err)
				assert.Len(t, records, 0)
			}
		})
	})
}
