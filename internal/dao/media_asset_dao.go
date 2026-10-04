package dao

import (
	"context"
	"time"

	"travel-diary-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MediaAssetDAO struct{ col *mongo.Collection }

func NewMediaAssetDAO(db *mongo.Database) MediaAssetDAO {
	col := db.Collection("media_assets")
	_, _ = col.Indexes().CreateMany(context.Background(), []mongo.IndexModel{
		{Keys: bson.D{{Key: "imageId", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "tripId", Value: 1}}},
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "contentHash", Value: 1}}},
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "perceptualHash", Value: 1}}},
	})
	return MediaAssetDAO{col: col}
}

func (d MediaAssetDAO) Upsert(ctx context.Context, asset models.MediaAsset) error {
	now := time.Now().UTC()
	if asset.CreatedAt.IsZero() {
		asset.CreatedAt = now
	}
	asset.UpdatedAt = now
	_, err := d.col.UpdateOne(ctx, bson.M{"imageId": asset.ImageID}, bson.M{
		"$set": bson.M{
			"imageId": asset.ImageID, "tripId": asset.TripID, "userId": asset.UserID,
			"kind": asset.Kind, "contentType": asset.ContentType, "originalS3Key": asset.OriginalS3Key,
			"width": asset.Width, "height": asset.Height, "displayWidth": asset.DisplayWidth,
			"displayHeight": asset.DisplayHeight, "orientation": asset.Orientation,
			"capturedAt": asset.CapturedAt, "latitude": asset.Latitude, "longitude": asset.Longitude,
			"contentHash": asset.ContentHash, "perceptualHash": asset.PerceptualHash,
			"dominantColors": asset.DominantColors, "quality": asset.Quality,
			"derivatives": asset.Derivatives, "metadataVersion": asset.MetadataVersion,
			"metadataStatus": asset.MetadataStatus, "metadataError": asset.MetadataError,
			"updatedAt": asset.UpdatedAt,
		},
		"$setOnInsert": bson.M{"_id": asset.ID, "createdAt": asset.CreatedAt},
	}, options.Update().SetUpsert(true))
	return err
}

func (d MediaAssetDAO) AddDerivative(ctx context.Context, imageID string, derivative models.MediaDerivative) error {
	_, err := d.col.UpdateOne(ctx, bson.M{"imageId": imageID}, bson.M{
		"$addToSet": bson.M{"derivatives": derivative},
		"$set":      bson.M{"updatedAt": time.Now().UTC()},
	})
	return err
}

func (d MediaAssetDAO) ListByTripID(ctx context.Context, userID, tripID string) ([]models.MediaAsset, error) {
	cur, err := d.col.Find(ctx, bson.M{"userId": userID, "tripId": tripID}, options.Find().SetSort(bson.D{{Key: "capturedAt", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var assets []models.MediaAsset
	for cur.Next(ctx) {
		var asset models.MediaAsset
		if err := cur.Decode(&asset); err != nil {
			return nil, err
		}
		assets = append(assets, asset)
	}
	return assets, cur.Err()
}

func (d MediaAssetDAO) DeleteByTripID(ctx context.Context, userID, tripID string) error {
	_, err := d.col.DeleteMany(ctx, bson.M{"userId": userID, "tripId": tripID})
	return err
}

func (d MediaAssetDAO) DeleteByImageID(ctx context.Context, userID, imageID string) error {
	_, err := d.col.DeleteOne(ctx, bson.M{"userId": userID, "imageId": imageID})
	return err
}
