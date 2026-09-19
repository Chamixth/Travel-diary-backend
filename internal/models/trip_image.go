package models

import "time"

type TripImage struct {
	ID              string     `bson:"_id,omitempty" json:"id"`
	TripID          string     `bson:"tripId" json:"tripId"`
	UserID          string     `bson:"userId" json:"userId"`
	FileName        string     `bson:"fileName" json:"fileName"`
	ContentType     string     `bson:"contentType" json:"contentType"`
	FileSizeBytes   int64      `bson:"fileSizeBytes" json:"fileSizeBytes"`
	Width           int        `bson:"width" json:"width"`
	Height          int        `bson:"height" json:"height"`
	DimensionName   string     `bson:"dimensionName" json:"dimensionName"`
	EXIF            *ImageEXIF `bson:"exif,omitempty" json:"exif,omitempty"`
	S3Key           string     `bson:"s3Key" json:"s3Key"`
	S3URL           string     `bson:"s3Url" json:"s3Url"`
	ThumbnailS3Key  string     `bson:"thumbnailS3Key,omitempty" json:"thumbnailS3Key,omitempty"`
	AnalysisStatus  string     `bson:"analysisStatus" json:"analysisStatus"`
	AnalysisError   string     `bson:"analysisError,omitempty" json:"analysisError,omitempty"`
	AnalyzedAt      *time.Time `bson:"analyzedAt,omitempty" json:"analyzedAt,omitempty"`
	AnalysisVersion int        `bson:"analysisVersion" json:"analysisVersion"`
	CreatedAt       time.Time  `bson:"createdAt" json:"createdAt"`
	UpdatedAt       time.Time  `bson:"updatedAt" json:"updatedAt"`
}

type ImageEXIF struct {
	CapturedAt   string  `bson:"capturedAt,omitempty" json:"captured_at,omitempty"`
	Timezone     string  `bson:"timezone,omitempty" json:"timezone,omitempty"`
	CameraMake   string  `bson:"cameraMake,omitempty" json:"camera_make,omitempty"`
	CameraModel  string  `bson:"cameraModel,omitempty" json:"camera_model,omitempty"`
	LensModel    string  `bson:"lensModel,omitempty" json:"lens_model,omitempty"`
	ISO          int     `bson:"iso,omitempty" json:"iso,omitempty"`
	Aperture     float64 `bson:"aperture,omitempty" json:"aperture,omitempty"`
	ShutterSpeed string  `bson:"shutterSpeed,omitempty" json:"shutter_speed,omitempty"`
	FocalLength  float64 `bson:"focalLength,omitempty" json:"focal_length,omitempty"`
}
