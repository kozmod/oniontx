package mongo

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/kozmod/oniontx"
)

// mongoTxOpt is the alias for list of Mongo transaction options.
type mongoTxOpt = []options.Lister[options.TransactionOptions]

// mongoClientWrapper wraps [mongo.Client] and implements [oniontx.TxBeginner].
type mongoClientWrapper struct {
	*mongo.Client
}

// BeginTx starts a transaction.
func (c mongoClientWrapper) BeginTx(ctx context.Context, opts ...oniontx.Option[mongoTxOpt]) (*sessionWrapper, error) {
	// need to init options
	var mongoTxOptions []options.Lister[options.TransactionOptions]
	for _, opt := range opts {
		opt.Apply(mongoTxOptions)
	}

	session, err := c.Client.StartSession()
	if err != nil {
		return nil, fmt.Errorf("failed to start mongo session: %w", err)
	}
	if err = session.StartTransaction(mongoTxOptions...); err != nil {
		defer session.EndSession(ctx)
		return nil, fmt.Errorf("failed to start mongo transaction: %w", err)
	}

	return &sessionWrapper{
		Session: session,
	}, nil
}

// sessionWrapper wraps [mongo.Session] and implements [oniontx.Tx].
type sessionWrapper struct {
	*mongo.Session
}

// Rollback aborts the transaction.
func (t *sessionWrapper) Rollback(ctx context.Context) error {
	defer t.Session.EndSession(ctx)
	return t.Session.AbortTransaction(ctx)
}

// Commit commits the transaction.
func (t *sessionWrapper) Commit(ctx context.Context) error {
	return t.Session.CommitTransaction(ctx)
}

// Transactor manage a transaction for single [redis.Client] instance.
type Transactor struct {
	*oniontx.Transactor[*mongoClientWrapper, *sessionWrapper, mongoTxOpt]
}

// NewTransactor returns new [Transactor].
func NewTransactor(client *mongo.Client) *Transactor {
	var (
		base       = mongoClientWrapper{Client: client}
		operator   = oniontx.NewContextOperator[*mongoClientWrapper, *sessionWrapper](&base)
		transactor = Transactor{
			Transactor: oniontx.NewTransactor[*mongoClientWrapper, *sessionWrapper, mongoTxOpt](&base, operator),
		}
	)
	return &transactor
}

// WithinTx execute all queries with [mongo.Session].
//
// Creates new [mongo.Session] or reuse [mongo.Session] obtained from [context.Context].
func (t *Transactor) WithinTx(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	return t.Transactor.WithinTx(ctx, fn)
}

// Session returns pointer of [mongo.Session].
func (t *Transactor) Session(ctx context.Context) (*mongo.Session, bool) {
	tx, ok := t.Transactor.TryGetTx(ctx)
	return tx.Session, ok
}
