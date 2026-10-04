package generation

import (
	"context"
	"errors"
	"testing"

	"travel-diary-backend/internal/creative"
	"travel-diary-backend/internal/experience"
	"travel-diary-backend/internal/models"
	"travel-diary-backend/internal/story"
)

type experiencePlannerStub struct {
	spec       experience.Spec
	provenance experience.Provenance
	err        error
}

func (s experiencePlannerStub) Generate(
	context.Context, string, story.Spec, creative.Direction, []models.TripImage,
	[]models.TripImageAnalysis, experience.Registry, map[string]struct{}, string,
) (experience.Spec, experience.Provenance, error) {
	return s.spec, s.provenance, s.err
}

func TestBuildExperienceRecordsModelFallback(t *testing.T) {
	t.Parallel()
	storySpec := story.Spec{
		TitleCandidates: []string{"Journey"}, ClimaxAssetID: "asset-1",
		Chapters: []story.Chapter{{ID: "chapter-1", Title: "Arrival", FeaturedAssets: []string{"asset-1"}}},
	}
	exploration := creative.BuildBaselineExploration(storySpec)
	direction := creative.BuildBaselineDirection(storySpec, exploration)
	result, provenance, err := buildExperience(
		context.Background(), true, experiencePlannerStub{err: errors.New("invalid plan")},
		"experience-1", storySpec, direction, []models.TripImage{{ID: "asset-1"}},
		nil, map[string]struct{}{"asset-1": {}}, "job-1",
	)
	if err == nil {
		t.Fatal("expected model error to remain observable")
	}
	if result.GeneratorVersion != experience.BaselinePlannerVersion || provenance.FallbackReason != "model_experience_unavailable" {
		t.Fatalf("expected deterministic fallback, got %#v %#v", result, provenance)
	}
}
