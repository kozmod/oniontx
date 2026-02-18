package mongo

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"

	"github.com/kozmod/oniontx/test/integration/internal/entity"
)

const (
	ErrTextNoDocResult = "mongo: no documents in result"
)

func Connect(ctx context.Context, t *testing.T) *mongo.Client {
	t.Helper()
	client, err := mongo.Connect(
		options.Client().
			ApplyURI(entity.MongoConnectionString),
	)
	assert.NoError(t, err)

	err = client.Ping(ctx, readpref.Primary())
	assert.NoError(t, err)
	return client
}

// GetDataByID returns "test data"
func GetDataByID(ctx context.Context, t *testing.T, collection *mongo.Collection, id int64) (Data, error) {
	t.Helper()
	var result Data
	err := collection.FindOne(ctx, bson.M{"_id": id}).Decode(&result)
	return result, err
}
