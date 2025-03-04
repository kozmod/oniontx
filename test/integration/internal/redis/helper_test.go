package redis

import (
	"context"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"

	"github.com/kozmod/oniontx/test/integration/internal/entity"
)

func Connect(t *testing.T, ctx context.Context) *redis.Client {
	t.Helper()

	client := redis.NewClient(&redis.Options{
		Addr:     entity.RedisHostPort,
		Username: entity.RedisUser,
		Password: entity.RedisUserPassword,
		DB:       0, // use default DB
	})
	err := client.Ping(ctx).Err()
	assert.NoError(t, err)
	return client
}

func LRange(ctx context.Context, t *testing.T, client *redis.Client, key string) []string {
	t.Helper()

	r := client.LRange(ctx, key, 0, -1)
	err := r.Err()
	assert.NoError(t, err)
	return r.Val()
}
