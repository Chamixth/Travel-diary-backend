package generation

import (
	"context"
	"errors"
	"testing"

	"travel-diary-backend/internal/creative"
	"travel-diary-backend/internal/journey"
	"travel-diary-backend/internal/models"
	"travel-diary-backend/internal/story"
)

type creativeExplorerStub struct{ err error }

func (s creativeExplorerStub) Generate(context.Context, story.Spec, journey.Spec, []models.TripImageAnalysis, []creative.Fingerprint, string) (creative.Exploration, creative.Provenance, error) {
	return creative.Exploration{}, creative.Provenance{}, s.err
}

func TestBuildExplorationRecordsDeterministicFallback(t *testing.T) {
	t.Parallel()
	result, provenance, err := buildExploration(
		context.Background(), true, creativeExplorerStub{err: errors.New("malformed concepts")},
		story.Spec{TitleCandidates: []string{"A Journey"}}, journey.Spec{}, nil, nil, "job-1",
	)
	if err == nil || len(result.Concepts) != 3 {
		t.Fatalf("expected observable fallback, got %#v %v", result, err)
	}
	if provenance.Provider != "deterministic" || provenance.FallbackReason != "model_exploration_unavailable" {
		t.Fatalf("unexpected fallback provenance: %#v", provenance)
	}
}
