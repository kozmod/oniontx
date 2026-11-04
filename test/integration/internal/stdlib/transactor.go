package stdlib

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/kozmod/oniontx/mtx"
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

// Tx combines SQL execution and transaction finalization so the savepoint tests
// can wrap an active transaction without losing access to SQL methods.
type Tx interface {
	Executor
	mtx.Tx
}

// Wrapper wraps [sql.DB] and implements [mtx.TxBeginner].
type Wrapper struct {
	*sql.DB
}

// BeginTx starts a transaction.
func (db Wrapper) BeginTx(ctx context.Context) (Tx, error) {
	var txOptions sql.TxOptions
	tx, err := db.DB.BeginTx(ctx, &txOptions)
	return &TxWrapper{Tx: tx}, err
}

// TxWrapper wraps [sql.Tx] and implements [mtx.Tx].
type TxWrapper struct {
	*sql.Tx
}

// Rollback aborts the transaction.
func (t *TxWrapper) Rollback(_ context.Context) error {
	return t.Tx.Rollback()
}

// Commit commits the transaction.
func (t *TxWrapper) Commit(_ context.Context) error {
	return t.Tx.Commit()
}

// SavepointTxWrapper demonstrates custom finalization in the savepoint tests.
// Unlike a full rollback, its Rollback preserves and commits earlier writes.
type SavepointTxWrapper struct {
	Tx
	// Savepoint is an already quoted SQL identifier supplied by TextRepository.Savepoint.
	Savepoint string
}

// Rollback is used by the savepoint tests to undo writes after the savepoint
// and commit the remaining writes. If rollback to the savepoint fails, it calls
// the wrapped transaction's Rollback instead. The transactor retains the callback
// error even when this method succeeds. Panic cleanup uses this method as well.
func (t *SavepointTxWrapper) Rollback(ctx context.Context) error {
	if err := t.RollbackTo(ctx); err != nil {
		return errors.Join(err, t.Tx.Rollback(ctx))
	}
	return t.Commit(ctx)
}

// RollbackTo undoes writes after the savepoint and keeps the transaction open.
// The savepoint tests use it to continue writing before the outer commit.
func (t *SavepointTxWrapper) RollbackTo(ctx context.Context) error {
	_, err := t.Tx.ExecContext(ctx, fmt.Sprintf("ROLLBACK TO %s;", t.Savepoint))
	if err != nil {
		return fmt.Errorf("rollback savepoint [%s]: %w", t.Savepoint, err)
	}
	return nil
}

// Commit commits the outer transaction, including writes before the savepoint.
// It completes the wrapped transaction for the savepoint tests.
func (t *SavepointTxWrapper) Commit(ctx context.Context) error {
	return t.Tx.Commit(ctx)
}

// Transactor manage a transaction for single [sql.DB] instance.
type Transactor struct {
	*ContextOperator[*Wrapper, Tx]
	*mtx.Transactor[*Wrapper, Tx]
}

// NewTransactor returns new [Transactor].
func NewTransactor(db *sql.DB) *Transactor {
	var (
		base       = Wrapper{DB: db}
		operator   = NewContextOperator[*Wrapper, Tx](&base)
		transactor = Transactor{
			ContextOperator: operator,
			Transactor: mtx.NewTransactor[*Wrapper, Tx](&base, operator).
				// Refresh is enabled to demonstrate transaction replacement in the savepoint tests.
				WithTxRefreshFromContext(true),
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

// TryGetTx returns pointer of [sql.Tx] and "true" from [context.Context] or return `false`.
func (t *Transactor) TryGetTx(ctx context.Context) (Tx, bool) {
	wrapper, ok := t.Transactor.TryGetTx(ctx)
	if !ok || wrapper == nil {
		return nil, false
	}
	return wrapper, true
}

// InjectSavepointTx installs a finalization wrapper for the savepoint tests.
// ctx must already contain a transaction stored by ContextOperator; savepoint
// must be a quoted SQL identifier. Refresh lets the outer WithinTx use the wrapper.
func (t *Transactor) InjectSavepointTx(ctx context.Context, tx Tx, savepoint string) {
	t.ContextOperator.Inject(ctx, &SavepointTxWrapper{Tx: tx, Savepoint: savepoint})
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
