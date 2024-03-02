package pgx

import (
	"context"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/jackc/pgx/v5"

	"github.com/kozmod/oniontx"
)

// Executor represents common methods of [pgx.Conn] and [pgx.Tx].
type Executor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (commandTag pgconn.CommandTag, err error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Prepare(ctx context.Context, name, sql string) (sd *pgconn.StatementDescription, err error)
}

// dbWrapper wraps [pgx.Conn] and implements [oniontx.TxBeginner].
type dbWrapper struct {
	*pgx.Conn
}

// BeginTx starts a transaction.
func (w *dbWrapper) BeginTx(ctx context.Context, opts ...oniontx.Option[*pgx.TxOptions]) (*txWrapper, error) {
	var txOptions pgx.TxOptions
	for _, opt := range opts {
		opt.Apply(&txOptions)
	}
	tx, err := w.Conn.BeginTx(ctx, txOptions)
	return &txWrapper{Tx: tx}, err
}

// txWrapper wraps [pgx.Tx] and implements [oniontx.Tx]
type txWrapper struct {
	pgx.Tx
}

// Rollback aborts the transaction.
func (t *txWrapper) Rollback(ctx context.Context) error {
	return t.Tx.Rollback(ctx)
}

// Commit commits the transaction.
func (t *txWrapper) Commit(ctx context.Context) error {
	return t.Tx.Commit(ctx)
}

// Transactor manage a transaction for single [pgx.Conn] instance.
type Transactor struct {
	*oniontx.Transactor[*dbWrapper, *txWrapper, *pgx.TxOptions]
}

// NewTransactor returns new Transactor ([pgx] implementation).
func NewTransactor(conn *pgx.Conn) *Transactor {
	var (
		base       = dbWrapper{Conn: conn}
		operator   = oniontx.NewContextOperator[*dbWrapper, *txWrapper](&base)
		transactor = oniontx.NewTransactor[*dbWrapper, *txWrapper, *pgx.TxOptions](&base, operator)
	)
	return &Transactor{
		Transactor: transactor,
	}
}

// TryGetTx returns pointer of [pgx.Tx] and "true" from [context.Context] or return `false`.
func (t *Transactor) TryGetTx(ctx context.Context) (pgx.Tx, bool) {
	wrapper, ok := t.Transactor.TryGetTx(ctx)
	if !ok || wrapper == nil || wrapper.Tx == nil {
		return nil, false
	}
	return wrapper.Tx, true
}

// TxBeginner returns pointer of [pgx.Conn].
func (t *Transactor) TxBeginner() *pgx.Conn {
	return t.Transactor.TxBeginner().Conn
}

// GetExecutor returns Executor implementation ([*pgx.Conn] or [*pgx.Tx] default wrappers).
func (t *Transactor) GetExecutor(ctx context.Context) Executor {
	if tx, ok := t.TryGetTx(ctx); ok {
		return tx
	}
	return t.TxBeginner()
}
