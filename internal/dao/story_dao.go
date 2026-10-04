package dao

import (
	"context"
	"time"
	"travel-diary-backend/internal/story"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type StoryDAO struct{ col *mongo.Collection }

func NewStoryDAO(db *mongo.Database) StoryDAO {
	col := db.Collection("stories")
	// Foundation builds keyed stories only by source revision. Remove that
	// transitional index so a prompt/generator upgrade can create a new
	// immutable artifact for the same photographs.
	_, _ = col.Indexes().DropOne(context.Background(), "userId_1_tripId_1_sourceRevision_1")
	_, _ = col.Indexes().CreateOne(context.Background(), mongo.IndexModel{
		Keys: bson.D{
			{Key: "userId", Value: 1}, {Key: "tripId", Value: 1},
			{Key: "sourceRevision", Value: 1}, {Key: "spec.generatorVersion", Value: 1},
		}, Options: options.Index().SetUnique(true),
	})
	return StoryDAO{col: col}
}

func (d StoryDAO) CreateOrGet(ctx context.Context, userID, tripID, sourceRevision string, spec story.Spec, provenance story.Provenance) (story.Record, error) {
	now := time.Now().UTC()
	record := story.Record{
		ID: uuid.NewString(), UserID: userID, TripID: tripID, SourceRevision: sourceRevision,
		Spec: spec, Provenance: provenance, CreatedAt: now,
	}
	if _, err := d.col.InsertOne(ctx, record); err == nil {
		return record, nil
	} else if !mongo.IsDuplicateKeyError(err) {
		return story.Record{}, err
	}
	var existing story.Record
	err := d.col.FindOne(ctx, bson.M{
		"userId": userID, "tripId": tripID, "sourceRevision": sourceRevision,
		"spec.generatorVersion": spec.GeneratorVersion,
	}).Decode(&existing)
	return existing, err
}

func (d StoryDAO) DeleteByTripID(ctx context.Context, userID, tripID string) error {
	_, err := d.col.DeleteMany(ctx, bson.M{"userId": userID, "tripId": tripID})
	return err
}

func (d StoryDAO) FindLatestBySourceRevision(ctx context.Context, userID, tripID, sourceRevision string) (story.Record, error) {
	var record story.Record
	err := d.col.FindOne(ctx, bson.M{
		"userId": userID, "tripId": tripID, "sourceRevision": sourceRevision,
	}, options.FindOne().SetSort(bson.D{{Key: "createdAt", Value: -1}})).Decode(&record)
	return record, err
}
