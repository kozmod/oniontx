package gorm

import (
	"database/sql"
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
func WithReadOnly(readonly bool) TxOption {
	return func(opt *sql.TxOptions) {
		opt.ReadOnly = readonly
	}
}

// WithIsolationLevel set sql.TxOptions isolation level.
//
// Look at [sql.TxOptions.Isolation].
func WithIsolationLevel(level int) TxOption {
	return func(opt *sql.TxOptions) {
		opt.Isolation = sql.IsolationLevel(level)
	}
}
