package mongo

import (
	"context"
	"fmt"

	"github.com/kozmod/oniontx"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// ClientWrapper wraps [mongo.Client] and implements [oniontx.TxBeginner].
type ClientWrapper struct {
	*mongo.Client

	sessionOpts     []options.Lister[options.SessionOptions]
	transactionOpts []options.Lister[options.TransactionOptions]
}

// NewMongo returns [mongo.Client] wrapper with transaction's options.
func NewMongo(
	client *mongo.Client,
) *ClientWrapper {
	return &ClientWrapper{
		Client: client,
	}
}

// WithSessionOptions adds [options.SessionOptions] to [ClientWrapper].
func (c *ClientWrapper) WithSessionOptions(opts ...options.Lister[options.SessionOptions]) *ClientWrapper {
	if c != nil {
		c.sessionOpts = append(c.sessionOpts, opts...)
	}
	return c
}

// WithTransactionOptions adds [options.TransactionOptions] to [ClientWrapper].
func (c *ClientWrapper) WithTransactionOptions(opts ...options.Lister[options.TransactionOptions]) *ClientWrapper {
	if c != nil {
		c.transactionOpts = append(c.transactionOpts, opts...)
	}
	return c
}

// BeginTx starts a transaction.
func (c *ClientWrapper) BeginTx(ctx context.Context) (*SessionWrapper, error) {
	session, err := c.Client.StartSession(c.sessionOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to start mongo session: %w", err)
	}
	if err = session.StartTransaction(c.transactionOpts...); err != nil {
		defer session.EndSession(ctx)
		return nil, fmt.Errorf("failed to start mongo transaction: %w", err)
	}

	return &SessionWrapper{
		Session: session,
	}, nil
}

// SessionWrapper wraps [mongo.Session] and implements [oniontx.Tx].
type SessionWrapper struct {
	*mongo.Session
}

// Rollback aborts the transaction.
func (t *SessionWrapper) Rollback(ctx context.Context) error {
	defer t.Session.EndSession(ctx)
	return t.Session.AbortTransaction(ctx)
}

// Commit commits the transaction.
func (t *SessionWrapper) Commit(ctx context.Context) error {
	return t.Session.CommitTransaction(ctx)
}

// Transactor manage a transaction for single [redis.Client] instance.
type Transactor struct {
	*oniontx.Transactor[*ClientWrapper, *SessionWrapper]
}

// NewTransactor returns new [Transactor].
func NewTransactor(client *ClientWrapper) *Transactor {
	var (
		operator   = oniontx.NewContextOperator[*ClientWrapper, *SessionWrapper](client)
		transactor = Transactor{
			Transactor: oniontx.NewTransactor[*ClientWrapper, *SessionWrapper](client, operator),
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
