package generation

import (
	"context"
	"errors"
	"testing"

	"travel-diary-backend/internal/journey"
	"travel-diary-backend/internal/models"
	"travel-diary-backend/internal/story"
)

type storyDirectorStub struct {
	spec       story.Spec
	provenance story.Provenance
	err        error
}

func (s storyDirectorStub) Generate(
	context.Context,
	models.Trip,
	journey.Spec,
	[]models.TripImageAnalysis,
	map[string]struct{},
	map[string]struct{},
	string,
) (story.Spec, story.Provenance, error) {
	return s.spec, s.provenance, s.err
}

func TestBuildStoryUsesModelResultWhenValid(t *testing.T) {
	t.Parallel()
	generated := story.Spec{SchemaVersion: "1.0", GeneratorVersion: "story-director-v1", Summary: "Model story"}
	provenance := story.Provenance{Provider: "openai", Model: "creative-model"}
	result, resultProvenance, err := buildStory(
		context.Background(), true, storyDirectorStub{spec: generated, provenance: provenance},
		models.Trip{}, journey.Spec{}, nil, nil, nil, "job-1",
	)
	if err != nil || result.Summary != "Model story" || resultProvenance.Provider != "openai" {
		t.Fatalf("expected model result, got %#v %#v %v", result, resultProvenance, err)
	}
}

func TestBuildStoryRecordsFallbackForMalformedModelOutput(t *testing.T) {
	t.Parallel()
	journeySpec := journey.Spec{Moments: []journey.Moment{{ID: "moment-01", Title: "Morning", AssetIDs: []string{"asset-1"}}}}
	result, provenance, err := buildStory(
		context.Background(), true, storyDirectorStub{err: errors.New("decode StorySpec: unknown field")},
		models.Trip{Title: "Fallback"}, journeySpec,
		[]models.TripImageAnalysis{{ImageID: "asset-1", Story: models.StoryAnalysis{Importance: .8}}},
		map[string]struct{}{"moment-01": {}}, map[string]struct{}{"asset-1": {}}, "job-1",
	)
	if err == nil {
		t.Fatal("expected model error to remain observable")
	}
	if result.GeneratorVersion != story.BaselineGeneratorVersion || provenance.Provider != "deterministic" {
		t.Fatalf("expected deterministic fallback, got %#v %#v", result, provenance)
	}
	if provenance.FallbackReason != "model_story_unavailable" {
		t.Fatalf("expected recorded fallback reason, got %q", provenance.FallbackReason)
	}
}
