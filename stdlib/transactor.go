package stdlib

import (
	"context"
	"database/sql"

	"github.com/kozmod/oniontx"
)

// Transactor manage a transaction for single sql.DB instance.
type Transactor struct {
	transactor *oniontx.Transactor[*DB, *Tx, *sql.TxOptions]
	operator   *oniontx.ContextOperator[*DB, *Tx]
}

// NewTransactor returns new Transactor.
func NewTransactor(db *sql.DB) *Transactor {
	var (
		base     = &DB{DB: db}
		operator = oniontx.NewContextOperator[*DB, *Tx](&base)
	)
	return &Transactor{
		operator: operator,
		transactor: oniontx.NewTransactor[*DB, *Tx, *sql.TxOptions](
			base,
			operator,
		),
	}
}

// WithinTx execute all queries with sql.Tx.
// The function create new sql.Tx or reuse sql.Tx obtained from context.Context.
func (t *Transactor) WithinTx(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	return t.transactor.WithinTx(ctx, fn)
}

// WithinTxWithOpts execute all queries with sql.Tx and transaction sql.TxOptions.
// The function create new sql.Tx or reuse sql.Tx obtained from context.Context.
func (t *Transactor) WithinTxWithOpts(ctx context.Context, fn func(ctx context.Context) error, opts ...oniontx.Option[*sql.TxOptions]) (err error) {
	return t.transactor.WithinTxWithOpts(ctx, fn, opts...)
}

// TryGetTx returns pointer of sql.Tx wrapper and "true" from context.Context or return `false`.
func (t *Transactor) TryGetTx(ctx context.Context) (*Tx, bool) {
	return t.transactor.TryGetTx(ctx)
}

// TxBeginner returns pointer of sql.DB wrapper.
func (t *Transactor) TxBeginner() *DB {
	return t.transactor.TxBeginner()
}

// GetExecutor returns Executor implementation (sql.DB or sql.Tx).
func (t *Transactor) GetExecutor(ctx context.Context) Executor {
	tx, ok := t.operator.Extract(ctx)
	if !ok {
		return t.transactor.TxBeginner()
	}
	return tx
}
