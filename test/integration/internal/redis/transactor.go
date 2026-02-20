package redis

import (
	"context"

	"github.com/redis/go-redis/v9"

	"github.com/kozmod/oniontx/mtx"
)

// Pipeliner represents common methods of [redis.Client] and [redis.Pipeliner].
type Pipeliner interface {
	Process(ctx context.Context, cmd redis.Cmder) error
	redis.StringCmdable
	redis.ListCmdable
}

// redisClientWrapper wraps [redis.Client] and implements [mtx.TxBeginner].
type redisClientWrapper struct {
	*redis.Client
}

// BeginTx starts a transaction.
func (rdb redisClientWrapper) BeginTx(_ context.Context) (*pipelinerWrapper, error) {
	return &pipelinerWrapper{
		Pipeliner: rdb.TxPipeline(),
	}, nil
}

// pipelinerWrapper wraps [redis.Pipeliner] and implements [mtx.Tx].
type pipelinerWrapper struct {
	redis.Pipeliner
}

// Rollback aborts the transaction.
func (t *pipelinerWrapper) Rollback(_ context.Context) error {
	t.Discard()
	return nil
}

// Commit commits the transaction.
func (t *pipelinerWrapper) Commit(ctx context.Context) error {
	c, err := t.Exec(ctx)
	_ = c
	return err
}

// Transactor manage a transaction for single [redis.Client] instance.
type Transactor struct {
	client *redis.Client
	*mtx.Transactor[*redisClientWrapper, *pipelinerWrapper]
}

// NewTransactor returns new [Transactor].
func NewTransactor(client *redis.Client) *Transactor {
	var (
		base       = redisClientWrapper{Client: client}
		operator   = mtx.NewContextOperator[*redisClientWrapper, *pipelinerWrapper](&base)
		transactor = Transactor{
			client:     client,
			Transactor: mtx.NewTransactor[*redisClientWrapper, *pipelinerWrapper](&base, operator),
		}
	)
	return &transactor
}

// WithinTx execute all queries with [redis.Pipeliner].
//
// Creates new [redis.Pipeliner] or reuse [redis.Pipeliner] obtained from [context.Context].
func (t *Transactor) WithinTx(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	return t.Transactor.WithinTx(ctx, fn)
}

// TxBeginner returns pointer of [redis.Pipeliner].
func (t *Transactor) TxBeginner() redis.Pipeliner {
	return t.Transactor.TxBeginner().Pipeline()
}

// GetExecutor returns [Pipliner] implementation ([redis.Client] or [redis.Pipeliner] default wrappers).
func (t *Transactor) GetExecutor(ctx context.Context) Pipeliner {
	tx, ok := t.Transactor.TryGetTx(ctx)
	if !ok {
		return t.Transactor.TxBeginner()
	}
	return tx
}
