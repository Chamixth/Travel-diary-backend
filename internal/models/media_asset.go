package models

import "time"

const MediaMetadataVersion = 1

// MediaAsset is the compatibility layer for richer media intelligence. The
// original TripImage remains the upload and ownership record; this document
// can evolve as extraction models improve without rewriting that contract.
type MediaAsset struct {
	ID              string            `bson:"_id,omitempty" json:"id"`
	ImageID         string            `bson:"imageId" json:"imageId"`
	TripID          string            `bson:"tripId" json:"tripId"`
	UserID          string            `bson:"userId" json:"userId"`
	Kind            string            `bson:"kind" json:"kind"`
	ContentType     string            `bson:"contentType" json:"contentType"`
	OriginalS3Key   string            `bson:"originalS3Key" json:"originalS3Key"`
	Width           int               `bson:"width" json:"width"`
	Height          int               `bson:"height" json:"height"`
	DisplayWidth    int               `bson:"displayWidth" json:"displayWidth"`
	DisplayHeight   int               `bson:"displayHeight" json:"displayHeight"`
	Orientation     string            `bson:"orientation" json:"orientation"`
	CapturedAt      *time.Time        `bson:"capturedAt,omitempty" json:"capturedAt,omitempty"`
	Latitude        *float64          `bson:"latitude,omitempty" json:"latitude,omitempty"`
	Longitude       *float64          `bson:"longitude,omitempty" json:"longitude,omitempty"`
	ContentHash     string            `bson:"contentHash" json:"contentHash"`
	PerceptualHash  string            `bson:"perceptualHash" json:"perceptualHash"`
	DominantColors  []DominantColor   `bson:"dominantColors" json:"dominantColors"`
	Quality         MediaQuality      `bson:"quality" json:"quality"`
	Derivatives     []MediaDerivative `bson:"derivatives" json:"derivatives"`
	MetadataVersion int               `bson:"metadataVersion" json:"metadataVersion"`
	MetadataStatus  string            `bson:"metadataStatus" json:"metadataStatus"`
	MetadataError   string            `bson:"metadataError,omitempty" json:"metadataError,omitempty"`
	CreatedAt       time.Time         `bson:"createdAt" json:"createdAt"`
	UpdatedAt       time.Time         `bson:"updatedAt" json:"updatedAt"`
}

type DominantColor struct {
	Hex        string  `bson:"hex" json:"hex"`
	Proportion float64 `bson:"proportion" json:"proportion"`
}

type MediaQuality struct {
	Brightness float64 `bson:"brightness" json:"brightness"`
	Contrast   float64 `bson:"contrast" json:"contrast"`
	Sharpness  float64 `bson:"sharpness" json:"sharpness"`
	Score      float64 `bson:"score" json:"score"`
}

type MediaDerivative struct {
	Kind        string `bson:"kind" json:"kind"`
	S3Key       string `bson:"s3Key" json:"s3Key"`
	ContentType string `bson:"contentType" json:"contentType"`
	Width       int    `bson:"width" json:"width"`
	Height      int    `bson:"height" json:"height"`
}
