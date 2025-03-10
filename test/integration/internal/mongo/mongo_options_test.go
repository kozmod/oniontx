package mongo

import (
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"

	"github.com/kozmod/oniontx"
)

// TxOption implements [oniontx.Option].
type TxOption func(opts *SessionAndTxOpts)

// Apply the TxOption to [options.TransactionOptions] and [options.SessionOptions].
func (r TxOption) Apply(opt *SessionAndTxOpts) {
	r(opt)
}

// Journaled set [writeconcern.Journaled].
func Journaled(journaled bool) oniontx.Option[*SessionAndTxOpts] {
	return TxOption(func(opt *SessionAndTxOpts) {
		if !journaled {
			return
		}
		*opt.TransactionOptions = append(*opt.TransactionOptions,
			options.Transaction().SetWriteConcern(
				writeconcern.Journaled(),
			))
	})
}

// SetSessionSnapshot set snapshot to session.
func SetSessionSnapshot(snapshot bool) oniontx.Option[*SessionAndTxOpts] {
	return TxOption(func(opt *SessionAndTxOpts) {
		*opt.SessionOptions = append(*opt.SessionOptions,
			options.Session().SetSnapshot(snapshot),
		)
	})
}
