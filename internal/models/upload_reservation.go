package models

import "time"

const (
	UploadReservationPending   = "pending"
	UploadReservationClaimed   = "claimed"
	UploadReservationCompleted = "completed"
)

type UploadReservation struct {
	ID                  string     `bson:"_id,omitempty" json:"id"`
	UserID              string     `bson:"userId" json:"userId"`
	TripID              string     `bson:"tripId" json:"tripId"`
	ClientID            string     `bson:"clientId" json:"clientId"`
	S3Key               string     `bson:"s3Key" json:"s3Key"`
	FileName            string     `bson:"fileName" json:"fileName"`
	ExpectedContentType string     `bson:"expectedContentType" json:"expectedContentType"`
	Status              string     `bson:"status" json:"status"`
	ClaimToken          string     `bson:"claimToken,omitempty" json:"-"`
	ClaimUntil          *time.Time `bson:"claimUntil,omitempty" json:"-"`
	ExpiresAt           time.Time  `bson:"expiresAt" json:"expiresAt"`
	CreatedAt           time.Time  `bson:"createdAt" json:"createdAt"`
	CompletedAt         *time.Time `bson:"completedAt,omitempty" json:"completedAt,omitempty"`
}
