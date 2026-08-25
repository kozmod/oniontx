package saga

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/kozmod/oniontx/internal/testtool"
	"github.com/kozmod/oniontx/saga"
	"github.com/kozmod/oniontx/test/integration/internal/entity"
	"github.com/kozmod/oniontx/test/integration/internal/mongo"
	"github.com/kozmod/oniontx/test/integration/internal/stdlib"

	"github.com/stretchr/testify/require"
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
			require.NoError(t, err)

			err = mongoCollectionA.Drop(globalCtx)
			require.NoError(t, err)
		}
	)

	defer func() {
		err := sqlDB.Close()
		require.NoError(t, err)

		err = mongoClient.Disconnect(context.Background())
		require.NoError(t, err)
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
		res, err := saga.NewSaga([]saga.Step{
			saga.NewStep("step_sql_0").
				WithAction(saga.NewOperation(func(ctx context.Context, _ saga.Track) error {
					err := sqlTransactor.WithinTx(ctx, func(ctx context.Context) error {
						return sqlRepo.Insert(ctx, sqlTextRecord)
					})
					require.NoError(t, err)
					return nil
				})).
				WithCompensation(saga.NewOperation(func(ctx context.Context, _ saga.Track) error {
					require.Fail(t, "should not call (sql)")
					return nil
				})),
			saga.NewStep("step_mongo_0").
				WithAction(saga.NewOperation(func(ctx context.Context, _ saga.Track) error {
					err := mongoTransactor.WithinTx(ctx, func(ctx context.Context) error {
						return mongoRepo.Save(ctx, mongoTestDataValA)
					})
					require.NoError(t, err)
					return nil
				})).
				WithCompensation(saga.NewOperation(func(ctx context.Context, _ saga.Track) error {
					require.Fail(t, "should not call (mongo)")
					return nil
				})),
			saga.NewStep("step_check_all").
				WithAction(saga.NewOperation(func(ctx context.Context, _ saga.Track) error {
					records, err := stdlib.GetTextRecords(sqlDB)
					require.NoError(t, err)
					require.Len(t, records, 1)
					require.ElementsMatch(t, []string{sqlTextRecord}, records)

					data, err := mongo.GetDataByID(ctx, t, mongoCollectionA, mongoTestID)
					require.Equal(t, mongoTestDataValA, data)
					require.NoError(t, err)

					return nil
				})),
		}).Execute(ctx)

		require.NoError(t, err)
		require.Equal(t, saga.StageResultSuccess, res.Status)

		testtool.TestFn(t, func() {
			printResult(t, res, err)
		})
	})

	t.Run("success_compensation", func(t *testing.T) {
		t.Cleanup(cleanupFn)

		var (
			sqlTransactor = stdlib.NewTransactor(sqlDB)
			sqlRepo       = stdlib.NewTextRepository(sqlTransactor, false)

			mongoTransactor = mongo.NewTransactor(mongo.NewMongo(mongoClient))
			mongoRepo       = mongo.NewRepository(mongoCollectionA, mongoTransactor, false)
		)
		res, err := saga.NewSaga([]saga.Step{
			saga.NewStep("step_sql_0").
				WithAction(saga.NewOperation(func(ctx context.Context, _ saga.Track) error {
					err := sqlTransactor.WithinTx(ctx, func(ctx context.Context) error {
						return sqlRepo.Insert(ctx, sqlTextRecord)
					})
					require.NoError(t, err)
					return err
				})).
				WithCompensation(saga.NewOperation(func(ctx context.Context, track saga.Track) error {
					data := track.GetStepData()
					require.Len(t, data.Action.Errors, 0)

					err := sqlTransactor.WithinTx(ctx, func(ctx context.Context) error {
						return sqlRepo.Delete(ctx, sqlTextRecord)
					})
					require.NoError(t, err)
					return err
				})),
			saga.NewStep("step_mongo_0").
				WithAction(saga.NewOperation(func(ctx context.Context, _ saga.Track) error {
					err := mongoTransactor.WithinTx(ctx, func(ctx context.Context) error {
						return mongoRepo.Save(ctx, mongoTestDataValA)
					})
					require.NoError(t, err)
					return err
				})).
				WithCompensation(saga.NewOperation(func(ctx context.Context, track saga.Track) error {
					data := track.GetStepData()
					require.Len(t, data.Action.Errors, 0)

					t.Log(data)
					err := mongoRepo.Delete(ctx, mongoTestDataValA)
					require.NoError(t, err)
					return err
				})),
			saga.NewStep("step_check_all").
				WithAction(saga.NewOperation(func(ctx context.Context, _ saga.Track) error {
					records, err := stdlib.GetTextRecords(sqlDB)
					require.NoError(t, err)
					require.Len(t, records, 1)
					require.ElementsMatch(t, []string{sqlTextRecord}, records)

					data, err := mongo.GetDataByID(ctx, t, mongoCollectionA, mongoTestID)
					require.Equal(t, mongoTestDataValA, data)
					require.NoError(t, err)

					return nil
				})),
			saga.NewStep("step_error").
				WithAction(saga.NewOperation(func(ctx context.Context, _ saga.Track) error {
					return entity.ErrExpected
				})),
		}).Execute(ctx)

		require.Error(t, err)
		require.ErrorIs(t, err, saga.ErrActionFailed)
		require.Equal(t, saga.StageResultCompensated, res.Status)

		testtool.TestFn(t, func() {
			printResult(t, res, err)
		})

		{
			records, err := stdlib.GetTextRecords(sqlDB)
			require.NoError(t, err)
			require.Len(t, records, 0)

			data, err := mongo.GetDataByID(ctx, t, mongoCollectionA, mongoTestID)
			require.Equal(t, mongoDummyData, data)
			require.Error(t, err)
			require.Containsf(t, err.Error(), mongo.ErrTextNoDocResult, "should have returned an error")
		}
	})

	t.Run("success_compensation_in_single_action", func(t *testing.T) {
		t.Cleanup(cleanupFn)
		t.Log("using `CompensationOnActionFailure` flag")

		var (
			sqlTransactor = stdlib.NewTransactor(sqlDB)
			sqlRepo       = stdlib.NewTextRepository(sqlTransactor, false)

			mongoTransactor = mongo.NewTransactor(mongo.NewMongo(mongoClient))
			mongoRepo       = mongo.NewRepository(mongoCollectionA, mongoTransactor, false)
		)
		res, err := saga.NewSaga([]saga.Step{
			saga.NewStep("step_sql_0").
				WithAction(saga.NewOperation(func(ctx context.Context, _ saga.Track) error {
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
					require.Error(t, err)
					return err
				})).
				// Need to add current compensation to list of compensations.
				WithCompensation(saga.NewOperation(func(ctx context.Context, track saga.Track) error {
					// check Mongo entities (commit).
					data, err := mongo.GetDataByID(ctx, t, mongoCollectionA, mongoTestID)
					require.NoError(t, err)
					require.Equal(t, mongoTestDataValA, data)

					// check SQL entities (rollback)
					records, err := stdlib.GetTextRecords(sqlDB)
					require.NoError(t, err)
					require.Len(t, records, 0)

					// Compensation logic.
					//
					// Check an error type and call compensation only for Mongo.
					trackData := track.GetStepData()
					if len(trackData.Action.Errors) > 0 && errors.Is(trackData.Action.Errors[0], entity.ErrExpected) {
						err = mongoRepo.Delete(ctx, mongoTestDataValA)
						require.NoError(t, err)
						return err
					}
					require.Fail(t, "should not have been called")
					return nil
				})).
				WithCompensationOnActionFailure(),
		}).Execute(ctx)

		require.ErrorIs(t, err, saga.ErrActionFailed)
		require.Equal(t, saga.StageResultCompensated, res.Status)

		testtool.TestFn(t, func() {
			printResult(t, res, err)
		})

		{
			records, err := stdlib.GetTextRecords(sqlDB)
			require.NoError(t, err)
			require.Len(t, records, 0)

			data, err := mongo.GetDataByID(ctx, t, mongoCollectionA, mongoTestID)
			require.Equal(t, mongoDummyData, data)
			require.Error(t, err)
			require.Containsf(t, err.Error(), mongo.ErrTextNoDocResult, "should have returned an error")
		}
	})
}
