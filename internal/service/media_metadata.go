package service

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/png"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/rwcarlsen/goexif/exif"
	"travel-diary-backend/internal/models"
)

const (
	mediaMetadataSampleSize = 128
	mediaHashSize           = 32
)

// BuildMediaAssetMetadata extracts bounded, deterministic metadata from one
// normalized image. It never sends pixels to an external service.
func BuildMediaAssetMetadata(data []byte, imageID, tripID, userID, s3Key, contentType string, width, height int) (models.MediaAsset, error) {
	if len(data) == 0 {
		return models.MediaAsset{}, fmt.Errorf("image data is empty")
	}
	decoded, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return models.MediaAsset{}, fmt.Errorf("decode image metadata: %w", err)
	}
	if width <= 0 || height <= 0 {
		bounds := decoded.Bounds()
		width, height = bounds.Dx(), bounds.Dy()
	}

	exifMetadata := parseEXIFMetadata(data)
	displayWidth, displayHeight := width, height
	if isRotatedEXIF(exifMetadata.orientation) {
		displayWidth, displayHeight = height, width
	}
	contentDigest := sha256.Sum256(data)

	return models.MediaAsset{
		ID:              imageID,
		ImageID:         imageID,
		TripID:          tripID,
		UserID:          userID,
		Kind:            "photo",
		ContentType:     contentType,
		OriginalS3Key:   s3Key,
		Width:           width,
		Height:          height,
		DisplayWidth:    displayWidth,
		DisplayHeight:   displayHeight,
		Orientation:     displayOrientation(displayWidth, displayHeight),
		CapturedAt:      exifMetadata.capturedAt,
		Latitude:        exifMetadata.latitude,
		Longitude:       exifMetadata.longitude,
		ContentHash:     hex.EncodeToString(contentDigest[:]),
		PerceptualHash:  perceptualHash(decoded),
		DominantColors:  dominantColors(decoded),
		Quality:         imageQuality(decoded),
		MetadataVersion: models.MediaMetadataVersion,
		MetadataStatus:  "complete",
		Derivatives: []models.MediaDerivative{{
			Kind: "original", S3Key: s3Key, ContentType: contentType, Width: width, Height: height,
		}},
	}, nil
}

type parsedEXIF struct {
	orientation string
	capturedAt  *time.Time
	latitude    *float64
	longitude   *float64
}

func parseEXIFMetadata(data []byte) parsedEXIF {
	metadata, err := exif.Decode(bytes.NewReader(data))
	if err != nil {
		return parsedEXIF{}
	}
	result := parsedEXIF{orientation: strings.TrimSpace(exifString(metadata, exif.Orientation))}
	if value := normalizeEXIFDate(exifString(metadata, exif.DateTimeOriginal)); value != "" {
		if parsed, parseErr := time.Parse(time.RFC3339, value); parseErr == nil {
			result.capturedAt = &parsed
		}
	}
	if latitude, longitude, latLongErr := metadata.LatLong(); latLongErr == nil {
		result.latitude, result.longitude = &latitude, &longitude
	}
	return result
}

func isRotatedEXIF(orientation string) bool {
	return orientation == "5" || orientation == "6" || orientation == "7" || orientation == "8"
}

func displayOrientation(width, height int) string {
	switch {
	case width == height:
		return "square"
	case width > height:
		return "landscape"
	default:
		return "portrait"
	}
}

func perceptualHash(src image.Image) string {
	gray := resizeLuma(src, mediaHashSize, mediaHashSize)
	var total float64
	for _, value := range gray {
		total += value
	}
	average := total / float64(len(gray))
	var hash uint64
	for index, value := range gray {
		if value >= average {
			hash |= uint64(1) << uint(63-index)
		}
	}
	return fmt.Sprintf("%016x", hash)
}

func dominantColors(src image.Image) []models.DominantColor {
	bounds := src.Bounds()
	stepX := max(1, bounds.Dx()/mediaMetadataSampleSize)
	stepY := max(1, bounds.Dy()/mediaMetadataSampleSize)
	type colorBin struct {
		r, g, b uint8
	}
	counts := make(map[colorBin]int)
	total := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y += stepY {
		for x := bounds.Min.X; x < bounds.Max.X; x += stepX {
			r, g, b, _ := src.At(x, y).RGBA()
			bin := colorBin{uint8(r >> 11), uint8(g >> 11), uint8(b >> 11)}
			counts[bin]++
			total++
		}
	}
	type rankedColor struct {
		bin   colorBin
		count int
	}
	ranked := make([]rankedColor, 0, len(counts))
	for bin, count := range counts {
		ranked = append(ranked, rankedColor{bin, count})
	}
	sort.Slice(ranked, func(i, j int) bool { return ranked[i].count > ranked[j].count })
	colors := make([]models.DominantColor, 0, min(3, len(ranked)))
	for _, entry := range ranked[:min(3, len(ranked))] {
		colors = append(colors, models.DominantColor{
			Hex:        fmt.Sprintf("#%02x%02x%02x", entry.bin.r<<3, entry.bin.g<<3, entry.bin.b<<3),
			Proportion: float64(entry.count) / float64(total),
		})
	}
	return colors
}

func imageQuality(src image.Image) models.MediaQuality {
	values := resizeLuma(src, mediaMetadataSampleSize, mediaMetadataSampleSize)
	if len(values) == 0 {
		return models.MediaQuality{}
	}
	var total, contrast, edge float64
	for index, value := range values {
		total += value
		if index%mediaMetadataSampleSize != 0 {
			difference := math.Abs(value - values[index-1])
			edge += difference
		}
	}
	brightness := total / float64(len(values))
	for _, value := range values {
		contrast += math.Abs(value - brightness)
	}
	brightnessScore := 1 - math.Min(1, math.Abs(brightness-0.5)*2)
	contrastScore := math.Min(1, contrast/float64(len(values))*3)
	sharpnessScore := math.Min(1, edge/float64(len(values))*4)
	return models.MediaQuality{
		Brightness: brightness, Contrast: contrastScore, Sharpness: sharpnessScore,
		Score: (brightnessScore + contrastScore + sharpnessScore) / 3,
	}
}

func resizeLuma(src image.Image, width, height int) []float64 {
	bounds := src.Bounds()
	values := make([]float64, 0, width*height)
	for y := 0; y < height; y++ {
		sy := bounds.Min.Y + y*bounds.Dy()/height
		for x := 0; x < width; x++ {
			sx := bounds.Min.X + x*bounds.Dx()/width
			r, g, b, _ := src.At(sx, sy).RGBA()
			values = append(values, (0.2126*float64(r)+0.7152*float64(g)+0.0722*float64(b))/65535)
		}
	}
	return values
}
