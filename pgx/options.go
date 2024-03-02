package pgx

import (
	"github.com/jackc/pgx/v5"

	"github.com/kozmod/oniontx"
)

// TxOption implements oniontx.Option.
type TxOption func(opt *pgx.TxOptions)

// Apply the TxOption to [sql.TxOptions].
func (o TxOption) Apply(opt *pgx.TxOptions) {
	o(opt)
}

// WithAccessMode -
func WithAccessMode(mode pgx.TxAccessMode) oniontx.Option[*pgx.TxOptions] {
	return TxOption(func(opt *pgx.TxOptions) {
		opt.AccessMode = mode
	})
}

func WithDeferrableMode(mode pgx.TxDeferrableMode) oniontx.Option[*pgx.TxOptions] {
	return TxOption(func(opt *pgx.TxOptions) {
		opt.DeferrableMode = mode
	})
}

func WithIsoLevel(lvl pgx.TxIsoLevel) oniontx.Option[*pgx.TxOptions] {
	return TxOption(func(opt *pgx.TxOptions) {
		opt.IsoLevel = lvl
	})
}

func WithBeginQuery(beginQuery string) oniontx.Option[*pgx.TxOptions] {
	return TxOption(func(opt *pgx.TxOptions) {
		opt.BeginQuery = beginQuery
	})
}
