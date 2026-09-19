package dao

import (
	"context"
	"time"

	"travel-diary-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type TripImageDAO struct {
	col *mongo.Collection
}

func (d TripImageDAO) FindByID(ctx context.Context, userID, imageID string) (models.TripImage, error) {
	var image models.TripImage
	if err := d.col.FindOne(ctx, bson.M{"_id": imageID, "userId": userID}).Decode(&image); err != nil {
		return models.TripImage{}, mongo.ErrNoDocuments
	}
	return image, nil
}

func (d TripImageDAO) SetThumbnail(ctx context.Context, imageID, thumbnailS3Key string) error {
	_, err := d.col.UpdateOne(ctx, bson.M{"_id": imageID},
		bson.M{"$set": bson.M{"thumbnailS3Key": thumbnailS3Key, "updatedAt": time.Now().UTC()}})
	return err
}

func (d TripImageDAO) SetAnalysisState(ctx context.Context, imageID, status, analysisError string, analyzedAt *time.Time) error {
	update := bson.M{"analysisStatus": status, "analysisError": analysisError, "updatedAt": time.Now().UTC()}
	if analyzedAt != nil {
		update["analyzedAt"] = analyzedAt
	}
	_, err := d.col.UpdateOne(ctx, bson.M{"_id": imageID}, bson.M{"$set": update})
	return err
}

func (d TripImageDAO) CountByStatus(ctx context.Context, userID, tripID string) (map[string]int64, int64, error) {
	cur, err := d.col.Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"userId": userID, "tripId": tripID}}},
		{{Key: "$group", Value: bson.M{"_id": "$analysisStatus", "count": bson.M{"$sum": 1}}}},
	})
	if err != nil {
		return nil, 0, err
	}
	defer cur.Close(ctx)
	counts := make(map[string]int64)
	var total int64
	for cur.Next(ctx) {
		var row struct {
			ID    string `bson:"_id"`
			Count int64  `bson:"count"`
		}
		if err := cur.Decode(&row); err != nil {
			return nil, 0, err
		}
		counts[row.ID] = row.Count
		total += row.Count
	}
	return counts, total, cur.Err()
}

func NewTripImageDAO(db *mongo.Database) TripImageDAO {
	col := db.Collection("trip_images")
	_, _ = col.Indexes().CreateMany(context.Background(), []mongo.IndexModel{
		{Keys: bson.D{{Key: "tripId", Value: 1}, {Key: "createdAt", Value: -1}}},
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "createdAt", Value: -1}}},
	})
	return TripImageDAO{col: col}
}

func (d TripImageDAO) Create(ctx context.Context, image models.TripImage) (models.TripImage, error) {
	now := time.Now().UTC()
	image.CreatedAt = now
	image.UpdatedAt = now
	_, err := d.col.InsertOne(ctx, image)
	if err != nil {
		return models.TripImage{}, err
	}
	return image, nil
}

func (d TripImageDAO) ListByTripID(ctx context.Context, userID, tripID string) ([]models.TripImage, error) {
	cur, err := d.col.Find(ctx, bson.M{"userId": userID, "tripId": tripID}, options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var images []models.TripImage
	for cur.Next(ctx) {
		var img models.TripImage
		if err := cur.Decode(&img); err != nil {
			return nil, err
		}
		images = append(images, img)
	}
	return images, cur.Err()
}

func (d TripImageDAO) DeleteByTripID(ctx context.Context, userID, tripID string) error {
	_, err := d.col.DeleteMany(ctx, bson.M{"userId": userID, "tripId": tripID})
	return err
}

func (d TripImageDAO) DeleteByID(ctx context.Context, userID, tripID, imageID string) error {
	_, err := d.col.DeleteOne(ctx, bson.M{"_id": imageID, "userId": userID, "tripId": tripID})
	return err
}
