package dao

import (
	"context"
	"time"

	"travel-diary-backend/internal/creative"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type CreativeDAO struct {
	explorations *mongo.Collection
	directions   *mongo.Collection
	fingerprints *mongo.Collection
}

func NewCreativeDAO(db *mongo.Database) CreativeDAO {
	explorations := db.Collection("creative_explorations")
	directions := db.Collection("creative_directions")
	fingerprints := db.Collection("creative_fingerprints")
	_, _ = explorations.Indexes().CreateOne(context.Background(), mongo.IndexModel{
		Keys: bson.D{
			{Key: "userId", Value: 1}, {Key: "tripId", Value: 1},
			{Key: "sourceRevision", Value: 1}, {Key: "exploration.generatorVersion", Value: 1},
		}, Options: options.Index().SetUnique(true),
	})
	_, _ = directions.Indexes().CreateOne(context.Background(), mongo.IndexModel{
		Keys: bson.D{
			{Key: "userId", Value: 1}, {Key: "tripId", Value: 1},
			{Key: "sourceRevision", Value: 1}, {Key: "direction.generatorVersion", Value: 1},
		}, Options: options.Index().SetUnique(true),
	})
	_, _ = fingerprints.Indexes().CreateMany(context.Background(), []mongo.IndexModel{
		{Keys: bson.D{{Key: "versionId", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "createdAt", Value: -1}}},
	})
	return CreativeDAO{explorations: explorations, directions: directions, fingerprints: fingerprints}
}

func (d CreativeDAO) CreateOrGetExploration(
	ctx context.Context,
	userID, tripID, sourceRevision string,
	exploration creative.Exploration,
	avoided []creative.Fingerprint,
	provenance creative.Provenance,
) (creative.Record, error) {
	record := creative.Record{
		ID: uuid.NewString(), UserID: userID, TripID: tripID, SourceRevision: sourceRevision,
		Exploration: exploration, AvoidedFingerprints: avoided, Provenance: provenance, CreatedAt: time.Now().UTC(),
	}
	if _, err := d.explorations.InsertOne(ctx, record); err == nil {
		return record, nil
	} else if !mongo.IsDuplicateKeyError(err) {
		return creative.Record{}, err
	}
	var existing creative.Record
	err := d.explorations.FindOne(ctx, bson.M{
		"userId": userID, "tripId": tripID, "sourceRevision": sourceRevision,
		"exploration.generatorVersion": exploration.GeneratorVersion,
	}).Decode(&existing)
	return existing, err
}

func (d CreativeDAO) FindLatestExplorationBySourceRevision(
	ctx context.Context, userID, tripID, sourceRevision string,
) (creative.Record, error) {
	var record creative.Record
	err := d.explorations.FindOne(ctx, bson.M{
		"userId": userID, "tripId": tripID, "sourceRevision": sourceRevision,
	}, options.FindOne().SetSort(bson.D{{Key: "createdAt", Value: -1}})).Decode(&record)
	return record, err
}

func (d CreativeDAO) CreateOrGetDirection(
	ctx context.Context,
	userID, tripID, sourceRevision, explorationID string,
	direction creative.Direction,
	fingerprint creative.Fingerprint,
	provenance creative.Provenance,
) (creative.DirectionRecord, error) {
	record := creative.DirectionRecord{
		ID: uuid.NewString(), UserID: userID, TripID: tripID, SourceRevision: sourceRevision,
		ExplorationID: explorationID, Direction: direction, Fingerprint: fingerprint,
		Provenance: provenance, CreatedAt: time.Now().UTC(),
	}
	if _, err := d.directions.InsertOne(ctx, record); err != nil {
		if !mongo.IsDuplicateKeyError(err) {
			return creative.DirectionRecord{}, err
		}
		if err := d.directions.FindOne(ctx, bson.M{
			"userId": userID, "tripId": tripID, "sourceRevision": sourceRevision,
			"direction.generatorVersion": direction.GeneratorVersion,
		}).Decode(&record); err != nil {
			return creative.DirectionRecord{}, err
		}
	}
	if err := d.ensureFingerprint(ctx, record); err != nil {
		return creative.DirectionRecord{}, err
	}
	return record, nil
}

func (d CreativeDAO) FindLatestDirectionBySourceRevision(
	ctx context.Context, userID, tripID, sourceRevision string,
) (creative.DirectionRecord, error) {
	var record creative.DirectionRecord
	err := d.directions.FindOne(ctx, bson.M{
		"userId": userID, "tripId": tripID, "sourceRevision": sourceRevision,
	}, options.FindOne().SetSort(bson.D{{Key: "createdAt", Value: -1}})).Decode(&record)
	return record, err
}

func (d CreativeDAO) ensureFingerprint(ctx context.Context, direction creative.DirectionRecord) error {
	_, err := d.fingerprints.UpdateOne(
		ctx,
		bson.M{"versionId": direction.ID},
		bson.M{"$setOnInsert": creative.FingerprintRecord{
			ID: uuid.NewString(), UserID: direction.UserID, TripID: direction.TripID,
			VersionID: direction.ID, Value: direction.Fingerprint, CreatedAt: direction.CreatedAt,
		}},
		options.Update().SetUpsert(true),
	)
	return err
}

func (d CreativeDAO) ListRecentFingerprints(ctx context.Context, userID, excludeTripID string, limit int64) ([]creative.Fingerprint, error) {
	if limit < 1 || limit > 50 {
		limit = 12
	}
	cursor, err := d.fingerprints.Find(ctx, bson.M{
		"userId": userID, "tripId": bson.M{"$ne": excludeTripID},
	}, options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetLimit(limit))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var records []creative.FingerprintRecord
	if err := cursor.All(ctx, &records); err != nil {
		return nil, err
	}
	result := make([]creative.Fingerprint, 0, len(records))
	for _, record := range records {
		result = append(result, record.Value)
	}
	return result, nil
}

func (d CreativeDAO) DeleteByTripID(ctx context.Context, userID, tripID string) error {
	if _, err := d.explorations.DeleteMany(ctx, bson.M{"userId": userID, "tripId": tripID}); err != nil {
		return err
	}
	if _, err := d.directions.DeleteMany(ctx, bson.M{"userId": userID, "tripId": tripID}); err != nil {
		return err
	}
	_, err := d.fingerprints.DeleteMany(ctx, bson.M{"userId": userID, "tripId": tripID})
	return err
}
