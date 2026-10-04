package journey

import (
	"testing"
	"time"

	"travel-diary-backend/internal/models"
)

func TestReconstructUsesChronologyAndSceneBoundaries(t *testing.T) {
	t.Parallel()
	base := time.Date(2026, 2, 3, 8, 0, 0, 0, time.UTC)
	images := []models.TripImage{
		{ID: "train-2", CreatedAt: base.Add(40 * time.Minute)},
		{ID: "train-1", CreatedAt: base},
		{ID: "summit", CreatedAt: base.Add(6 * time.Hour)},
	}
	analyses := []models.TripImageAnalysis{
		{ImageID: "train-1", Scene: models.SceneAnalysis{Primary: "Train"}, LocationGuess: models.LocationGuess{Name: "Kandy"}},
		{ImageID: "train-2", Scene: models.SceneAnalysis{Primary: "Train"}, LocationGuess: models.LocationGuess{Name: "Kandy"}},
		{ImageID: "summit", Scene: models.SceneAnalysis{Primary: "Mountain"}, LocationGuess: models.LocationGuess{Name: "Ella"}},
	}

	spec := Reconstruct("trip-1", images, analyses)
	if len(spec.Moments) != 2 {
		t.Fatalf("expected 2 moments, got %d", len(spec.Moments))
	}
	if got := spec.Moments[0].AssetIDs; len(got) != 2 || got[0] != "train-1" || got[1] != "train-2" {
		t.Fatalf("expected chronological train moment, got %#v", got)
	}
	if spec.Moments[1].AssetIDs[0] != "summit" {
		t.Fatalf("expected summit climax, got %#v", spec.Moments[1].AssetIDs)
	}
}

func TestReconstructKeepsUnanalyzedAssetsExplicit(t *testing.T) {
	t.Parallel()
	spec := Reconstruct("trip-1", []models.TripImage{{ID: "missing"}}, nil)
	if len(spec.UnplacedAssetIDs) != 1 || spec.UnplacedAssetIDs[0] != "missing" {
		t.Fatalf("expected unplaced asset, got %#v", spec.UnplacedAssetIDs)
	}
}

func TestReconstructWithMediaKeepsNearDuplicatesTogether(t *testing.T) {
	t.Parallel()
	base := time.Date(2026, 2, 3, 23, 50, 0, 0, time.UTC)
	images := []models.TripImage{{ID: "a", CreatedAt: base}, {ID: "b", CreatedAt: base.Add(24 * time.Hour)}}
	analyses := []models.TripImageAnalysis{
		{ImageID: "a", Scene: models.SceneAnalysis{Primary: "Night"}},
		{ImageID: "b", Scene: models.SceneAnalysis{Primary: "Night"}},
	}
	assets := []models.MediaAsset{{ImageID: "a", ContentHash: "same"}, {ImageID: "b", ContentHash: "same"}}
	spec := ReconstructWithMedia("trip-1", images, analyses, assets)
	if len(spec.Moments) != 1 || len(spec.Moments[0].AssetIDs) != 2 {
		t.Fatalf("expected duplicate assets in one moment, got %#v", spec.Moments)
	}
}

func TestReconstructWithMediaUsesGPSBoundary(t *testing.T) {
	t.Parallel()
	base := time.Date(2026, 2, 3, 8, 0, 0, 0, time.UTC)
	images := []models.TripImage{{ID: "a", CreatedAt: base}, {ID: "b", CreatedAt: base.Add(time.Hour)}}
	analyses := []models.TripImageAnalysis{{ImageID: "a"}, {ImageID: "b"}}
	latA, lonA, latB, lonB := 6.9, 79.8, 7.3, 80.6
	assets := []models.MediaAsset{{ImageID: "a", Latitude: &latA, Longitude: &lonA}, {ImageID: "b", Latitude: &latB, Longitude: &lonB}}
	spec := ReconstructWithMedia("trip-1", images, analyses, assets)
	if len(spec.Moments) != 2 || len(spec.Moments[1].BoundaryEvidence) == 0 {
		t.Fatalf("expected GPS boundary evidence, got %#v", spec.Moments)
	}
}
