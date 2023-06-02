package oniontx

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	"golang.org/x/xerrors"
)

func Test_injectTx(t *testing.T) {
	t.Run("inject_tx", func(t *testing.T) {
		var (
			db  = sql.DB{}
			tx  = sql.Tx{}
			ctx = context.Background()
		)
		newCtx := injectTx(ctx, &db, &tx)
		extractedTx := newCtx.Value(createKey(&db))
		assertTrue(t, extractedTx == &tx)
	})
	t.Run("not_inject_tx_when_db_is_nil", func(t *testing.T) {
		var (
			tx  = sql.Tx{}
			ctx = context.Background()
		)
		newCtx := injectTx(ctx, nil, &tx)
		extractedTx := newCtx.Value(createKey(nil))
		assertTrue(t, extractedTx == nil)
	})
	t.Run("not_inject_tx_when_tx_is_nil", func(t *testing.T) {
		var (
			db  = sql.DB{}
			ctx = context.Background()
		)
		newCtx := injectTx(ctx, &db, nil)
		extractedTx := newCtx.Value(createKey(&db))
		assertTrue(t, extractedTx == nil)
	})
}

func Test_ExtractExecutorOrDefault(t *testing.T) {
	t.Run("return_default_when_tx_is_nil", func(t *testing.T) {
		var (
			defaultDB  = sql.DB{}
			transactor = NewTransactor(&defaultDB)
			ctx        = context.WithValue(context.Background(), &defaultDB, nil)
		)
		executor := transactor.ExtractExecutorOrDefault(ctx)
		assertTrue(t, executor == &defaultDB)
	})
	t.Run("return_new_tx_when_tx_does_not_exist_for_same_db_instance", func(t *testing.T) {
		var (
			db     = sql.DB{}
			txMock = transactionMock{
				commitFn: func() error {
					return nil
				},
			}
			transactor = NewTransactor(&db)
			ctx        = context.Background()
		)
		transactor.beginTxFn = func(ctx context.Context, options *sql.TxOptions) (transaction, error) {
			return &txMock, nil
		}
		err := transactor.WithinTransaction(ctx, func(ctx context.Context) error {
			executor := transactor.ExtractExecutorOrDefault(ctx)
			assertTrue(t, executor == &txMock)
			return nil
		})
		assertTrue(t, err == nil)
	})
	t.Run("return_tx_when_tx_exists_for_same_db_instance", func(t *testing.T) {
		var (
			db     = sql.DB{}
			txMock = transactionMock{
				commitFn: func() error {
					return nil
				},
			}
			transactor = NewTransactor(&db)
			ctx        = injectTx(context.Background(), &db, &txMock)
		)
		err := transactor.WithinTransaction(ctx, func(ctx context.Context) error {
			executor := transactor.ExtractExecutorOrDefault(ctx)
			assertTrue(t, executor == &txMock)
			return nil
		})
		assertTrue(t, err == nil)
	})
	t.Run("return_new_tx_when_tx_exists_but_db_instance_is_diferent", func(t *testing.T) {
		var (
			db     = sql.DB{}
			txMock = transactionMock{
				commitFn: func() error {
					return nil
				},
			}
			txOtherMock = transactionMock{}
			transactor  = NewTransactor(&db)
			ctx         = injectTx(context.Background(), &sql.DB{}, &txOtherMock)
		)
		transactor.beginTxFn = func(ctx context.Context, options *sql.TxOptions) (transaction, error) {
			return &txMock, nil
		}
		err := transactor.WithinTransaction(ctx, func(ctx context.Context) error {
			executor := transactor.ExtractExecutorOrDefault(ctx)
			assertTrue(t, executor == &txMock)
			assertTrue(t, executor != &txOtherMock)
			return nil
		})
		assertTrue(t, err == nil)
	})
}

func Test_Transactor(t *testing.T) {
	t.Run("with_new_tx", func(t *testing.T) {
		t.Run("not_create_tx_when_db_is_nil", func(t *testing.T) {
			var (
				executed bool
			)
			transactor := NewTransactor(nil)
			err := transactor.WithinTransaction(context.Background(), func(ctx context.Context) error {
				executed = true
				return nil
			})
			assertTrue(t, err != nil)
			assertTrue(t, errors.Is(err, ErrNilDB))
			assertTrue(t, !executed)
		})
		t.Run("error_when_tx_commit_error_happen", func(t *testing.T) {
			var (
				executed bool
				expErr   = xerrors.New("commit_error")
				txMock   = transactionMock{
					commitFn: func() error {
						return expErr
					},
				}
				transactor = Transactor{
					db: databaseMock{},
					beginTxFn: func(ctx context.Context, options *sql.TxOptions) (transaction, error) {
						return &txMock, nil
					},
				}
			)

			err := transactor.WithinTransaction(context.Background(), func(ctx context.Context) error {
				executed = true
				return nil
			})
			assertTrue(t, err != nil)
			assertTrue(t, errors.Is(err, expErr))
			assertTrue(t, executed)
		})
		t.Run("error_when_fn_panic_and_rollback_error_happen", func(t *testing.T) {
			var (
				executed bool
				expErr   = xerrors.New("rollback_error")
				txMock   = transactionMock{
					rollbackFn: func() error {
						return expErr
					},
				}
				transactor = Transactor{
					db: databaseMock{},
					beginTxFn: func(ctx context.Context, options *sql.TxOptions) (transaction, error) {
						return &txMock, nil
					},
				}
			)

			err := transactor.WithinTransaction(context.Background(), func(ctx context.Context) error {
				executed = true
				panic("some_panic")
			})
			assertTrue(t, err != nil)
			assertTrue(t, errors.Is(err, expErr))
			assertTrue(t, executed)
		})
		t.Run("error_when_fn_return_error_and_rollback_error_happen", func(t *testing.T) {
			var (
				executed bool
				expFnErr = xerrors.New("fn_error")
				expRbErr = xerrors.New("rollback_error")
				txMock   = transactionMock{
					rollbackFn: func() error {
						return expRbErr
					},
				}
				transactor = Transactor{
					db: databaseMock{},
					beginTxFn: func(ctx context.Context, options *sql.TxOptions) (transaction, error) {
						return &txMock, nil
					},
				}
			)

			err := transactor.WithinTransaction(context.Background(), func(ctx context.Context) error {
				executed = true
				return expFnErr
			})
			assertTrue(t, err != nil)
			assertTrue(t, errors.Is(err, expRbErr))
			assertTrue(t, strings.Contains(err.Error(), expFnErr.Error()))
			assertTrue(t, executed)
		})
	})
	t.Run("with_tx_from_context", func(t *testing.T) {
		t.Run("success_commit_tx_when_tx_is_exists_in_context", func(t *testing.T) {
			var (
				executed, committed bool
				dbMock              = databaseMock{}
				txMock              = transactionMock{
					commitFn: func() error {
						committed = true
						return nil
					},
				}
				dbKey = createKey(&dbMock)
				ctx   = context.WithValue(context.Background(), dbKey, &txMock)

				transactor = Transactor{
					db: &dbMock,
					beginTxFn: func(ctx context.Context, options *sql.TxOptions) (transaction, error) {
						return &txMock, nil
					},
				}
			)

			err := transactor.WithinTransaction(ctx, func(ctx context.Context) error {
				executed = true
				return nil
			})
			assertTrue(t, err == nil)
			assertTrue(t, executed)
			assertTrue(t, committed)
		})
	})
}

func Test_Option(t *testing.T) {
	const (
		isolationLevelSerializable = sql.LevelSerializable
		isolationLevelDefault      = sql.LevelDefault
		readOnly                   = true
	)

	t.Run("WithIsolationLevel", func(t *testing.T) {
		var (
			txOptions = sql.TxOptions{}
			option    = WithIsolationLevel(int(isolationLevelSerializable))
		)
		option(&txOptions)
		assertTrue(t, txOptions.Isolation == isolationLevelSerializable)
		assertTrue(t, !txOptions.ReadOnly)
	})
	t.Run("WithIsolationLevel", func(t *testing.T) {
		var (
			txOptions = sql.TxOptions{}
			option    = WithReadOnly(readOnly)
		)
		option(&txOptions)
		assertTrue(t, txOptions.Isolation == isolationLevelDefault)
		assertTrue(t, txOptions.ReadOnly)
	})
	t.Run("all", func(t *testing.T) {
		var (
			txOptions = sql.TxOptions{}
			options   = []Option{WithReadOnly(readOnly), WithIsolationLevel(int(isolationLevelSerializable))}
		)
		for _, option := range options {
			option(&txOptions)
		}
		assertTrue(t, txOptions.Isolation == isolationLevelSerializable)
		assertTrue(t, txOptions.ReadOnly)
	})
}

// transactionMock was added to avoid to use external dependencies for mocking
func assertTrue(t *testing.T, val bool) {
	t.Helper()
	if !val {
		t.Fatal()
	}
}

// transactionMock was added to avoid to use external dependencies for mocking
type transactionMock struct {
	transaction

	rollbackFn func() error
	commitFn   func() error
}

func (t *transactionMock) Rollback() error {
	return t.rollbackFn()
}

func (t *transactionMock) Commit() error {
	return t.commitFn()
}

// databaseMock was added to avoid to use external dependencies for mocking
type databaseMock struct {
	database
}
