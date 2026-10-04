package generation

import (
	"context"
	"errors"
	"testing"

	"travel-diary-backend/internal/creative"
	"travel-diary-backend/internal/models"
	"travel-diary-backend/internal/story"
)

type creativeDirectorStub struct {
	direction  creative.Direction
	provenance creative.Provenance
	err        error
}

func (s creativeDirectorStub) Generate(
	context.Context, story.Spec, creative.Exploration, []models.TripImageAnalysis,
	[]creative.Fingerprint, map[string]struct{}, string,
) (creative.Direction, creative.Provenance, error) {
	return s.direction, s.provenance, s.err
}

func TestBuildDirectionUsesModelResult(t *testing.T) {
	t.Parallel()
	generated := creative.Direction{SchemaVersion: "1.0", GeneratorVersion: "creative-director-v1", SelectedConceptID: "concept-1"}
	provenance := creative.Provenance{Provider: "openai", Model: "creative-model"}
	result, resultProvenance, err := buildDirection(
		context.Background(), true, creativeDirectorStub{direction: generated, provenance: provenance},
		story.Spec{}, creative.Exploration{}, nil, nil, nil, "job-1",
	)
	if err != nil || result.SelectedConceptID != "concept-1" || resultProvenance.Provider != "openai" {
		t.Fatalf("expected model result, got %#v %#v %v", result, resultProvenance, err)
	}
}

func TestBuildDirectionRecordsDeterministicFallback(t *testing.T) {
	t.Parallel()
	storySpec := story.Spec{TitleCandidates: []string{"Night Train"}, ClimaxAssetID: "asset-1"}
	exploration := creative.BuildBaselineExploration(storySpec)
	result, provenance, err := buildDirection(
		context.Background(), true, creativeDirectorStub{err: errors.New("malformed direction")},
		storySpec, exploration, nil, nil, map[string]struct{}{"asset-1": {}}, "job-1",
	)
	if err == nil {
		t.Fatal("expected model error to remain observable")
	}
	if result.GeneratorVersion != creative.BaselineDirectorVersion || provenance.Provider != "deterministic" {
		t.Fatalf("expected deterministic fallback, got %#v %#v", result, provenance)
	}
	if provenance.FallbackReason != "model_direction_unavailable" {
		t.Fatalf("expected fallback reason, got %q", provenance.FallbackReason)
	}
}
