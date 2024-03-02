package stdlib

import (
	"context"
	"database/sql"

	"github.com/kozmod/oniontx"
)

// Executor represents common methods of sql.DB and sql.Tx.
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

// dbWrapper wraps [sql.DB] and implements [oniontx.TxBeginner].
type dbWrapper struct {
	*sql.DB
}

// BeginTx starts a transaction.
func (db dbWrapper) BeginTx(ctx context.Context, opts ...oniontx.Option[*sql.TxOptions]) (*txWrapper, error) {
	var txOptions sql.TxOptions
	for _, opt := range opts {
		opt.Apply(&txOptions)
	}
	tx, err := db.DB.BeginTx(ctx, &txOptions)
	return &txWrapper{Tx: tx}, err
}

// txWrapper wraps [sql.Tx] and implements [oniontx.Tx].
type txWrapper struct {
	*sql.Tx
}

// Rollback aborts the transaction.
func (t *txWrapper) Rollback(_ context.Context) error {
	return t.Tx.Rollback()
}

// Commit commits the transaction.
func (t *txWrapper) Commit(_ context.Context) error {
	return t.Tx.Commit()
}

// Transactor manage a transaction for single [sql.DB] instance.
type Transactor struct {
	*oniontx.Transactor[*dbWrapper, *txWrapper, *sql.TxOptions]
}

// NewTransactor returns new [Transactor].
func NewTransactor(db *sql.DB) *Transactor {
	var (
		base       = dbWrapper{DB: db}
		operator   = oniontx.NewContextOperator[*dbWrapper, *txWrapper](&base)
		transactor = Transactor{
			Transactor: oniontx.NewTransactor[*dbWrapper, *txWrapper, *sql.TxOptions](&base, operator),
		}
	)
	return &transactor
}

// WithinTx execute all queries with [sql.Tx].
//
// Creates new [sql.Tx] or reuse [sql.Tx] obtained from [context.Context].
func (t *Transactor) WithinTx(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	return t.Transactor.WithinTx(ctx, fn)
}

// WithinTxWithOpts execute all queries with [sql.Tx] and transaction [sql.TxOptions].
//
// Creates new [sql.Tx] or reuse [sql.Tx] obtained from [context.Context].
func (t *Transactor) WithinTxWithOpts(ctx context.Context, fn func(ctx context.Context) error, opts ...oniontx.Option[*sql.TxOptions]) (err error) {
	return t.Transactor.WithinTxWithOpts(ctx, fn, opts...)
}

// TryGetTx returns pointer of [sql.Tx] and "true" from [context.Context] or return `false`.
func (t *Transactor) TryGetTx(ctx context.Context) (*sql.Tx, bool) {
	wrapper, ok := t.Transactor.TryGetTx(ctx)
	if !ok || wrapper == nil || wrapper.Tx == nil {
		return nil, false
	}
	return wrapper.Tx, true
}

// TxBeginner returns pointer of [sql.DB].
func (t *Transactor) TxBeginner() *sql.DB {
	return t.Transactor.TxBeginner().DB
}

// GetExecutor returns [Executor] implementation ([*sql.DB] or [*sql.Tx] default wrappers).
func (t *Transactor) GetExecutor(ctx context.Context) Executor {
	tx, ok := t.Transactor.TryGetTx(ctx)
	if !ok {
		return t.Transactor.TxBeginner()
	}
	return tx
}
