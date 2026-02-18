package saga

import (
	"context"
	"testing"
	"time"

	"github.com/kozmod/oniontx"
	"github.com/kozmod/oniontx/test/integration/internal/entity"
	"github.com/kozmod/oniontx/test/integration/internal/mongo"
	"github.com/kozmod/oniontx/test/integration/internal/stdlib"

	"github.com/stretchr/testify/assert"
)

func Test_Saga_multi_Facade(t *testing.T) {
	const (
		sqlTextRecord = "text_SAGA_2"

		testMongoDB          = "test_SAGE"
		testMongoCollectionA = "test_SAGA_collection_A"
		mongoTestID          = 1
	)

	var (
		globalCtx, cancel = context.WithTimeout(context.Background(), 10*time.Second)

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
			ctx           = context.Background()
			sqlTransactor = stdlib.NewTransactor(sqlDB)
			sqlRepo       = stdlib.NewTextRepository(sqlTransactor, false)

			mongoTransactor = mongo.NewTransactor(mongo.NewMongo(mongoClient))
			mongoRepo       = mongo.NewRepository(mongoCollectionA, mongoTransactor, false)
		)
		err := oniontx.NewSaga([]oniontx.Step{
			{
				Name:       "step_sql_0",
				Transactor: sqlTransactor,
				Action: func(ctx context.Context) error {
					err := sqlRepo.Insert(ctx, sqlTextRecord)
					assert.NoError(t, err)
					return err
				},
				Compensation: func(ctx context.Context) error {
					assert.Fail(t, "should not call (sql)")
					return nil
				},
			},
			{
				Name:       "step_mongo_0",
				Transactor: mongoTransactor,
				Action: func(ctx context.Context) error {
					err := mongoRepo.Save(ctx, mongoTestDataValA)
					assert.NoError(t, err)
					return err
				},
				Compensation: func(ctx context.Context) error {
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

	t.Run("success_success_compensation", func(t *testing.T) {
		t.Cleanup(cleanupFn)

		var (
			ctx           = context.Background()
			sqlTransactor = stdlib.NewTransactor(sqlDB)
			sqlRepo       = stdlib.NewTextRepository(sqlTransactor, false)

			mongoTransactor = mongo.NewTransactor(mongo.NewMongo(mongoClient))
			mongoRepo       = mongo.NewRepository(mongoCollectionA, mongoTransactor, false)
		)
		err := oniontx.NewSaga([]oniontx.Step{
			{
				Name:       "step_sql_0",
				Transactor: sqlTransactor,
				Action: func(ctx context.Context) error {
					err := sqlRepo.Insert(ctx, sqlTextRecord)
					assert.NoError(t, err)
					return err
				},
				Compensation: func(ctx context.Context) error {
					err := sqlRepo.Delete(ctx, sqlTextRecord)
					assert.NoError(t, err)
					return err
				},
			},
			{
				Name:       "step_mongo_0",
				Transactor: mongoTransactor,
				Action: func(ctx context.Context) error {
					err := mongoRepo.Save(ctx, mongoTestDataValA)
					assert.NoError(t, err)
					return err
				},
				Compensation: func(ctx context.Context) error {
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
				Name:       "step_error",
				Transactor: sqlTransactor,
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
			assert.Contains(t, err.Error(), mongo.ErrTextNoDocResult)
		}
	})
}
