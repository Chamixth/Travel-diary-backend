package dto

import "travel-diary-backend/internal/models"

type TripImageResponse struct {
	ID              string            `json:"id"`
	TripID          string            `json:"tripId"`
	UserID          string            `json:"userId"`
	FileName        string            `json:"fileName"`
	ContentType     string            `json:"contentType"`
	FileSizeBytes   int64             `json:"fileSizeBytes"`
	Width           int               `json:"width"`
	Height          int               `json:"height"`
	DimensionName   string            `json:"dimensionName"`
	EXIF            *models.ImageEXIF `json:"exif,omitempty"`
	S3Key           string            `json:"s3Key"`
	S3URL           string            `json:"s3Url"`
	ThumbnailURL    string            `json:"thumbnailUrl,omitempty"`
	AnalysisStatus  string            `json:"analysisStatus"`
	AnalysisError   string            `json:"analysisError,omitempty"`
	AnalyzedAt      string            `json:"analyzedAt,omitempty"`
	AnalysisVersion int               `json:"analysisVersion"`
	CreatedAt       string            `json:"createdAt"`
	UpdatedAt       string            `json:"updatedAt"`
}

type TripImageUploadResponse struct {
	Uploaded []TripImageResponse `json:"uploaded"`
}

// FailedImageUpload reports a per-file failure. ClientID and S3Key are
// included whenever available so the frontend can correlate a failure back
// to the right file without relying on FileName, which two different files
// in the same batch could share.
type FailedImageUpload struct {
	FileName string `json:"fileName"`
	Error    string `json:"error"`
	ClientID string `json:"clientId,omitempty"`
	S3Key    string `json:"s3Key,omitempty"`
}

// PresignImagesRequest lists the files the client wants to upload, so the
// backend can hand back one presigned S3 PUT URL per file without ever
// receiving the file bytes itself. ClientID is an opaque id the frontend
// generates to correlate a presigned upload back to its File object.
type PresignImagesRequest struct {
	Files []PresignImageRequest `json:"files"`
}

type PresignImageRequest struct {
	ClientID    string `json:"clientId"`
	FileName    string `json:"fileName"`
	ContentType string `json:"contentType"`
}

type PresignedImageUpload struct {
	ClientID    string `json:"clientId"`
	FileName    string `json:"fileName"`
	S3Key       string `json:"s3Key"`
	UploadURL   string `json:"uploadUrl"`
	ContentType string `json:"contentType"`
}

type PresignImagesResponse struct {
	Uploads []PresignedImageUpload `json:"uploads"`
	Failed  []FailedImageUpload    `json:"failed,omitempty"`
}

// CompleteImagesRequest tells the backend which presigned uploads actually
// landed in S3, so it can verify each object directly against S3 (never
// trusting client-reported size/type), record it, and queue analysis.
type CompleteImagesRequest struct {
	Images []CompleteImageUpload `json:"images"`
}

type CompleteImageUpload struct {
	ClientID string `json:"clientId"`
	S3Key    string `json:"s3Key"`
	FileName string `json:"fileName"`
}

type CompleteImagesResponse struct {
	Uploaded []TripImageResponse `json:"uploaded"`
	Failed   []FailedImageUpload `json:"failed,omitempty"`
}
