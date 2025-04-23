package mongo

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/kozmod/oniontx/test/integration/internal/entity"
)

type (
	// Data - test data
	Data struct {
		ID  int64  `bson:"_id"`
		Val string `bson:"value"`
	}
)

// Repository is the Mongo client wrapper.
type Repository struct {
	collection *mongo.Collection
	transactor *Transactor

	// errorExpected - need to emulate error
	errorExpected bool
}

func NewRepository(collection *mongo.Collection, transactor *Transactor, errorExpected bool) *Repository {
	return &Repository{
		collection:    collection,
		transactor:    transactor,
		errorExpected: errorExpected,
	}
}

func (r *Repository) Save(ctx context.Context, data Data) error {
	if r.errorExpected {
		return entity.ErrExpected
	}

	session, ok := r.transactor.Session(ctx)
	if !ok {
		return fmt.Errorf(`transaction does not have a session`)
	}

	if err := mongo.WithSession(ctx, session, func(ctx context.Context) error {
		if err := r.collection.FindOneAndUpdate(
			ctx,
			bson.M{"_id": data.ID},
			bson.M{"$set": data},
			options.FindOneAndUpdate().
				SetReturnDocument(options.After).
				SetUpsert(true),
		).Err(); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return fmt.Errorf("could not save data: %w", err)
	}
	return nil
}
