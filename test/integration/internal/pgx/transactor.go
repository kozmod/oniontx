package pgx

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/kozmod/oniontx/mtx"
)

// Executor represents common methods of [pgx.Conn] and [pgx.Tx].
type Executor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (commandTag pgconn.CommandTag, err error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Prepare(ctx context.Context, name, sql string) (sd *pgconn.StatementDescription, err error)
}

// Wrapper wraps [pgx.Conn] and implements [mtx.TxBeginner].
type Wrapper struct {
	*pgx.Conn
}

// BeginTx starts a transaction.
func (w *Wrapper) BeginTx(ctx context.Context) (*TxWrapper, error) {
	var txOptions pgx.TxOptions
	tx, err := w.Conn.BeginTx(ctx, txOptions)
	return &TxWrapper{Tx: tx}, err
}

// TxWrapper wraps [pgx.Tx] and implements [mtx.Tx]
type TxWrapper struct {
	pgx.Tx
}

// Rollback aborts the transaction.
func (t *TxWrapper) Rollback(ctx context.Context) error {
	return t.Tx.Rollback(ctx)
}

// Commit commits the transaction.
func (t *TxWrapper) Commit(ctx context.Context) error {
	return t.Tx.Commit(ctx)
}

// Transactor manage a transaction for single [pgx.Conn] instance.
type Transactor struct {
	*mtx.Transactor[*Wrapper, *TxWrapper]
}

// NewTransactor returns new Transactor ([pgx] implementation).
func NewTransactor(conn *pgx.Conn) *Transactor {
	var (
		base       = Wrapper{Conn: conn}
		operator   = mtx.NewContextOperator[*Wrapper, *TxWrapper](&base)
		transactor = mtx.NewTransactor[*Wrapper, *TxWrapper](&base, operator)
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
