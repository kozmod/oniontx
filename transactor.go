package oniontx

import (
	"context"
	"database/sql"
	"fmt"

	"golang.org/x/xerrors"
)

var (
	ErrNilDB = xerrors.New(" database is nil")
)

type (
	// Executor represents common methods of sql.DB and sql.Tx.
	Executor interface {
		Exec(query string, args ...any) (sql.Result, error)
		ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
		Query(query string, args ...any) (*sql.Rows, error)
		QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
		QueryRow(query string, args ...any) *sql.Row
		QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
		Prepare(query string) (*sql.Stmt, error)
		PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
	}

	transaction interface {
		Executor
		Rollback() error
		Commit() error
	}

	database interface {
		Executor
		BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
	}
)

type ctxKey string

func createKey(in database) ctxKey {
	return ctxKey(fmt.Sprintf("%p", in))
}

// injectTx injects transaction to context.
func injectTx(ctx context.Context, db database, tx transaction) context.Context {
	if db == nil || tx == nil {
		return ctx
	}
	key := createKey(db)
	return context.WithValue(ctx, key, tx)
}

type Transactor struct {
	db        database
	beginTxFn func(ctx context.Context, options *sql.TxOptions) (transaction, error)
}

// NewTransactor creates new  Transactor with pointer of the sql.DB.
func NewTransactor(db *sql.DB) *Transactor {
	var base database
	if db != nil {
		base = db
	}
	return &Transactor{
		db: base,
		beginTxFn: func(ctx context.Context, options *sql.TxOptions) (transaction, error) {
			return db.BeginTx(ctx, options)
		}}
}

// WithinTransaction execute all queries in transaction (create new transaction or reuse transaction obtained from context.Context).
func (t *Transactor) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	return t.WithinOptionalTransaction(ctx, fn)
}

// WithinOptionalTransaction execute all queries in transaction with options (create new transaction or reuse transaction obtained from context.Context).
func (t *Transactor) WithinOptionalTransaction(ctx context.Context, fn func(ctx context.Context) error, options ...Option) (err error) {
	if t.db == nil {
		return xerrors.Errorf("transactor: cannot begin transaction: %w", ErrNilDB)
	}

	var (
		key    = createKey(t.db)
		tx, ok = ctx.Value(key).(transaction)
	)

	if !ok {
		var txOptions sql.TxOptions
		for _, option := range options {
			option(&txOptions)
		}

		tx, err = t.beginTxFn(ctx, &txOptions)
		if err != nil {
			return xerrors.Errorf("transactor: cannot begin transaction: %w", err)
		}
	}

	defer func() {
		switch p := recover(); {
		case p != nil:
			if rbErr := tx.Rollback(); rbErr != nil {
				err = xerrors.Errorf("transactor: tx execute with panic [%v]: rollback err: %w", p, rbErr)
			}
		case err != nil:
			if rbErr := tx.Rollback(); rbErr != nil {
				err = xerrors.Errorf("transactor: tx err [%v] , rollback err: %w", err, rbErr)
			}
		default:
			if err = tx.Commit(); err != nil {
				err = xerrors.Errorf("transactor: cannot commit transaction: %w", err)
			}
		}
	}()

	return fn(injectTx(ctx, t.db, tx))
}

// ExtractExecutorOrDefault extracts Executor (*sql.Tx) from context.Context or return default Executor (*sql.DB).
func (t *Transactor) ExtractExecutorOrDefault(ctx context.Context) Executor {
	var (
		key    = createKey(t.db)
		tx, ok = ctx.Value(key).(transaction)
	)
	if !ok {
		return t.db
	}
	return tx
}

// TryExtractTransaction extracts pointer of sql.Tx from context.Context or return `false`.
func (t *Transactor) TryExtractTransaction(ctx context.Context) (*sql.Tx, bool) {
	var (
		key    = createKey(t.db)
		tx, ok = ctx.Value(key).(*sql.Tx)
	)
	return tx, ok
}

// DB returns pointer of sql.DB which using in Transactor.
func (t *Transactor) DB() *sql.DB {
	return t.db.(*sql.DB)
}
