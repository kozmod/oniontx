package redis

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_redis_client(t *testing.T) {
	var (
		globalCtx   = context.Background()
		redisClient = Connect(t, globalCtx)
		cleanupFn   = func() {
			err := redisClient.FlushAll(globalCtx).Err()
			require.NoError(t, err)
		}
	)
	defer func() {
		err := redisClient.Close()
		require.NoError(t, err)
	}()

	cleanupFn()

	const (
		lPushKey  = "l-push-key"
		lPushValA = "1"
		lPushValB = "2"
	)

	t.Run("success_LPush", func(t *testing.T) {
		t.Cleanup(cleanupFn)

		var (
			ctx        = context.Background()
			transactor = NewTransactor(redisClient)
			clientA    = NewClient(transactor, false)
			clientB    = NewClient(transactor, false)
		)

		err := transactor.WithinTx(ctx, func(ctx context.Context) error {
			err := clientA.LPush(ctx, lPushKey, lPushValA)
			if err != nil {
				return fmt.Errorf("clientA LPush: %w", err)
			}

			err = clientB.LPush(ctx, lPushKey, lPushValB)
			if err != nil {
				return fmt.Errorf("clientB LPush: %w", err)
			}
			return err
		})
		require.NoError(t, err)

		result := LRange(ctx, t, redisClient, lPushKey)
		require.ElementsMatch(t, result, []string{lPushValA, lPushValB})
	})
	t.Run("err_and_discard_LPush", func(t *testing.T) {
		t.Cleanup(cleanupFn)

		var (
			ctx        = context.Background()
			transactor = NewTransactor(redisClient)
			clientA    = NewClient(transactor, false)
			clientB    = NewClient(transactor, true)
		)

		err := transactor.WithinTx(ctx, func(ctx context.Context) error {
			err := clientA.LPush(ctx, lPushKey, lPushValA)
			if err != nil {
				return fmt.Errorf("clientA LPush: %w", err)
			}

			err = clientB.LPush(ctx, lPushKey, lPushValB)
			if err != nil {
				return fmt.Errorf("clientB LPush: %w", err)
			}
			return err
		})
		require.Error(t, err)

		result := LRange(ctx, t, redisClient, lPushKey)
		require.Empty(t, result)
	})
}
