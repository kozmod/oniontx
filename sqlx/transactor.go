package sqlx

import (
	"context"
	"database/sql"
	_ "github.com/lib/pq"

	"github.com/jmoiron/sqlx"
	"github.com/kozmod/oniontx"
)

// Executor represents common methods of [sqlx.DB] and [sqlx.Tx].
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

// dbWrapper wraps [sqlx.DB] and implements [oniontx.TxBeginner].
type dbWrapper struct {
	*sqlx.DB
}

// BeginTx starts a transaction.
func (w *dbWrapper) BeginTx(ctx context.Context, opts ...oniontx.Option[*sql.TxOptions]) (*txWrapper, error) {
	var txOptions sql.TxOptions
	for _, opt := range opts {
		opt.Apply(&txOptions)
	}
	tx, err := w.DB.BeginTxx(ctx, &txOptions)
	return &txWrapper{Tx: tx}, err
}

// txWrapper wraps [sqlx.Tx] and implements [oniontx.Tx]
type txWrapper struct {
	*sqlx.Tx
}

// Rollback aborts the transaction.
func (t *txWrapper) Rollback(_ context.Context) error {
	return t.Tx.Rollback()
}

// Commit commits the transaction.
func (t *txWrapper) Commit(_ context.Context) error {
	return t.Tx.Commit()
}

// Transactor manage a transaction for single [pgx.Conn] instance.
type Transactor struct {
	*oniontx.Transactor[*dbWrapper, *txWrapper, *sql.TxOptions]
}

// NewTransactor returns new Transactor ([sqlx] implementation).
func NewTransactor(db *sqlx.DB) *Transactor {
	var (
		base       = dbWrapper{DB: db}
		operator   = oniontx.NewContextOperator[*dbWrapper, *txWrapper](&base)
		transactor = oniontx.NewTransactor[*dbWrapper, *txWrapper, *sql.TxOptions](&base, operator)
	)
	return &Transactor{
		Transactor: transactor,
	}
}

// TryGetTx returns pointer of [sqlx.Tx] and "true" from [context.Context] or return `false`.
func (t *Transactor) TryGetTx(ctx context.Context) (*sqlx.Tx, bool) {
	wrapper, ok := t.Transactor.TryGetTx(ctx)
	if !ok || wrapper == nil || wrapper.Tx == nil {
		return nil, false
	}
	return wrapper.Tx, true
}

// TxBeginner returns pointer of [sqlx.DB].
func (t *Transactor) TxBeginner() *sqlx.DB {
	return t.Transactor.TxBeginner().DB
}

// GetExecutor returns Executor implementation ([*sqlx.DB] or [*sqlx.Tx] default wrappers).
func (t *Transactor) GetExecutor(ctx context.Context) Executor {
	if tx, ok := t.TryGetTx(ctx); ok {
		return tx
	}
	return t.TxBeginner()
}
