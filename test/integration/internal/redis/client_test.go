package redis

import (
	"context"
	"fmt"

	"github.com/kozmod/oniontx/test/integration/internal/entity"
)

type (
	// redisTransactor is a contract of the custom [Transactor].
	redisTransactor interface {
		GetExecutor(ctx context.Context) Pipeliner
	}
)

// Client is the Redis client wrapper.
type Client struct {
	transactor redisTransactor

	// errorExpected - need to emulate error
	errorExpected bool
}

func NewClient(transactor redisTransactor, errorExpected bool) *Client {
	return &Client{
		transactor:    transactor,
		errorExpected: errorExpected,
	}
}

func (r *Client) LPush(ctx context.Context, key, val string) error {
	if r.errorExpected {
		return entity.ErrExpected
	}
	ex := r.transactor.GetExecutor(ctx)
	err := ex.LPush(ctx, key, val).Err()
	if err != nil {
		return fmt.Errorf("redis client: %w", err)
	}
	return nil
}
