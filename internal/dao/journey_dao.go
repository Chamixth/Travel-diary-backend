package dao

import (
	"context"
	"time"

	"travel-diary-backend/internal/journey"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type JourneyDAO struct {
	col *mongo.Collection
}

func NewJourneyDAO(db *mongo.Database) JourneyDAO {
	col := db.Collection("journeys")
	_, _ = col.Indexes().CreateOne(context.Background(), mongo.IndexModel{
		Keys:    bson.D{{Key: "userId", Value: 1}, {Key: "tripId", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	return JourneyDAO{col: col}
}

func (d JourneyDAO) Upsert(ctx context.Context, userID, tripID string, spec journey.Spec) (journey.Record, error) {
	now := time.Now().UTC()
	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)
	var record journey.Record
	err := d.col.FindOneAndUpdate(ctx, bson.M{"userId": userID, "tripId": tripID}, bson.M{
		"$set": bson.M{"spec": spec, "updatedAt": now},
		"$setOnInsert": bson.M{
			"_id": uuid.NewString(), "userId": userID, "tripId": tripID, "createdAt": now,
		},
	}, opts).Decode(&record)
	return record, err
}

func (d JourneyDAO) FindByTripID(ctx context.Context, userID, tripID string) (journey.Record, error) {
	var record journey.Record
	err := d.col.FindOne(ctx, bson.M{"userId": userID, "tripId": tripID}).Decode(&record)
	return record, err
}

func (d JourneyDAO) DeleteByTripID(ctx context.Context, userID, tripID string) error {
	_, err := d.col.DeleteMany(ctx, bson.M{"userId": userID, "tripId": tripID})
	return err
}
