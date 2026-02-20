package saga

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/kozmod/oniontx/saga"
	"github.com/kozmod/oniontx/test/integration/internal/entity"
	"github.com/kozmod/oniontx/test/integration/internal/mongo"
	"github.com/kozmod/oniontx/test/integration/internal/stdlib"

	"github.com/stretchr/testify/assert"
)

func Test_Saga_multi_Facade(t *testing.T) {
	const (
		sqlTextRecord = "text_SAGA_2"

		testMongoDB          = "test_SAGA"
		testMongoCollectionA = "test_SAGA_collection_A"
		mongoTestID          = 1
	)

	var (
		ctx               = context.Background()
		globalCtx, cancel = context.WithTimeout(ctx, 10*time.Second)

		sqlDB = stdlib.ConnectDB(t)

		mongoClient       = mongo.Connect(globalCtx, t)
		mongoCollectionA  = mongoClient.Database(testMongoDB).Collection(testMongoCollectionA)
		mongoTestDataValA = mongo.Data{
			ID:  mongoTestID,
			Val: "data_value_SAGA_A",
		}
		mongoDummyData = mongo.Data{}

		cleanupFn = func() {
			err := stdlib.ClearDB(sqlDB)
			assert.NoError(t, err)

			err = mongoCollectionA.Drop(globalCtx)
			assert.NoError(t, err)
		}
	)

	defer func() {
		err := sqlDB.Close()
		assert.NoError(t, err)

		err = mongoClient.Disconnect(context.Background())
		assert.NoError(t, err)
		cancel()
	}()

	cleanupFn()

	t.Run("success_exec_v1", func(t *testing.T) {
		t.Cleanup(cleanupFn)

		var (
			sqlTransactor = stdlib.NewTransactor(sqlDB)
			sqlRepo       = stdlib.NewTextRepository(sqlTransactor, false)

			mongoTransactor = mongo.NewTransactor(mongo.NewMongo(mongoClient))
			mongoRepo       = mongo.NewRepository(mongoCollectionA, mongoTransactor, false)
		)
		err := saga.NewSaga([]saga.Step{
			{
				Name: "step_sql_0",
				Action: func(ctx context.Context) error {
					err := sqlTransactor.WithinTx(ctx, func(ctx context.Context) error {
						return sqlRepo.Insert(ctx, sqlTextRecord)
					})
					assert.NoError(t, err)
					return nil
				},
				Compensation: func(ctx context.Context, _ error) error {
					assert.Fail(t, "should not call (sql)")
					return nil
				},
			},
			{
				Name: "step_mongo_0",
				Action: func(ctx context.Context) error {
					err := mongoTransactor.WithinTx(ctx, func(ctx context.Context) error {
						return mongoRepo.Save(ctx, mongoTestDataValA)
					})
					assert.NoError(t, err)
					return nil
				},
				Compensation: func(ctx context.Context, _ error) error {
					assert.Fail(t, "should not call (mongo)")
					return nil
				},
			},
			{
				Name: "step_check_all",
				Action: func(ctx context.Context) error {
					records, err := stdlib.GetTextRecords(sqlDB)
					assert.NoError(t, err)
					assert.Len(t, records, 1)
					assert.ElementsMatch(t, []string{sqlTextRecord}, records)

					data, err := mongo.GetDataByID(ctx, t, mongoCollectionA, mongoTestID)
					assert.Equal(t, mongoTestDataValA, data)
					assert.NoError(t, err)

					return nil
				},
			},
		}).Execute(ctx)

		assert.NoError(t, err)
	})

	t.Run("success_compensation", func(t *testing.T) {
		t.Cleanup(cleanupFn)

		var (
			sqlTransactor = stdlib.NewTransactor(sqlDB)
			sqlRepo       = stdlib.NewTextRepository(sqlTransactor, false)

			mongoTransactor = mongo.NewTransactor(mongo.NewMongo(mongoClient))
			mongoRepo       = mongo.NewRepository(mongoCollectionA, mongoTransactor, false)
		)
		err := saga.NewSaga([]saga.Step{
			{
				Name: "step_sql_0",
				Action: func(ctx context.Context) error {
					err := sqlTransactor.WithinTx(ctx, func(ctx context.Context) error {
						return sqlRepo.Insert(ctx, sqlTextRecord)
					})
					assert.NoError(t, err)
					return nil
				},
				Compensation: func(ctx context.Context, aroseErr error) error {
					assert.Error(t, aroseErr)
					assert.ErrorIs(t, aroseErr, entity.ErrExpected)

					err := sqlTransactor.WithinTx(ctx, func(ctx context.Context) error {
						return sqlRepo.Delete(ctx, sqlTextRecord)
					})
					assert.NoError(t, err)
					return nil
				},
			},
			{
				Name: "step_mongo_0",
				Action: func(ctx context.Context) error {
					err := mongoTransactor.WithinTx(ctx, func(ctx context.Context) error {
						return mongoRepo.Save(ctx, mongoTestDataValA)
					})
					assert.NoError(t, err)
					return nil
				},
				Compensation: func(ctx context.Context, aroseErr error) error {
					assert.Error(t, aroseErr)
					assert.ErrorIs(t, aroseErr, entity.ErrExpected)

					err := mongoRepo.Delete(ctx, mongoTestDataValA)
					assert.NoError(t, err)
					return err
				},
			},
			{
				Name: "step_check_all",
				Action: func(ctx context.Context) error {
					records, err := stdlib.GetTextRecords(sqlDB)
					assert.NoError(t, err)
					assert.Len(t, records, 1)
					assert.ElementsMatch(t, []string{sqlTextRecord}, records)

					data, err := mongo.GetDataByID(ctx, t, mongoCollectionA, mongoTestID)
					assert.Equal(t, mongoTestDataValA, data)
					assert.NoError(t, err)

					return nil
				},
			},
			{
				Name: "step_error",
				Action: func(ctx context.Context) error {
					return entity.ErrExpected
				},
			},
		}).Execute(ctx)

		assert.Error(t, err)
		assert.ErrorIs(t, err, entity.ErrExpected)

		t.Logf("test error output: \n{\n%v\n}", err)

		{
			records, err := stdlib.GetTextRecords(sqlDB)
			assert.NoError(t, err)
			assert.Len(t, records, 0)

			data, err := mongo.GetDataByID(ctx, t, mongoCollectionA, mongoTestID)
			assert.Equal(t, mongoDummyData, data)
			assert.Error(t, err)
			assert.Containsf(t, err.Error(), mongo.ErrTextNoDocResult, "should have returned an error")
		}
	})

	t.Run("success_compensation_in_single_action", func(t *testing.T) {
		t.Cleanup(cleanupFn)
		t.Log("using `CompensationOnFail` flag")

		var (
			sqlTransactor = stdlib.NewTransactor(sqlDB)
			sqlRepo       = stdlib.NewTextRepository(sqlTransactor, false)

			mongoTransactor = mongo.NewTransactor(mongo.NewMongo(mongoClient))
			mongoRepo       = mongo.NewRepository(mongoCollectionA, mongoTransactor, false)
		)
		err := saga.NewSaga([]saga.Step{
			{
				Name: "step_sql_0",
				Action: func(ctx context.Context) error {
					// The parent [Transactor] which maintain SQL transactions.
					err := sqlTransactor.WithinTx(ctx, func(ctx context.Context) error {
						err := sqlRepo.Insert(ctx, sqlTextRecord)
						if err != nil {
							return fmt.Errorf("1 sql insert failed: %w", err)
						}

						// The child [Transactor] which maintain Mongo transactions.
						err = mongoTransactor.WithinTx(ctx, func(ctx context.Context) error {
							return mongoRepo.Save(ctx, mongoTestDataValA)
						})
						if err != nil {
							return fmt.Errorf("1 mongo save failed: %w", err)
						}
						err = sqlRepo.Insert(ctx, sqlTextRecord)
						if err != nil {
							return fmt.Errorf("2 sql insert failed: %w", err)
						}

						// Because Mongo transaction was commited, need to imitate an error
						// in the last step for the parent [Transactor] (sql).
						return entity.ErrExpected
					})
					assert.Error(t, err)
					return err
				},
				// Need to add current compensation to list of compensations.
				CompensationOnFail: true,
				Compensation: func(ctx context.Context, aroseErr error) error {
					// check Mongo entities (commit).
					data, err := mongo.GetDataByID(ctx, t, mongoCollectionA, mongoTestID)
					assert.NoError(t, err)
					assert.Equal(t, mongoTestDataValA, data)

					// check SQL entities (rollback)
					records, err := stdlib.GetTextRecords(sqlDB)
					assert.NoError(t, err)
					assert.Len(t, records, 0)

					// Compensation logic.
					//
					// Check an error type and call compensation only for Mongo.
					if aroseErr != nil && errors.Is(aroseErr, entity.ErrExpected) {
						err = mongoRepo.Delete(ctx, mongoTestDataValA)
						assert.NoError(t, err)
						return err
					}
					assert.Fail(t, "should not have been called")
					return nil
				},
			},
		}).Execute(ctx)

		assert.ErrorIs(t, err, entity.ErrExpected)

		t.Logf("test error output: \n{\n%v\n}", err)

		{
			records, err := stdlib.GetTextRecords(sqlDB)
			assert.NoError(t, err)
			assert.Len(t, records, 0)

			data, err := mongo.GetDataByID(ctx, t, mongoCollectionA, mongoTestID)
			assert.Equal(t, mongoDummyData, data)
			assert.Error(t, err)
			assert.Containsf(t, err.Error(), mongo.ErrTextNoDocResult, "should have returned an error")
		}
	})
}
