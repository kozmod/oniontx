package sqlx

import (
	"database/sql"

	"github.com/kozmod/oniontx"
)

// TxOption implements oniontx.Option.
type TxOption func(opt *sql.TxOptions)

// Apply the TxOption to [sql.TxOptions].
func (o TxOption) Apply(opt *sql.TxOptions) {
	o(opt)
}

// WithReadOnly set `ReadOnly` sql.TxOptions option.
//
// Look at [sql.TxOptions.ReadOnly].
func WithReadOnly(readonly bool) oniontx.Option[*sql.TxOptions] {
	return TxOption(func(opt *sql.TxOptions) {
		opt.ReadOnly = readonly
	})
}

// WithIsolationLevel set sql.TxOptions isolation level.
//
// Look at [sql.TxOptions.Isolation].
func WithIsolationLevel(level int) oniontx.Option[*sql.TxOptions] {
	return TxOption(func(opt *sql.TxOptions) {
		opt.Isolation = sql.IsolationLevel(level)
	})
}
