package dao

import (
	"context"
	"errors"
	"time"

	"travel-diary-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var ErrUserNotFound = errors.New("user not found")

type UserDAO struct {
	col *mongo.Collection
}

func NewUserDAO(db *mongo.Database) UserDAO {
	col := db.Collection("users")
	ctx := context.Background()
	_, _ = col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "username", Value: 1}}, Options: options.Index().SetUnique(true),
	})
	// Auth0 access tokens may not contain an email claim. A sparse index allows
	// multiple users without an email while preserving uniqueness when present.
	_, _ = col.Indexes().DropOne(ctx, "email_1")
	_, _ = col.UpdateMany(ctx, bson.M{"email": ""}, bson.M{"$unset": bson.M{"email": ""}})
	_, _ = col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "email", Value: 1}}, Options: options.Index().SetUnique(true).SetSparse(true),
	})
	_, _ = col.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "googleId", Value: 1}}, Options: options.Index().SetUnique(true).SetSparse(true)},
		{Keys: bson.D{{Key: "auth0Id", Value: 1}}, Options: options.Index().SetUnique(true).SetSparse(true)},
	})
	return UserDAO{col: col}
}

func (d UserDAO) Create(ctx context.Context, u models.User) (models.User, error) {
	now := time.Now().UTC()
	u.CreatedAt = now
	u.UpdatedAt = now

	_, err := d.col.InsertOne(ctx, u)
	if err != nil {
		return models.User{}, err
	}

	return u, nil
}

func (d UserDAO) FindByUsername(ctx context.Context, username string) (models.User, error) {
	return d.findOne(ctx, "username", username)
}

func (d UserDAO) FindByEmail(ctx context.Context, email string) (models.User, error) {
	return d.findOne(ctx, "email", email)
}

func (d UserDAO) FindByID(ctx context.Context, id string) (models.User, error) {
	return d.findOne(ctx, "_id", id)
}

func (d UserDAO) FindByAuth0ID(ctx context.Context, auth0ID string) (models.User, error) {
	return d.findOne(ctx, "auth0Id", auth0ID)
}

func (d UserDAO) AssignPricingPlan(ctx context.Context, userID, planID, planSlug string) error {
	_, err := d.col.UpdateOne(ctx, bson.M{"_id": userID}, bson.M{"$set": bson.M{"pricingPlanId": planID, "pricingPlan": planSlug, "updatedAt": time.Now().UTC()}})
	return err
}

func (d UserDAO) UpsertGoogleUser(ctx context.Context, u models.User) (models.User, error) {
	now := time.Now().UTC()
	u.UpdatedAt = now

	filter := bson.M{"googleId": u.GoogleID}
	update := bson.M{
		"$set": bson.M{
			"username":     u.Username,
			"email":        u.Email,
			"googleId":     u.GoogleID,
			"auth0Id":      u.Auth0ID,
			"authProvider": u.AuthProvider,
			"name":         u.Name,
			"pictureUrl":   u.PictureURL,
			"updatedAt":    now,
		},
		"$setOnInsert": bson.M{
			"_id":           u.ID,
			"createdAt":     now,
			"pricingPlanId": "free",
			"pricingPlan":   "free",
		},
	}
	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)
	var updated models.User
	if err := d.col.FindOneAndUpdate(ctx, filter, update, opts).Decode(&updated); err != nil {
		return models.User{}, err
	}
	return updated, nil
}

func (d UserDAO) UpsertAuth0User(ctx context.Context, u models.User) (models.User, error) {
	now := time.Now().UTC()
	u.UpdatedAt = now

	filter := bson.M{"auth0Id": u.Auth0ID}
	set := bson.M{
		"username":     u.Username,
		"auth0Id":      u.Auth0ID,
		"authProvider": u.AuthProvider,
		"name":         u.Name,
		"pictureUrl":   u.PictureURL,
		"updatedAt":    now,
	}
	if u.Email != "" {
		set["email"] = u.Email
	}
	update := bson.M{
		"$set": set,
		"$setOnInsert": bson.M{
			"_id":           u.ID,
			"createdAt":     now,
			"pricingPlanId": "free",
			"pricingPlan":   "free",
		},
	}
	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)
	var updated models.User
	if err := d.col.FindOneAndUpdate(ctx, filter, update, opts).Decode(&updated); err != nil {
		return models.User{}, err
	}
	return updated, nil
}

func (d UserDAO) findOne(ctx context.Context, column, value string) (models.User, error) {
	var u models.User
	err := d.col.FindOne(ctx, bson.M{column: value}).Decode(&u)
	if err != nil {
		return models.User{}, ErrUserNotFound
	}
	return u, nil
}
