package mongo

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/kozmod/oniontx"
)

// MongoTxOpt transaction and session options.
type SessionAndTxOpts struct {
	SessionOptions     *[]options.Lister[options.SessionOptions]
	TransactionOptions *[]options.Lister[options.TransactionOptions]
}

// GetSessionOptions return all [options.SessionOptions].
func (o SessionAndTxOpts) GetSessionOptions() []options.Lister[options.SessionOptions] {
	if o.SessionOptions == nil {
		return nil
	}
	return *o.SessionOptions
}

// GetTransactionOptions return all [options.TransactionOptions]].
func (o SessionAndTxOpts) GetTransactionOptions() []options.Lister[options.TransactionOptions] {
	if o.TransactionOptions == nil {
		return nil
	}
	return *o.TransactionOptions
}

// mongoClientWrapper wraps [mongo.Client] and implements [oniontx.TxBeginner].
type mongoClientWrapper struct {
	*mongo.Client
}

// BeginTx starts a transaction.
func (c mongoClientWrapper) BeginTx(ctx context.Context, opts ...oniontx.Option[*SessionAndTxOpts]) (*sessionWrapper, error) {
	// need to init options
	mongoOptions := SessionAndTxOpts{
		SessionOptions:     new([]options.Lister[options.SessionOptions]),
		TransactionOptions: new([]options.Lister[options.TransactionOptions]),
	}
	for _, opt := range opts {
		opt.Apply(&mongoOptions)
	}

	session, err := c.Client.StartSession(mongoOptions.GetSessionOptions()...)
	if err != nil {
		return nil, fmt.Errorf("failed to start mongo session: %w", err)
	}
	if err = session.StartTransaction(mongoOptions.GetTransactionOptions()...); err != nil {
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
	*oniontx.Transactor[*mongoClientWrapper, *sessionWrapper, *SessionAndTxOpts]
}

// NewTransactor returns new [Transactor].
func NewTransactor(client *mongo.Client) *Transactor {
	var (
		base       = mongoClientWrapper{Client: client}
		operator   = oniontx.NewContextOperator[*mongoClientWrapper, *sessionWrapper](&base)
		transactor = Transactor{
			Transactor: oniontx.NewTransactor[*mongoClientWrapper, *sessionWrapper, *SessionAndTxOpts](&base, operator),
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

// WithinTxWithOpts execute all queries with [mongo.Session](set [options.SessionOptions] and [options.TransactionOptions].
//
// Creates new [mongo.Session] or reuse [mongo.Session] obtained from [context.Context].
func (t *Transactor) WithinTxWithOpts(ctx context.Context, fn func(ctx context.Context) error, opts ...oniontx.Option[*SessionAndTxOpts]) (err error) {
	return t.Transactor.WithinTxWithOpts(ctx, fn, opts...)
}

// Session returns pointer of [mongo.Session].
func (t *Transactor) Session(ctx context.Context) (*mongo.Session, bool) {
	tx, ok := t.Transactor.TryGetTx(ctx)
	return tx.Session, ok
}
