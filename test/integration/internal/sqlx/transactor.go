package sqlx

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"github.com/kozmod/oniontx/mtx"
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

// Wrapper wraps [sqlx.DB] and implements [oniontx.TxBeginner].
type Wrapper struct {
	*sqlx.DB
}

// BeginTx starts a transaction.
func (w *Wrapper) BeginTx(ctx context.Context) (*TxWrapper, error) {
	var txOptions sql.TxOptions
	tx, err := w.DB.BeginTxx(ctx, &txOptions)
	return &TxWrapper{Tx: tx}, err
}

// TxWrapper wraps [sqlx.Tx] and implements [oniontx.Tx]
type TxWrapper struct {
	*sqlx.Tx
}

// Rollback aborts the transaction.
func (t *TxWrapper) Rollback(_ context.Context) error {
	return t.Tx.Rollback()
}

// Commit commits the transaction.
func (t *TxWrapper) Commit(_ context.Context) error {
	return t.Tx.Commit()
}

// Transactor manage a transaction for single [pgx.Conn] instance.
type Transactor struct {
	*mtx.Transactor[*Wrapper, *TxWrapper]
}

// NewTransactor returns new Transactor ([sqlx] implementation).
func NewTransactor(db *sqlx.DB) *Transactor {
	var (
		base       = Wrapper{DB: db}
		operator   = mtx.NewContextOperator[*Wrapper, *TxWrapper](&base)
		transactor = mtx.NewTransactor[*Wrapper, *TxWrapper](&base, operator)
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
