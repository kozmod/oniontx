package mongo

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
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
			require.NoError(t, err)
			err = collectionB.Drop(globalCtx)
			require.NoError(t, err)
		}
	)
	defer func() {
		err := client.Disconnect(context.Background())
		require.NoError(t, err)
		cancel()
	}()

	cleanupFn()

	t.Run("single_collection", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			t.Cleanup(cleanupFn)

			var (
				ctx        = context.Background()
				transactor = NewTransactor(NewMongo(client))
				repoA      = NewRepository(collectionA, transactor, false)
				repoB      = NewRepository(collectionA, transactor, false)
			)

			err := transactor.WithinTx(ctx, func(ctx context.Context) error {
				err := repoA.Save(ctx, testDataValA)
				require.NoError(t, err)
				err = repoB.Save(ctx, testDataChange)
				require.NoError(t, err)
				return err
			})
			require.NoError(t, err)

			data, err := GetDataByID(ctx, t, collectionA, testID)
			require.NoError(t, err)
			require.Equal(t, testDataChange, data)
		})
		t.Run("err_and_rollback", func(t *testing.T) {
			t.Cleanup(cleanupFn)

			var (
				ctx        = context.Background()
				transactor = NewTransactor(NewMongo(client))
				repoA      = NewRepository(collectionA, transactor, false)
				repoB      = NewRepository(collectionA, transactor, true)
			)

			err := transactor.WithinTx(ctx, func(ctx context.Context) error {
				err := repoA.Save(ctx, testDataValA)
				require.NoError(t, err)
				err = repoB.Save(ctx, testDataChange)
				require.Error(t, err)
				return err
			})
			require.Error(t, err)

			data, err := GetDataByID(ctx, t, collectionA, testID)
			require.Equal(t, dummyData, data)
			require.Error(t, err)
			require.Containsf(t, err.Error(), ErrTextNoDocResult, "should have returned an error")
		})
	})
	t.Run("multiple_collection", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			t.Cleanup(cleanupFn)

			var (
				ctx        = context.Background()
				transactor = NewTransactor(NewMongo(client))
				repoA      = NewRepository(collectionA, transactor, false)
				repoB      = NewRepository(collectionB, transactor, false)
			)

			err := transactor.WithinTx(ctx, func(ctx context.Context) error {
				err := repoA.Save(ctx, testDataValA)
				require.NoError(t, err)
				err = repoB.Save(ctx, testDataValB)
				require.NoError(t, err)
				return err
			})
			require.NoError(t, err)

			data, err := GetDataByID(ctx, t, collectionA, testID)
			require.NoError(t, err)
			require.Equal(t, testDataValA, data)

			data, err = GetDataByID(ctx, t, collectionB, testID)
			require.NoError(t, err)
			require.Equal(t, testDataValB, data)
		})
		t.Run("err_and_rollback", func(t *testing.T) {
			t.Cleanup(cleanupFn)

			var (
				ctx        = context.Background()
				transactor = NewTransactor(NewMongo(client))
				repoA      = NewRepository(collectionA, transactor, false)
				repoB      = NewRepository(collectionA, transactor, true)
			)

			err := transactor.WithinTx(ctx, func(ctx context.Context) error {
				err := repoA.Save(ctx, testDataValA)
				require.NoError(t, err)
				err = repoB.Save(ctx, testDataValB)
				require.Error(t, err)
				return err
			})
			require.Error(t, err)

			data, err := GetDataByID(ctx, t, collectionA, testID)
			require.Equal(t, dummyData, data)
			require.Error(t, err)
			require.Containsf(t, err.Error(), ErrTextNoDocResult, "should have returned an error")
		})
	})
	t.Run("with_options", func(t *testing.T) {
		t.Run("success__journaled", func(t *testing.T) {
			t.Cleanup(cleanupFn)

			var (
				ctx        = context.Background()
				transactor = NewTransactor(
					NewMongo(client).
						WithTransactionOptions(
							options.Transaction().SetWriteConcern(
								writeconcern.Journaled(),
							),
						),
				)
				repoA = NewRepository(collectionA, transactor, false)
				repoB = NewRepository(collectionA, transactor, false)
			)

			err := transactor.WithinTx(ctx, func(ctx context.Context) error {
				err := repoA.Save(ctx, testDataValA)
				require.NoError(t, err)
				err = repoB.Save(ctx, testDataChange)
				require.NoError(t, err)
				return err
			})
			require.NoError(t, err)

			data, err := GetDataByID(ctx, t, collectionA, testID)
			require.NoError(t, err)
			require.Equal(t, testDataChange, data)
		})
		t.Run("error_start_when_snapshot_session_was_set", func(t *testing.T) {
			t.Cleanup(cleanupFn)

			var (
				ctx        = context.Background()
				transactor = NewTransactor(
					NewMongo(client).
						WithTransactionOptions(
							options.Transaction().SetWriteConcern(
								writeconcern.Journaled(),
							),
						).
						WithSessionOptions(
							// error: transactions are not supported in snapshot sessions
							options.Session().SetSnapshot(true),
						),
				)
				repoA = NewRepository(collectionA, transactor, false)
				repoB = NewRepository(collectionA, transactor, false)
			)

			err := transactor.WithinTx(ctx, func(ctx context.Context) error {
				if err := repoA.Save(ctx, testDataValA); err != nil {
					return fmt.Errorf("fist call in tx: %w", err)
				}
				if err := repoB.Save(ctx, testDataChange); err != nil {
					return fmt.Errorf("second call in tx: %w", err)
				}
				return nil
			})
			require.Error(t, err)
			require.Contains(t, err.Error(), "transactions are not supported in snapshot sessions")

			data, err := GetDataByID(ctx, t, collectionA, testID)
			require.Error(t, err)
			require.Equal(t, dummyData, data)
		})
	})
}
