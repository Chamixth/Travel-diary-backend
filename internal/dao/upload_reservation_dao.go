package dao

import (
	"context"
	"time"

	"travel-diary-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type UploadReservationDAO struct {
	col *mongo.Collection
}

func NewUploadReservationDAO(db *mongo.Database) UploadReservationDAO {
	col := db.Collection("upload_reservations")
	_, _ = col.Indexes().CreateMany(context.Background(), []mongo.IndexModel{
		{Keys: bson.D{{Key: "s3Key", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "expiresAt", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(0)},
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "tripId", Value: 1}, {Key: "status", Value: 1}}},
	})
	return UploadReservationDAO{col: col}
}

func (d UploadReservationDAO) Create(ctx context.Context, reservation models.UploadReservation) error {
	_, err := d.col.InsertOne(ctx, reservation)
	return err
}

func (d UploadReservationDAO) Delete(ctx context.Context, reservationID string) error {
	_, err := d.col.DeleteOne(ctx, bson.M{"_id": reservationID})
	return err
}

func (d UploadReservationDAO) Claim(
	ctx context.Context,
	userID, tripID, s3Key, claimToken string,
	claimDuration time.Duration,
) (models.UploadReservation, error) {
	now := time.Now().UTC()
	claimUntil := now.Add(claimDuration)
	filter := bson.M{
		"userId": userID, "tripId": tripID, "s3Key": s3Key, "expiresAt": bson.M{"$gt": now},
		"$or": bson.A{
			bson.M{"status": models.UploadReservationPending},
			bson.M{"status": models.UploadReservationClaimed, "claimUntil": bson.M{"$lte": now}},
		},
	}
	update := bson.M{"$set": bson.M{
		"status": models.UploadReservationClaimed, "claimToken": claimToken, "claimUntil": claimUntil,
	}}
	var reservation models.UploadReservation
	err := d.col.FindOneAndUpdate(
		ctx, filter, update, options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&reservation)
	return reservation, err
}

func (d UploadReservationDAO) Release(ctx context.Context, reservationID, claimToken string) error {
	_, err := d.col.UpdateOne(ctx, bson.M{
		"_id": reservationID, "status": models.UploadReservationClaimed, "claimToken": claimToken,
	}, bson.M{
		"$set":   bson.M{"status": models.UploadReservationPending},
		"$unset": bson.M{"claimToken": "", "claimUntil": ""},
	})
	return err
}

func (d UploadReservationDAO) Complete(ctx context.Context, reservationID, claimToken string) error {
	now := time.Now().UTC()
	_, err := d.col.UpdateOne(ctx, bson.M{
		"_id": reservationID, "status": models.UploadReservationClaimed, "claimToken": claimToken,
	}, bson.M{
		"$set":   bson.M{"status": models.UploadReservationCompleted, "completedAt": now},
		"$unset": bson.M{"claimToken": "", "claimUntil": ""},
	})
	return err
}

func (d UploadReservationDAO) DeleteByTripID(ctx context.Context, userID, tripID string) error {
	_, err := d.col.DeleteMany(ctx, bson.M{"userId": userID, "tripId": tripID})
	return err
}
