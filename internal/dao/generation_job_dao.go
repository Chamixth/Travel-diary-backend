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

type GenerationJobDAO struct {
	col *mongo.Collection
}

var ErrGenerationLeaseLost = errors.New("generation job worker lease was lost")

func NewGenerationJobDAO(db *mongo.Database) GenerationJobDAO {
	col := db.Collection("generation_jobs")
	_, _ = col.Indexes().CreateMany(context.Background(), []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "activeKey", Value: 1}},
			Options: options.Index().SetUnique(true).SetPartialFilterExpression(bson.M{"activeKey": bson.M{"$type": "string"}}),
		},
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "tripId", Value: 1}, {Key: "createdAt", Value: -1}}},
		{Keys: bson.D{{Key: "status", Value: 1}, {Key: "leaseUntil", Value: 1}, {Key: "createdAt", Value: 1}}},
	})
	return GenerationJobDAO{col: col}
}

func (d GenerationJobDAO) CreateOrGetActive(ctx context.Context, job models.GenerationJob) (models.GenerationJob, bool, error) {
	_, err := d.col.InsertOne(ctx, job)
	if err == nil {
		return job, true, nil
	}
	if !mongo.IsDuplicateKeyError(err) {
		return models.GenerationJob{}, false, err
	}
	var existing models.GenerationJob
	if findErr := d.col.FindOne(ctx, bson.M{"activeKey": job.ActiveKey}).Decode(&existing); findErr != nil {
		return models.GenerationJob{}, false, findErr
	}
	return existing, false, nil
}

func (d GenerationJobDAO) FindByIDAndUserID(ctx context.Context, jobID, userID, tripID string) (models.GenerationJob, error) {
	var job models.GenerationJob
	if err := d.col.FindOne(ctx, bson.M{"_id": jobID, "userId": userID, "tripId": tripID}).Decode(&job); err != nil {
		return models.GenerationJob{}, err
	}
	return job, nil
}

// ClaimNext atomically takes either a queued job or a job whose worker lease
// expired. This makes process restarts and multi-instance deployments safe.
func (d GenerationJobDAO) ClaimNext(ctx context.Context, workerID string, leaseDuration time.Duration, maxAttempts int) (models.GenerationJob, error) {
	now := time.Now().UTC()
	leaseUntil := now.Add(leaseDuration)
	filter := bson.M{"$and": bson.A{
		bson.M{"$or": bson.A{
			bson.M{"attempt": bson.M{"$lt": maxAttempts}},
			bson.M{"attempt": bson.M{"$exists": false}},
		}},
		bson.M{"$or": bson.A{
			bson.M{"status": models.GenerationJobQueued, "$or": bson.A{
				bson.M{"nextAttemptAt": bson.M{"$lte": now}},
				bson.M{"nextAttemptAt": bson.M{"$exists": false}},
			}},
			bson.M{"status": models.GenerationJobRunning, "leaseUntil": bson.M{"$lte": now}},
		}},
	}}
	update := bson.M{
		"$set": bson.M{
			"status":      models.GenerationJobRunning,
			"leaseOwner":  workerID,
			"leaseUntil":  leaseUntil,
			"maxAttempts": maxAttempts,
			"updatedAt":   now,
		},
		"$min":   bson.M{"startedAt": now},
		"$inc":   bson.M{"attempt": 1},
		"$unset": bson.M{"nextAttemptAt": ""},
	}
	opts := options.FindOneAndUpdate().
		SetSort(bson.D{{Key: "createdAt", Value: 1}}).
		SetReturnDocument(options.After)
	var job models.GenerationJob
	if err := d.col.FindOneAndUpdate(ctx, filter, update, opts).Decode(&job); err != nil {
		return models.GenerationJob{}, err
	}
	return job, nil
}

func (d GenerationJobDAO) Heartbeat(ctx context.Context, jobID, workerID string, leaseDuration time.Duration) error {
	now := time.Now().UTC()
	result, err := d.col.UpdateOne(ctx, bson.M{
		"_id": jobID, "leaseOwner": workerID, "status": models.GenerationJobRunning,
	}, bson.M{"$set": bson.M{"leaseUntil": now.Add(leaseDuration), "updatedAt": now}})
	if err != nil {
		return err
	}
	if result.ModifiedCount != 1 {
		return ErrGenerationLeaseLost
	}
	return nil
}

func (d GenerationJobDAO) Advance(ctx context.Context, jobID, workerID string, from, to models.GenerationStage) error {
	now := time.Now().UTC()
	result, err := d.col.UpdateOne(ctx, bson.M{
		"_id": jobID, "leaseOwner": workerID, "status": models.GenerationJobRunning, "stage": from,
	}, bson.M{
		"$set":  bson.M{"stage": to, "updatedAt": now},
		"$push": bson.M{"history": models.GenerationStageEvent{Stage: to, StartedAt: now}},
	})
	if err != nil {
		return err
	}
	if result.ModifiedCount != 1 {
		return ErrGenerationLeaseLost
	}
	return nil
}

func (d GenerationJobDAO) Complete(ctx context.Context, jobID, workerID string) error {
	now := time.Now().UTC()
	result, err := d.col.UpdateOne(ctx, bson.M{
		"_id": jobID, "leaseOwner": workerID, "status": models.GenerationJobRunning,
	}, bson.M{
		"$set": bson.M{
			"status": models.GenerationJobSucceeded, "stage": models.GenerationStageRenderReady,
			"finishedAt": now, "updatedAt": now,
		},
		"$unset": bson.M{"activeKey": "", "leaseOwner": "", "leaseUntil": ""},
	})
	if err != nil {
		return err
	}
	if result.ModifiedCount != 1 {
		return errors.New("generation job could not be completed")
	}
	return nil
}

func (d GenerationJobDAO) Retry(
	ctx context.Context, jobID, workerID, code, message string, nextAttemptAt time.Time,
) error {
	now := time.Now().UTC()
	result, err := d.col.UpdateOne(ctx, bson.M{
		"_id": jobID, "leaseOwner": workerID, "status": models.GenerationJobRunning,
	}, bson.M{
		"$set": bson.M{
			"status": models.GenerationJobQueued, "lastErrorCode": code,
			"failureMessage": message, "lastErrorAt": now,
			"nextAttemptAt": nextAttemptAt, "updatedAt": now,
		},
		"$unset": bson.M{"leaseOwner": "", "leaseUntil": "", "finishedAt": "", "failureCode": "", "failureClass": ""},
	})
	if err != nil {
		return err
	}
	if result.ModifiedCount != 1 {
		return ErrGenerationLeaseLost
	}
	return nil
}

// Release returns an interrupted job to the queue without consuming an
// attempt. It is used when the process is shutting down, not for failures.
func (d GenerationJobDAO) Release(ctx context.Context, jobID, workerID string) error {
	now := time.Now().UTC()
	result, err := d.col.UpdateOne(ctx, bson.M{
		"_id": jobID, "leaseOwner": workerID, "status": models.GenerationJobRunning,
	}, bson.M{
		"$set":   bson.M{"status": models.GenerationJobQueued, "nextAttemptAt": now, "updatedAt": now},
		"$inc":   bson.M{"attempt": -1},
		"$unset": bson.M{"leaseOwner": "", "leaseUntil": ""},
	})
	if err != nil {
		return err
	}
	if result.ModifiedCount != 1 {
		return ErrGenerationLeaseLost
	}
	return nil
}

func (d GenerationJobDAO) Fail(
	ctx context.Context, jobID, workerID, code, message string, class models.GenerationFailureClass,
) error {
	now := time.Now().UTC()
	result, err := d.col.UpdateOne(ctx, bson.M{
		"_id": jobID, "leaseOwner": workerID, "status": models.GenerationJobRunning,
	}, bson.M{
		"$set": bson.M{
			"status": models.GenerationJobFailed, "failureCode": code,
			"failureMessage": message, "failureClass": class,
			"lastErrorCode": code, "lastErrorAt": now,
			"finishedAt": now, "updatedAt": now,
		},
		"$unset": bson.M{"activeKey": "", "leaseOwner": "", "leaseUntil": ""},
	})
	if err != nil {
		return err
	}
	if result.ModifiedCount != 1 {
		return ErrGenerationLeaseLost
	}
	return nil
}

func (d GenerationJobDAO) FailExpiredAttempts(ctx context.Context, maxAttempts int) (int64, error) {
	now := time.Now().UTC()
	result, err := d.col.UpdateMany(ctx, bson.M{
		"status": models.GenerationJobRunning, "leaseUntil": bson.M{"$lte": now},
		"attempt": bson.M{"$gte": maxAttempts},
	}, bson.M{
		"$set": bson.M{
			"status": models.GenerationJobFailed, "failureCode": "retry_exhausted",
			"failureMessage": "worker lease expired after the final attempt",
			"failureClass":   models.GenerationFailureTransient,
			"finishedAt":     now, "updatedAt": now,
		},
		"$unset": bson.M{"activeKey": "", "leaseOwner": "", "leaseUntil": ""},
	})
	if err != nil {
		return 0, err
	}
	return result.ModifiedCount, nil
}

func (d GenerationJobDAO) DeleteByTripID(ctx context.Context, userID, tripID string) error {
	_, err := d.col.DeleteMany(ctx, bson.M{"userId": userID, "tripId": tripID})
	return err
}
