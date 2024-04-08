package pgx

import (
	"github.com/jackc/pgx/v5"
)

// TxOption implements oniontx.Option.
type TxOption func(opt *pgx.TxOptions)

// Apply the TxOption to [sql.TxOptions].
func (o TxOption) Apply(opt *pgx.TxOptions) {
	o(opt)
}

// WithAccessMode -
func WithAccessMode(mode pgx.TxAccessMode) TxOption {
	return func(opt *pgx.TxOptions) {
		opt.AccessMode = mode
	}
}

func WithDeferrableMode(mode pgx.TxDeferrableMode) TxOption {
	return func(opt *pgx.TxOptions) {
		opt.DeferrableMode = mode
	}
}

func WithIsoLevel(lvl pgx.TxIsoLevel) TxOption {
	return func(opt *pgx.TxOptions) {
		opt.IsoLevel = lvl
	}
}

func WithBeginQuery(beginQuery string) TxOption {
	return func(opt *pgx.TxOptions) {
		opt.BeginQuery = beginQuery
	}
}
