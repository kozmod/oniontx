package oniontx

import (
	"context"
	"database/sql"

	"golang.org/x/xerrors"
)

type Executor interface {
	Exec(query string, args ...any) (sql.Result, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	Prepare(query string) (*sql.Stmt, error)
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
}

// injectTx injects transaction to context
func injectTx(ctx context.Context, db *sql.DB, tx *sql.Tx) context.Context {
	if db == nil {
		return ctx
	}
	return context.WithValue(ctx, db, tx)
}

// ExtractExecutorOrDefault extracts Executor (*sql.Tx) from context or return default Executor (*sql.DB)
func ExtractExecutorOrDefault(ctx context.Context, db *sql.DB) Executor {
	tx, ok := ctx.Value(db).(*sql.Tx)
	if !ok {
		return db
	}
	return tx
}

type Transactor struct {
	db *sql.DB
}

func NewTransactor(db *sql.DB) *Transactor {
	return &Transactor{db: db}
}

// WithinTransaction execute all queries in transaction (create new transaction or reuse transaction obtained from context.Context)
func (t *Transactor) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error, options ...Option) (err error) {
	tx, ok := ctx.Value(t.db).(*sql.Tx)
	if !ok {
		if t.db == nil {
			return xerrors.Errorf("transactor: cannot begin transaction: database is nil")
		}

		var txOptions sql.TxOptions
		for _, option := range options {
			option(&txOptions)
		}

		tx, err = t.db.BeginTx(ctx, &txOptions)
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
				err = xerrors.Errorf("transactor: cannot commit transaction: %v", err)
			}
		}
	}()

	return fn(injectTx(ctx, t.db, tx))
}

// ExtractExecutorOrDefault extracts Executor (*sql.Tx) from context or return default Executor (*sql.DB)
func (t *Transactor) ExtractExecutorOrDefault(ctx context.Context) Executor {
	return ExtractExecutorOrDefault(ctx, t.db)
}
