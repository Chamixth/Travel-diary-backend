package service

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rwcarlsen/goexif/exif"
	"golang.org/x/image/draw"
	"travel-diary-backend/internal/dao"
	"travel-diary-backend/internal/dto"
	"travel-diary-backend/internal/integrations"
	"travel-diary-backend/internal/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type TripImageService struct {
	trips    dao.TripDAO
	images   dao.TripImageDAO
	analyses dao.TripImageAnalysisDAO
	s3       *integrations.S3Client
	analysis *ImageAnalysisQueue

	// De-dupes the fire-and-forget thumbnail backfill (see toTripImageResponse)
	// so an old image without a thumbnail doesn't get one background job
	// spawned per page load — one attempt per image per process lifetime.
	thumbnailBackfill sync.Map
	// Bounds how many backfills actually run at once. A trip full of
	// pre-thumbnail images would otherwise fire one goroutine per image the
	// moment it's first listed — each downloading and resizing a full-size
	// original at the same time — and that CPU/network burst was starving
	// every other request on the process (including unrelated ones, since
	// JWT verification is also CPU-bound). Capacity 2 keeps the backfill
	// self-healing without reproducing that stampede.
	thumbnailSem chan struct{}
}

func NewTripImageService(trips dao.TripDAO, images dao.TripImageDAO, analyses dao.TripImageAnalysisDAO, s3 *integrations.S3Client, analysis *ImageAnalysisQueue) *TripImageService {
	return &TripImageService{
		trips: trips, images: images, analyses: analyses, s3: s3, analysis: analysis,
		thumbnailSem: make(chan struct{}, 2),
	}
}

const (
	maxImageBytes  = 15 * 1024 * 1024 // 15 MB, mirrors the frontend's limit.
	headerPeekSize = 256 * 1024       // enough to cover JPEG/PNG/GIF headers + EXIF.
	presignExpiry  = 10 * time.Minute
	viewURLExpiry  = 1 * time.Hour // how long a returned s3Url/thumbnailUrl stays fetchable by the browser.

	thumbnailMaxDimension = 480 // long-edge px — sharp enough for a 2x-density grid tile, far smaller than an original photo.
	thumbnailJPEGQuality  = 82
)

var acceptedImageTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/gif":  true,
	"image/heic": true,
	"image/heif": true,
}

// heicContentTypes narrows acceptedImageTypes down to the formats that need
// convertHEICToJPEG before anything else in this file (image.Decode, EXIF
// extraction, thumbnailing) can touch their pixels.
var heicContentTypes = map[string]bool{
	"image/heic": true,
	"image/heif": true,
}

// PresignUploads issues one presigned S3 PUT URL per requested file, so the
// browser can upload bytes directly to S3 — this backend never sees them.
// Files with an unsupported content type are rejected here, before any S3
// call; the actual size limit can only be enforced once the object exists
// (see CompleteUploads), since a presigned PUT URL can't cap body size.
func (s *TripImageService) PresignUploads(ctx context.Context, userID, tripID string, files []dto.PresignImageRequest) (dto.PresignImagesResponse, error) {
	if _, err := s.trips.FindByIDAndUserID(ctx, tripID, userID); err != nil {
		return dto.PresignImagesResponse{}, fiber.ErrNotFound
	}
	if len(files) == 0 {
		return dto.PresignImagesResponse{}, fmt.Errorf("no files provided")
	}

	resp := dto.PresignImagesResponse{
		Uploads: make([]dto.PresignedImageUpload, 0, len(files)),
	}
	for _, f := range files {
		if !acceptedImageTypes[f.ContentType] {
			resp.Failed = append(resp.Failed, dto.FailedImageUpload{
				ClientID: f.ClientID,
				FileName: f.FileName,
				Error:    "unsupported image type (use JPEG, PNG, GIF, or HEIC)",
			})
			continue
		}

		key := fmt.Sprintf("trips/%s/%s%s", tripID, uuid.NewString(), filepath.Ext(f.FileName))
		url, err := s.s3.PresignPutObject(ctx, key, f.ContentType, presignExpiry)
		if err != nil {
			return dto.PresignImagesResponse{}, err
		}
		resp.Uploads = append(resp.Uploads, dto.PresignedImageUpload{
			ClientID:    f.ClientID,
			FileName:    f.FileName,
			S3Key:       key,
			UploadURL:   url,
			ContentType: f.ContentType,
		})
	}
	return resp, nil
}

// CompleteUploads is called once the browser has PUT each file straight to
// S3. For every object it verifies size and content type directly against
// S3 (an authoritative source — never trusting what the client claims),
// reads just enough of the object to decode dimensions and EXIF, records the
// image, and queues analysis. An object that fails validation is deleted
// from S3 rather than left orphaned.
func (s *TripImageService) CompleteUploads(ctx context.Context, userID, tripID string, items []dto.CompleteImageUpload) (dto.CompleteImagesResponse, error) {
	if _, err := s.trips.FindByIDAndUserID(ctx, tripID, userID); err != nil {
		return dto.CompleteImagesResponse{}, fiber.ErrNotFound
	}
	if len(items) == 0 {
		return dto.CompleteImagesResponse{}, fmt.Errorf("no uploads to complete")
	}

	resp := dto.CompleteImagesResponse{Uploaded: make([]dto.TripImageResponse, 0, len(items))}
	for _, item := range items {
		record, err := s.completeOne(ctx, userID, tripID, item)
		if err != nil {
			resp.Failed = append(resp.Failed, dto.FailedImageUpload{
				ClientID: item.ClientID,
				FileName: item.FileName,
				S3Key:    item.S3Key,
				Error:    err.Error(),
			})
			continue
		}
		resp.Uploaded = append(resp.Uploaded, record)
	}
	return resp, nil
}

func (s *TripImageService) completeOne(ctx context.Context, userID, tripID string, item dto.CompleteImageUpload) (dto.TripImageResponse, error) {
	size, contentType, err := s.s3.HeadObject(ctx, item.S3Key)
	if err != nil {
		return dto.TripImageResponse{}, fmt.Errorf("upload did not complete")
	}
	if contentType == "" || contentType == "application/octet-stream" {
		// Some clients omit Content-Type on the PUT; fall back to sniffing
		// the object itself rather than failing a perfectly good upload.
		if prefix, err := s.s3.GetObjectPrefix(ctx, item.S3Key, 512); err == nil {
			contentType = httpDetectContentType(prefix)
		}
	}
	if !acceptedImageTypes[contentType] {
		_ = s.s3.DeleteObject(ctx, item.S3Key)
		return dto.TripImageResponse{}, fmt.Errorf("unsupported image type (use JPEG, PNG, GIF, or HEIC)")
	}
	if size <= 0 || size > maxImageBytes {
		_ = s.s3.DeleteObject(ctx, item.S3Key)
		return dto.TripImageResponse{}, fmt.Errorf("file is larger than 15 MB")
	}

	// iPhones capture HEIC by default, but it isn't renderable in most
	// non-Safari browsers and Go can't decode its HEVC-compressed pixels
	// natively — normalize to JPEG once, up front, so every step after this
	// (dimensions, EXIF, thumbnailing, the s3Url the gallery renders) just
	// deals with an ordinary JPEG like any other upload.
	var heicExif []byte
	if heicContentTypes[contentType] {
		original, err := s.s3.GetObject(ctx, item.S3Key)
		if err != nil {
			return dto.TripImageResponse{}, fmt.Errorf("could not read uploaded file")
		}
		jpegBytes, exifBytes, err := convertHEICToJPEG(ctx, original)
		if err != nil {
			_ = s.s3.DeleteObject(ctx, item.S3Key)
			return dto.TripImageResponse{}, err
		}
		newKey := strings.TrimSuffix(item.S3Key, filepath.Ext(item.S3Key)) + ".jpg"
		if err := s.s3.PutObject(ctx, newKey, "image/jpeg", jpegBytes); err != nil {
			return dto.TripImageResponse{}, fmt.Errorf("could not store converted image")
		}
		_ = s.s3.DeleteObject(ctx, item.S3Key)
		item.S3Key = newKey
		contentType = "image/jpeg"
		size = int64(len(jpegBytes))
		heicExif = exifBytes
	}

	data, err := s.s3.GetObjectPrefix(ctx, item.S3Key, headerPeekSize)
	if err != nil {
		return dto.TripImageResponse{}, fmt.Errorf("could not read uploaded file")
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		// Rare: a header that spans past our peek window. Fetch the whole
		// object once rather than failing the upload outright.
		if full, fullErr := s.s3.GetObject(ctx, item.S3Key); fullErr == nil {
			data = full
			cfg, _, err = image.DecodeConfig(bytes.NewReader(data))
		}
		if err != nil {
			_ = s.s3.DeleteObject(ctx, item.S3Key)
			return dto.TripImageResponse{}, fmt.Errorf("could not decode image")
		}
	}
	exifData := extractEXIF(data)
	if exifData == nil && heicExif != nil {
		// The re-encoded JPEG carries no EXIF of its own — heif-convert wrote
		// it to a sidecar instead, extracted from the original HEIC.
		exifData = extractEXIF(heicExif)
	}

	record, err := s.images.Create(ctx, models.TripImage{
		ID:             uuid.NewString(),
		TripID:         tripID,
		UserID:         userID,
		FileName:       item.FileName,
		ContentType:    contentType,
		FileSizeBytes:  size,
		Width:          cfg.Width,
		Height:         cfg.Height,
		DimensionName:  fmt.Sprintf("%dx%d", cfg.Width, cfg.Height),
		EXIF:           exifData,
		S3Key:          item.S3Key,
		S3URL:          s.s3.PublicURL(item.S3Key),
		AnalysisStatus: "UPLOADED",
	})
	if err != nil {
		return dto.TripImageResponse{}, err
	}
	// Generate the thumbnail in the background rather than blocking this
	// response on it: CompleteUploads processes a whole batch sequentially in
	// one request, and a synchronous download-resize-upload per file here
	// made a large batch (dozens of photos) take minutes to acknowledge.
	s.backfillThumbnail(record.ID, item.S3Key)
	if s.analysis != nil {
		if err := s.analysis.Enqueue(ctx, record); err != nil {
			return dto.TripImageResponse{}, err
		}
		record.AnalysisStatus = "QUEUED"
	}
	return s.toTripImageResponse(ctx, record), nil
}

func (s *TripImageService) List(ctx context.Context, userID, tripID string) ([]dto.TripImageResponse, error) {
	images, err := s.images.ListByTripID(ctx, userID, tripID)
	if err != nil {
		return nil, err
	}
	out := make([]dto.TripImageResponse, 0, len(images))
	for _, img := range images {
		out = append(out, s.toTripImageResponse(ctx, img))
	}
	return out, nil
}

// RetryFailedAnalysis requeues only images that previously failed analysis.
func (s *TripImageService) RetryFailedAnalysis(ctx context.Context, userID, tripID string) (int, error) {
	if _, err := s.trips.FindByIDAndUserID(ctx, tripID, userID); err != nil {
		return 0, fiber.ErrNotFound
	}
	images, err := s.images.ListByTripID(ctx, userID, tripID)
	if err != nil {
		return 0, err
	}
	if s.analysis == nil {
		return 0, fmt.Errorf("image analysis is unavailable")
	}

	retried := 0
	for _, image := range images {
		if image.AnalysisStatus != "FAILED" {
			continue
		}
		if err := s.analysis.Enqueue(ctx, image); err != nil {
			return retried, err
		}
		retried++
	}
	return retried, nil
}

func (s *TripImageService) DeleteByTripID(ctx context.Context, userID, tripID string) error {
	if _, err := s.trips.FindByIDAndUserID(ctx, tripID, userID); err != nil {
		return fiber.ErrNotFound
	}
	images, err := s.images.ListByTripID(ctx, userID, tripID)
	if err != nil {
		return err
	}
	for _, image := range images {
		if err := s.s3.DeleteObject(ctx, image.S3Key); err != nil {
			return err
		}
	}
	if err := s.analyses.DeleteByTripID(ctx, userID, tripID); err != nil {
		return err
	}
	return s.images.DeleteByTripID(ctx, userID, tripID)
}

func (s *TripImageService) Delete(ctx context.Context, userID, tripID, imageID string) error {
	if _, err := s.trips.FindByIDAndUserID(ctx, tripID, userID); err != nil {
		return fiber.ErrNotFound
	}
	image, err := s.images.FindByID(ctx, userID, imageID)
	if err != nil || image.TripID != tripID {
		return fiber.ErrNotFound
	}
	if err := s.s3.DeleteObject(ctx, image.S3Key); err != nil {
		return err
	}
	if err := s.analyses.DeleteByImageID(ctx, userID, imageID); err != nil {
		return err
	}
	return s.images.DeleteByID(ctx, userID, tripID, imageID)
}

func httpDetectContentType(data []byte) string {
	return http.DetectContentType(data)
}

// convertHEICToJPEG shells out to `heif-convert` (from libheif) rather than
// decoding HEIC in-process: the actual pixels are HEVC-compressed, and every
// way to decode that in Go — including the one pure-Go-API library available
// — is a cgo wrapper around the same native codec libraries libheif already
// ships, so there's no dependency-free alternative. Requires heif-convert on
// PATH (`brew install libheif` locally, `apt-get install libheif-examples`
// or equivalent in the deploy image).
func convertHEICToJPEG(ctx context.Context, data []byte) (jpegBytes, exifBytes []byte, err error) {
	if _, err := exec.LookPath("heif-convert"); err != nil {
		return nil, nil, fmt.Errorf("HEIC images aren't supported on this server yet")
	}

	dir, err := os.MkdirTemp("", "heic-convert-*")
	if err != nil {
		return nil, nil, fmt.Errorf("could not convert HEIC image")
	}
	defer os.RemoveAll(dir)

	inPath := filepath.Join(dir, "input.heic")
	outPath := filepath.Join(dir, "output.jpg")
	if err := os.WriteFile(inPath, data, 0o600); err != nil {
		return nil, nil, fmt.Errorf("could not convert HEIC image")
	}

	cmd := exec.CommandContext(ctx, "heif-convert", "-q", "88", "--with-exif", inPath, outPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, nil, fmt.Errorf("could not convert HEIC image: %s", strings.TrimSpace(string(out)))
	}

	jpegBytes, err = os.ReadFile(outPath)
	if err != nil {
		return nil, nil, fmt.Errorf("HEIC conversion produced no output")
	}
	exifPath := strings.TrimSuffix(outPath, filepath.Ext(outPath)) + ".exif"
	if exif, exifErr := os.ReadFile(exifPath); exifErr == nil {
		exifBytes = exif
	}
	return jpegBytes, exifBytes, nil
}

// toTripImageResponse presigns fresh GET URLs for the object (and its
// thumbnail, if any) on every call rather than trusting the unsigned,
// permanently stale URL stored on the record — the bucket is private, so a
// stored static URL always 403s.
func (s *TripImageService) toTripImageResponse(ctx context.Context, img models.TripImage) dto.TripImageResponse {
	url := img.S3URL
	if img.S3Key != "" {
		if signed, err := s.s3.PresignGetObject(ctx, img.S3Key, viewURLExpiry); err == nil {
			url = signed
		}
	}

	thumbnailURL := url
	if img.ThumbnailS3Key != "" {
		if signed, err := s.s3.PresignGetObject(ctx, img.ThumbnailS3Key, viewURLExpiry); err == nil {
			thumbnailURL = signed
		}
	} else if img.S3Key != "" {
		// Uploaded before thumbnails existed: backfill once in the
		// background rather than blocking this (or every future) page load
		// on a resize.
		s.backfillThumbnail(img.ID, img.S3Key)
	}

	return dto.TripImageResponse{
		ID:              img.ID,
		TripID:          img.TripID,
		UserID:          img.UserID,
		FileName:        img.FileName,
		ContentType:     img.ContentType,
		FileSizeBytes:   img.FileSizeBytes,
		Width:           img.Width,
		Height:          img.Height,
		DimensionName:   img.DimensionName,
		EXIF:            img.EXIF,
		S3Key:           img.S3Key,
		S3URL:           url,
		ThumbnailURL:    thumbnailURL,
		AnalysisStatus:  img.AnalysisStatus,
		AnalysisError:   img.AnalysisError,
		AnalysisVersion: img.AnalysisVersion,
		CreatedAt:       img.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       img.UpdatedAt.Format(time.RFC3339),
	}
}

// backfillThumbnail generates a thumbnail for an image uploaded before this
// feature existed. Fire-and-forget and de-duped per process (thumbnailBackfill)
// so a trip full of un-thumbnailed images doesn't spawn a resize job on every
// single page load — only the first request per image per process does.
func (s *TripImageService) backfillThumbnail(imageID, s3Key string) {
	if _, inFlight := s.thumbnailBackfill.LoadOrStore(imageID, struct{}{}); inFlight {
		return
	}
	go func() {
		s.thumbnailSem <- struct{}{}
		defer func() { <-s.thumbnailSem }()

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if thumbKey, err := s.createThumbnail(ctx, s3Key); err == nil {
			_ = s.images.SetThumbnail(ctx, imageID, thumbKey)
		}
	}()
}

// createThumbnail downloads the already-uploaded original, resizes it down
// to thumbnailMaxDimension on its long edge, and stores the result under a
// sibling "thumbs/" key — so the gallery can load a file sized in KB instead
// of an original photo that can run into the megabytes.
func (s *TripImageService) createThumbnail(ctx context.Context, s3Key string) (string, error) {
	original, err := s.s3.GetObject(ctx, s3Key)
	if err != nil {
		return "", err
	}
	thumb, err := generateThumbnailJPEG(original)
	if err != nil {
		return "", err
	}
	thumbKey := thumbnailKeyFor(s3Key)
	if err := s.s3.PutObject(ctx, thumbKey, "image/jpeg", thumb); err != nil {
		return "", err
	}
	return thumbKey, nil
}

// thumbnailKeyFor derives "trips/<id>/thumbs/<name>.jpg" from
// "trips/<id>/<name>.<ext>" — thumbnails are always encoded as JPEG
// regardless of the original's format.
func thumbnailKeyFor(key string) string {
	dir, base := filepath.Split(key)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	return fmt.Sprintf("%sthumbs/%s.jpg", dir, name)
}

func generateThumbnailJPEG(data []byte) ([]byte, error) {
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	bounds := src.Bounds()
	srcW, srcH := bounds.Dx(), bounds.Dy()
	if srcW <= 0 || srcH <= 0 {
		return nil, fmt.Errorf("invalid image bounds")
	}

	dstW, dstH := srcW, srcH
	if longest := max(srcW, srcH); longest > thumbnailMaxDimension {
		ratio := float64(thumbnailMaxDimension) / float64(longest)
		dstW = max(1, int(float64(srcW)*ratio))
		dstH = max(1, int(float64(srcH)*ratio))
	}

	dst := image.NewRGBA(image.Rect(0, 0, dstW, dstH))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, bounds, draw.Over, nil)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: thumbnailJPEGQuality}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func extractEXIF(data []byte) *models.ImageEXIF {
	metadata, err := exif.Decode(bytes.NewReader(data))
	if err != nil {
		return nil
	}

	result := &models.ImageEXIF{
		CapturedAt:   normalizeEXIFDate(exifString(metadata, exif.DateTimeOriginal)),
		CameraMake:   exifString(metadata, exif.Make),
		CameraModel:  exifString(metadata, exif.Model),
		LensModel:    exifString(metadata, exif.LensModel),
		ShutterSpeed: exifString(metadata, exif.ExposureTime),
	}
	result.ISO = exifInt(metadata, exif.ISOSpeedRatings)
	result.Aperture = exifFloat(metadata, exif.FNumber)
	result.FocalLength = exifFloat(metadata, exif.FocalLength)

	if *result == (models.ImageEXIF{}) {
		return nil
	}
	return result
}

func normalizeEXIFDate(value string) string {
	parsed, err := time.Parse("2006:01:02 15:04:05", value)
	if err != nil {
		return value
	}
	return parsed.Format("2006-01-02T15:04:05")
}

func exifString(metadata *exif.Exif, tag exif.FieldName) string {
	value, err := metadata.Get(tag)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(value.String())
}

func exifInt(metadata *exif.Exif, tag exif.FieldName) int {
	value := exifString(metadata, tag)
	parsed, _ := strconv.Atoi(value)
	return parsed
}

func exifFloat(metadata *exif.Exif, tag exif.FieldName) float64 {
	value := exifString(metadata, tag)
	if strings.Contains(value, "/") {
		parts := strings.SplitN(value, "/", 2)
		numerator, numErr := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		denominator, denErr := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if numErr == nil && denErr == nil && denominator != 0 {
			return numerator / denominator
		}
	}
	parsed, _ := strconv.ParseFloat(value, 64)
	return parsed
}
