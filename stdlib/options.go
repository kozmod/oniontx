package stdlib

import (
	"database/sql"

	"github.com/kozmod/oniontx"
)

// TxOption implements oniontx.Option.
type TxOption func(opt *sql.TxOptions)

// Apply the TxOption to sql.TxOptions.
func (r TxOption) Apply(opt *sql.TxOptions) {
	r(opt)
}

// WithReadOnly set `ReadOnly` sql.TxOptions option.
func WithReadOnly(readonly bool) oniontx.Option[*sql.TxOptions] {
	return TxOption(func(opt *sql.TxOptions) {
		opt.ReadOnly = readonly
	})
}

// WithIsolationLevel set sql.TxOptions isolation level.
func WithIsolationLevel(level int) oniontx.Option[*sql.TxOptions] {
	return TxOption(func(opt *sql.TxOptions) {
		opt.Isolation = sql.IsolationLevel(level)
	})
}
