package story

import (
	"testing"
	"time"

	"travel-diary-backend/internal/journey"
	"travel-diary-backend/internal/models"
)

func TestBuildBaselineProducesValidatedEvidenceOnlyStory(t *testing.T) {
	t.Parallel()
	trip := models.Trip{Title: "Highland Lines", Destination: "Ella"}
	journeySpec := journey.Spec{Moments: []journey.Moment{
		{ID: "moment-01", Order: 1, Title: "Kandy · Train", AssetIDs: []string{"train", "window"}},
		{ID: "moment-02", Order: 2, Title: "Ella · Mountain", AssetIDs: []string{"summit"}, StartAt: time.Now()},
	}}
	analyses := []models.TripImageAnalysis{
		{ImageID: "train", Scene: models.SceneAnalysis{Primary: "train"}, Story: models.StoryAnalysis{Importance: .6, EmotionalTone: "anticipation"}},
		{ImageID: "window", Scene: models.SceneAnalysis{Primary: "landscape"}, Story: models.StoryAnalysis{Importance: .4}},
		{ImageID: "summit", Scene: models.SceneAnalysis{Primary: "mountain"}, Story: models.StoryAnalysis{Importance: .95, EmotionalTone: "wonder"}},
	}

	spec := BuildBaseline(trip, journeySpec, analyses)
	knownMoments := map[string]struct{}{"moment-01": {}, "moment-02": {}}
	knownAssets := map[string]struct{}{"train": {}, "window": {}, "summit": {}}
	if err := Validate(spec, knownMoments, knownAssets); err != nil {
		t.Fatalf("expected valid baseline story: %v", err)
	}
	if spec.HeroAssetID != "summit" || spec.ClimaxAssetID != "summit" {
		t.Fatalf("expected summit to lead the story, got hero=%q climax=%q", spec.HeroAssetID, spec.ClimaxAssetID)
	}
	if len(spec.TitleCandidates) < 2 || spec.Chapters[0].NarrativeRole != "opening" || spec.Chapters[1].NarrativeRole != "ending" {
		t.Fatal("expected explicit titles and story arc roles")
	}
}
