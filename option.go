package oniontx

import "database/sql"

// Option set options of the transactions (sql.TxOptions).
type Option func(client *sql.TxOptions)

// WithReadOnly set `ReadOnly` transaction's option.
func WithReadOnly(readonly bool) Option {
	return func(opts *sql.TxOptions) {
		opts.ReadOnly = readonly
	}
}

// WithIsolationLevel set the transaction isolation level.
func WithIsolationLevel(level int) Option {
	return func(opts *sql.TxOptions) {
		opts.Isolation = sql.IsolationLevel(level)
	}
}
