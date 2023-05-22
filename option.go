package oniontx

import "database/sql"

type Option func(client *sql.TxOptions)

func WithReadOnly(readonly bool) Option {
	return func(opts *sql.TxOptions) {
		opts.ReadOnly = readonly
	}
}

func WithIsolationLevel(level int) Option {
	return func(opts *sql.TxOptions) {
		opts.Isolation = sql.IsolationLevel(level)
	}
}
