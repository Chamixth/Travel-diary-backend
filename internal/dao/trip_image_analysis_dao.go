package dao

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
	"travel-diary-backend/internal/models"
)

type TripImageAnalysisDAO struct{ col *mongo.Collection }

func NewTripImageAnalysisDAO(db *mongo.Database) TripImageAnalysisDAO {
	col := db.Collection("trip_image_analysis")
	_, _ = col.Indexes().CreateMany(context.Background(), []mongo.IndexModel{
		{Keys: bson.D{{Key: "imageId", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "tripId", Value: 1}}},
	})
	return TripImageAnalysisDAO{col: col}
}
func (d TripImageAnalysisDAO) Upsert(ctx context.Context, a models.TripImageAnalysis) error {
	now := time.Now().UTC()
	_, err := d.col.UpdateOne(ctx, bson.M{"imageId": a.ImageID}, bson.M{
		"$set": bson.M{
			"imageId":         a.ImageID,
			"tripId":          a.TripID,
			"userId":          a.UserID,
			"takenAt":         a.TakenAt,
			"latitude":        a.Latitude,
			"longitude":       a.Longitude,
			"suitePlace":      a.SuitePlace,
			"orientation":     a.Orientation,
			"aspectRatio":     a.AspectRatio,
			"caption":         a.Caption,
			"scene":           a.Scene,
			"visual":          a.Visual,
			"content":         a.Content,
			"story":           a.Story,
			"quality":         a.Quality,
			"composition":     a.Composition,
			"locationGuess":   a.LocationGuess,
			"memory":          a.Memory,
			"model":           a.Model,
			"analysisVersion": a.AnalysisVersion,
			"promptVersion":   a.PromptVersion,
			"updatedAt":       now,
		},
		"$setOnInsert": bson.M{
			"_id":       a.ID,
			"createdAt": now,
		},
	}, options.Update().SetUpsert(true))
	return err
}

func (d TripImageAnalysisDAO) DeleteByTripID(ctx context.Context, userID, tripID string) error {
	_, err := d.col.DeleteMany(ctx, bson.M{"userId": userID, "tripId": tripID})
	return err
}

func (d TripImageAnalysisDAO) DeleteByImageID(ctx context.Context, userID, imageID string) error {
	_, err := d.col.DeleteOne(ctx, bson.M{"userId": userID, "imageId": imageID})
	return err
}

func (d TripImageAnalysisDAO) FindByImageID(ctx context.Context, userID, tripID, imageID string) (models.TripImageAnalysis, error) {
	var analysis models.TripImageAnalysis
	err := d.col.FindOne(ctx, bson.M{
		"userId":  userID,
		"tripId":  tripID,
		"imageId": imageID,
	}).Decode(&analysis)
	return analysis, err
}

func (d TripImageAnalysisDAO) ListByTripID(ctx context.Context, userID, tripID string) ([]models.TripImageAnalysis, error) {
	cur, err := d.col.Find(ctx, bson.M{"userId": userID, "tripId": tripID}, options.Find().SetSort(bson.D{{Key: "story.importance", Value: -1}}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var analyses []models.TripImageAnalysis
	for cur.Next(ctx) {
		var analysis models.TripImageAnalysis
		if err := cur.Decode(&analysis); err != nil {
			return nil, err
		}
		analyses = append(analyses, analysis)
	}
	return analyses, cur.Err()
}
