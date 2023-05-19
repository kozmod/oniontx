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

type TxProducer interface {
}

type txKey struct{}

// InjectTx injects transaction to context
func InjectTx(ctx context.Context, tx *sql.Tx) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}

// ExtractExecutor extracts Executor from context
func ExtractExecutor(ctx context.Context, db *sql.DB) (Executor, bool) {
	tx, ok := ctx.Value(txKey{}).(*sql.Tx)
	return tx, ok
}

// ExtractExecutorOrDefault extracts Executor from context or return default
func ExtractExecutorOrDefault(ctx context.Context, db *sql.DB) Executor {
	tx, ok := ctx.Value(txKey{}).(*sql.Tx)
	if ok {
		return tx
	}
	return db
}

type Transactor struct {
	db      *sql.DB
	options sql.TxOptions
}

func NewTransactor(db *sql.DB) *Transactor {
	return &Transactor{db: db, options: sql.TxOptions{}}
}

func NewTransactorWithOptions(db *sql.DB, options sql.TxOptions) *Transactor {
	return &Transactor{db: db, options: options}
}

func (t *Transactor) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	tx, ok := ctx.Value(txKey{}).(*sql.Tx)
	if !ok {
		tx, err = t.db.BeginTx(ctx, &t.options)
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

	return fn(InjectTx(ctx, tx))
}
