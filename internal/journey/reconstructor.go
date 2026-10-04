package journey

import (
	"encoding/hex"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"travel-diary-backend/internal/models"
)

const GeneratorVersionV1 = "journey-reconstructor-v1"

type evidence struct {
	assetID      string
	takenAt      time.Time
	primaryScene string
	location     string
	emotion      string
	confidence   float64
	latitude     *float64
	longitude    *float64
	perceptual   string
	contentHash  string
}

// Reconstruct builds evidence-based moments without inventing events. It is a
// deterministic baseline that later semantic/embedding strategies can enrich.
func Reconstruct(tripID string, images []models.TripImage, analyses []models.TripImageAnalysis) Spec {
	return ReconstructWithMedia(tripID, images, analyses, nil)
}

// ReconstructWithMedia uses extracted media evidence when available while
// retaining the legacy analysis-only behavior for older trips.
func ReconstructWithMedia(tripID string, images []models.TripImage, analyses []models.TripImageAnalysis, assets []models.MediaAsset) Spec {
	analysisByImage := make(map[string]models.TripImageAnalysis, len(analyses))
	for _, analysis := range analyses {
		analysisByImage[analysis.ImageID] = analysis
	}
	assetByImage := make(map[string]models.MediaAsset, len(assets))
	for _, asset := range assets {
		assetByImage[asset.ImageID] = asset
	}

	items := make([]evidence, 0, len(images))
	unplaced := make([]string, 0)
	for _, image := range images {
		analysis, analyzed := analysisByImage[image.ID]
		if !analyzed {
			unplaced = append(unplaced, image.ID)
			continue
		}
		takenAt := evidenceTime(image, analysis)
		confidence := .55
		if analysis.TakenAt != nil || (image.EXIF != nil && image.EXIF.CapturedAt != "") {
			confidence = .85
		}
		items = append(items, evidence{
			assetID: image.ID, takenAt: takenAt, primaryScene: analysis.Scene.Primary,
			location: analysis.LocationGuess.Name, emotion: analysis.Story.EmotionalTone,
			confidence: confidence,
			latitude:   assetByImage[image.ID].Latitude, longitude: assetByImage[image.ID].Longitude,
			perceptual: assetByImage[image.ID].PerceptualHash, contentHash: assetByImage[image.ID].ContentHash,
		})
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].takenAt.Before(items[j].takenAt) })

	moments := make([]Moment, 0)
	for _, item := range items {
		if len(moments) == 0 {
			moments = append(moments, newMoment(len(moments)+1, item, []string{"journey opening"}))
			continue
		}
		current := &moments[len(moments)-1]
		boundary := boundaryEvidence(*current, item)
		if len(boundary) > 0 {
			moments = append(moments, newMoment(len(moments)+1, item, boundary))
			continue
		}
		current.AssetIDs = append(current.AssetIDs, item.assetID)
		current.EndAt = item.takenAt
		current.Confidence = (current.Confidence*float64(len(current.AssetIDs)-1) + item.confidence) / float64(len(current.AssetIDs))
	}

	return Spec{
		SchemaVersion: SchemaVersionV1, GeneratorVersion: GeneratorVersionV1,
		TripID: tripID, Moments: moments, UnplacedAssetIDs: unplaced,
	}
}

func evidenceTime(image models.TripImage, analysis models.TripImageAnalysis) time.Time {
	if analysis.TakenAt != nil && !analysis.TakenAt.IsZero() {
		return analysis.TakenAt.UTC()
	}
	if image.EXIF != nil && image.EXIF.CapturedAt != "" {
		for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05", "2006:01:02 15:04:05"} {
			if parsed, err := time.Parse(layout, image.EXIF.CapturedAt); err == nil {
				return parsed.UTC()
			}
		}
	}
	return image.CreatedAt.UTC()
}

func boundaryEvidence(current Moment, next evidence) []string {
	reasons := make([]string, 0, 3)
	gap := next.takenAt.Sub(current.EndAt)
	nearDuplicate := current.ContentHash != "" && current.ContentHash == next.contentHash
	if current.PerceptualHash != "" && next.perceptual != "" && perceptualDistance(current.PerceptualHash, next.perceptual) <= 4 {
		nearDuplicate = true
	}
	if !nearDuplicate && !sameDay(current.EndAt, next.takenAt) {
		reasons = append(reasons, "calendar day changed")
	}
	if !nearDuplicate && gap >= 12*time.Hour {
		reasons = append(reasons, "twelve-hour chronology gap")
	}
	if !nearDuplicate && gap >= time.Hour && current.PrimaryScene != "" && next.primaryScene != "" && !strings.EqualFold(current.PrimaryScene, next.primaryScene) {
		reasons = append(reasons, "scene changed after a chronology gap")
	}
	if !nearDuplicate && gap >= 30*time.Minute && current.Location != "" && next.location != "" && !strings.EqualFold(current.Location, next.location) {
		reasons = append(reasons, "location changed")
	}
	if !nearDuplicate && gap >= 20*time.Minute && geoDistanceKm(current.Latitude, current.Longitude, next.latitude, next.longitude) >= 25 {
		reasons = append(reasons, "gps distance changed")
	}
	return reasons
}

func perceptualDistance(left, right string) int {
	a, errA := hex.DecodeString(left)
	b, errB := hex.DecodeString(right)
	if errA != nil || errB != nil || len(a) != len(b) {
		return 64
	}
	distance := 0
	for i := range a {
		distance += bitsCount8(a[i] ^ b[i])
	}
	return distance
}

func bitsCount8(value byte) int {
	count := 0
	for value != 0 {
		count++
		value &= value - 1
	}
	return count
}

func geoDistanceKm(latA, lonA, latB, lonB *float64) float64 {
	if latA == nil || lonA == nil || latB == nil || lonB == nil {
		return 0
	}
	const earthRadiusKm = 6371.0
	lat1, lat2 := *latA*math.Pi/180, *latB*math.Pi/180
	dLat, dLon := lat2-lat1, (*lonB-*lonA)*math.Pi/180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1)*math.Cos(lat2)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return earthRadiusKm * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

func sameDay(left, right time.Time) bool {
	return left.Year() == right.Year() && left.YearDay() == right.YearDay()
}

func newMoment(order int, item evidence, reasons []string) Moment {
	return Moment{
		ID: fmt.Sprintf("moment-%02d", order), Order: order, Title: momentTitle(item),
		StartAt: item.takenAt, EndAt: item.takenAt, AssetIDs: []string{item.assetID},
		PrimaryScene: item.primaryScene, Location: item.location, EmotionalTone: item.emotion,
		Confidence: item.confidence, BoundaryEvidence: reasons,
		Latitude: item.latitude, Longitude: item.longitude, PerceptualHash: item.perceptual, ContentHash: item.contentHash,
	}
}

func momentTitle(item evidence) string {
	if item.location != "" && item.primaryScene != "" {
		return fmt.Sprintf("%s · %s", item.location, item.primaryScene)
	}
	if item.location != "" {
		return item.location
	}
	if item.primaryScene != "" {
		return item.primaryScene
	}
	switch hour := item.takenAt.Hour(); {
	case hour < 6:
		return "Before dawn"
	case hour < 12:
		return "Morning"
	case hour < 18:
		return "Afternoon"
	default:
		return "Evening"
	}
}
