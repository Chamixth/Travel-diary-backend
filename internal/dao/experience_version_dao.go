package dao

import (
	"context"
	"fmt"
	"time"
	"travel-diary-backend/internal/experience"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ExperienceVersionDAO struct {
	versions *mongo.Collection
	heads    *mongo.Collection
}

func NewExperienceVersionDAO(db *mongo.Database) ExperienceVersionDAO {
	versions := db.Collection("experience_versions")
	heads := db.Collection("experience_heads")
	_, _ = versions.Indexes().CreateMany(context.Background(), []mongo.IndexModel{
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "tripId", Value: 1}, {Key: "version", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "tripId", Value: 1}, {Key: "createdAt", Value: -1}}},
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "tripId", Value: 1}, {Key: "sourceType", Value: 1}, {Key: "sourceId", Value: 1}}, Options: options.Index().SetUnique(true)},
	})
	_, _ = heads.Indexes().CreateOne(context.Background(), mongo.IndexModel{
		Keys: bson.D{{Key: "userId", Value: 1}, {Key: "tripId", Value: 1}}, Options: options.Index().SetUnique(true),
	})
	return ExperienceVersionDAO{versions: versions, heads: heads}
}

// CreateAndActivate inserts a new immutable snapshot, then moves the active
// pointer only if no newer writer has reserved a version in the meantime.
func (d ExperienceVersionDAO) CreateAndActivate(ctx context.Context, version experience.Version) (experience.Version, error) {
	now := time.Now().UTC()
	headID := version.UserID + ":" + version.TripID
	var head experience.Head
	err := d.heads.FindOneAndUpdate(
		ctx,
		bson.M{"_id": headID},
		bson.M{
			"$inc":         bson.M{"nextVersion": 1},
			"$set":         bson.M{"updatedAt": now},
			"$setOnInsert": bson.M{"userId": version.UserID, "tripId": version.TripID},
		},
		options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After),
	).Decode(&head)
	if err != nil {
		return experience.Version{}, err
	}

	version.Version = head.NextVersion
	version.CreatedAt = now
	if _, err := d.versions.InsertOne(ctx, version); err != nil {
		return experience.Version{}, err
	}
	result, err := d.heads.UpdateOne(
		ctx,
		bson.M{"_id": headID, "nextVersion": version.Version},
		bson.M{"$set": bson.M{"activeVersion": version.Version, "activeId": version.ID, "updatedAt": now}},
	)
	if err != nil {
		return experience.Version{}, err
	}
	if result.MatchedCount == 0 {
		return experience.Version{}, fmt.Errorf("a newer experience version superseded version %d", version.Version)
	}
	return version, nil
}

func (d ExperienceVersionDAO) FindActive(ctx context.Context, userID, tripID string) (experience.Version, error) {
	var head experience.Head
	if err := d.heads.FindOne(ctx, bson.M{"userId": userID, "tripId": tripID, "activeId": bson.M{"$ne": ""}}).Decode(&head); err != nil {
		return experience.Version{}, err
	}
	var version experience.Version
	err := d.versions.FindOne(ctx, bson.M{"_id": head.ActiveID, "userId": userID, "tripId": tripID}).Decode(&version)
	return version, err
}

func (d ExperienceVersionDAO) FindBySource(ctx context.Context, userID, tripID, sourceType, sourceID string) (experience.Version, error) {
	var version experience.Version
	err := d.versions.FindOne(ctx, bson.M{
		"userId": userID, "tripId": tripID, "sourceType": sourceType, "sourceId": sourceID,
	}, options.FindOne().SetSort(bson.D{{Key: "version", Value: -1}})).Decode(&version)
	return version, err
}

func (d ExperienceVersionDAO) ListByTripID(ctx context.Context, userID, tripID string, limit int64) ([]experience.Version, error) {
	if limit < 1 || limit > 100 {
		limit = 25
	}
	cursor, err := d.versions.Find(
		ctx,
		bson.M{"userId": userID, "tripId": tripID},
		options.Find().SetSort(bson.D{{Key: "version", Value: -1}}).SetLimit(limit),
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	versions := make([]experience.Version, 0)
	if err := cursor.All(ctx, &versions); err != nil {
		return nil, err
	}
	return versions, nil
}

func (d ExperienceVersionDAO) Activate(ctx context.Context, userID, tripID string, versionNumber int64) (experience.Version, error) {
	var version experience.Version
	if err := d.versions.FindOne(ctx, bson.M{
		"userId": userID, "tripId": tripID, "version": versionNumber,
	}).Decode(&version); err != nil {
		return experience.Version{}, err
	}
	result, err := d.heads.UpdateOne(
		ctx,
		bson.M{"userId": userID, "tripId": tripID},
		bson.M{"$set": bson.M{
			"activeVersion": version.Version, "activeId": version.ID, "updatedAt": time.Now().UTC(),
		}},
	)
	if err != nil {
		return experience.Version{}, err
	}
	if result.MatchedCount == 0 {
		return experience.Version{}, mongo.ErrNoDocuments
	}
	return version, nil
}

func (d ExperienceVersionDAO) DeleteByTripID(ctx context.Context, userID, tripID string) error {
	if _, err := d.versions.DeleteMany(ctx, bson.M{"userId": userID, "tripId": tripID}); err != nil {
		return err
	}
	_, err := d.heads.DeleteOne(ctx, bson.M{"userId": userID, "tripId": tripID})
	return err
}
