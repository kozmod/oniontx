package mongo

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test(t *testing.T) {
	const (
		testID          = 1
		testDB          = "test"
		testCollectionA = "test_collection_A"
		testCollectionB = "test_collection_B"
	)
	var (
		testDataValA = Data{
			ID:  testID,
			Val: "data_value_A",
		}
		testDataValB = Data{
			ID:  testID,
			Val: "data_value_B",
		}
		testDataChange = Data{
			ID:  testID,
			Val: "changed_data_value",
		}
		dummyData = Data{}
	)
	var (
		globalCtx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		client            = Connect(globalCtx, t)
		collectionA       = client.Database(testDB).Collection(testCollectionA)
		collectionB       = client.Database(testDB).Collection(testCollectionB)
		cleanupFn         = func() {
			err := collectionA.Drop(globalCtx)
			assert.NoError(t, err)
			err = collectionB.Drop(globalCtx)
			assert.NoError(t, err)
		}
	)
	defer func() {
		err := client.Disconnect(context.Background())
		assert.NoError(t, err)
		cancel()
	}()

	cleanupFn()

	t.Run("single_collection", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			t.Cleanup(cleanupFn)

			var (
				ctx        = context.Background()
				transactor = NewTransactor(client)
				repoA      = NewRepository(collectionA, transactor, false)
				repoB      = NewRepository(collectionA, transactor, false)
			)

			err := transactor.WithinTx(ctx, func(ctx context.Context) error {
				err := repoA.Save(ctx, testDataValA)
				assert.NoError(t, err)
				err = repoB.Save(ctx, testDataChange)
				assert.NoError(t, err)
				return err
			})
			assert.NoError(t, err)

			data, err := GetDataByID(ctx, t, collectionA, testID)
			assert.NoError(t, err)
			assert.Equal(t, testDataChange, data)
		})
		t.Run("err_and_rollback", func(t *testing.T) {
			t.Cleanup(cleanupFn)

			var (
				ctx        = context.Background()
				transactor = NewTransactor(client)
				repoA      = NewRepository(collectionA, transactor, false)
				repoB      = NewRepository(collectionA, transactor, true)
			)

			err := transactor.WithinTx(ctx, func(ctx context.Context) error {
				err := repoA.Save(ctx, testDataValA)
				assert.NoError(t, err)
				err = repoB.Save(ctx, testDataChange)
				assert.Error(t, err)
				return err
			})
			assert.Error(t, err)

			data, err := GetDataByID(ctx, t, collectionA, testID)
			assert.Equal(t, dummyData, data)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "mongo: no documents in result")
		})
	})
	t.Run("multiple_collection", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			t.Cleanup(cleanupFn)

			var (
				ctx        = context.Background()
				transactor = NewTransactor(client)
				repoA      = NewRepository(collectionA, transactor, false)
				repoB      = NewRepository(collectionB, transactor, false)
			)

			err := transactor.WithinTx(ctx, func(ctx context.Context) error {
				err := repoA.Save(ctx, testDataValA)
				assert.NoError(t, err)
				err = repoB.Save(ctx, testDataValB)
				assert.NoError(t, err)
				return err
			})
			assert.NoError(t, err)

			data, err := GetDataByID(ctx, t, collectionA, testID)
			assert.NoError(t, err)
			assert.Equal(t, testDataValA, data)

			data, err = GetDataByID(ctx, t, collectionB, testID)
			assert.NoError(t, err)
			assert.Equal(t, testDataValB, data)
		})
		t.Run("err_and_rollback", func(t *testing.T) {
			t.Cleanup(cleanupFn)

			var (
				ctx        = context.Background()
				transactor = NewTransactor(client)
				repoA      = NewRepository(collectionA, transactor, false)
				repoB      = NewRepository(collectionA, transactor, true)
			)

			err := transactor.WithinTx(ctx, func(ctx context.Context) error {
				err := repoA.Save(ctx, testDataValA)
				assert.NoError(t, err)
				err = repoB.Save(ctx, testDataValB)
				assert.Error(t, err)
				return err
			})
			assert.Error(t, err)

			data, err := GetDataByID(ctx, t, collectionA, testID)
			assert.Equal(t, dummyData, data)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "mongo: no documents in result")
		})
	})
	t.Run("with_options", func(t *testing.T) {
		t.Run("success__journaled", func(t *testing.T) {
			t.Cleanup(cleanupFn)

			var (
				ctx        = context.Background()
				transactor = NewTransactor(client)
				repoA      = NewRepository(collectionA, transactor, false)
				repoB      = NewRepository(collectionA, transactor, false)
			)

			err := transactor.WithinTxWithOpts(ctx, func(ctx context.Context) error {
				err := repoA.Save(ctx, testDataValA)
				assert.NoError(t, err)
				err = repoB.Save(ctx, testDataChange)
				assert.NoError(t, err)
				return err
			},
				Journaled(true),
			)
			assert.NoError(t, err)

			data, err := GetDataByID(ctx, t, collectionA, testID)
			assert.NoError(t, err)
			assert.Equal(t, testDataChange, data)
		})
		t.Run("error_start_when_snapshot_session_was_set", func(t *testing.T) {
			t.Cleanup(cleanupFn)

			var (
				ctx        = context.Background()
				transactor = NewTransactor(client)
				repoA      = NewRepository(collectionA, transactor, false)
				repoB      = NewRepository(collectionA, transactor, false)
			)

			err := transactor.WithinTxWithOpts(ctx, func(ctx context.Context) error {
				if err := repoA.Save(ctx, testDataValA); err != nil {
					return fmt.Errorf("fist call in tx: %w", err)
				}
				if err := repoB.Save(ctx, testDataChange); err != nil {
					return fmt.Errorf("second call in tx: %w", err)
				}
				return nil
			},
				SetSessionSnapshot(true), // error: transactions are not supported in snapshot sessions
				Journaled(true),
			)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "transactions are not supported in snapshot sessions")

			data, err := GetDataByID(ctx, t, collectionA, testID)
			assert.Error(t, err)
			assert.Equal(t, dummyData, data)
		})
	})
}
