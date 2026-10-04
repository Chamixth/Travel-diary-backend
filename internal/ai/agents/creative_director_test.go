package agents

import (
	"context"
	"encoding/json"
	"testing"

	"travel-diary-backend/internal/ai/provider"
	"travel-diary-backend/internal/creative"
	"travel-diary-backend/internal/story"
)

func TestCreativeDirectorReturnsValidatedDirectionAndProvenance(t *testing.T) {
	t.Parallel()
	storySpec := story.Spec{
		TitleCandidates: []string{"After Rain"}, ClimaxAssetID: "asset-1",
		EmotionalArc: []story.EmotionBeat{{Name: "Arrival", Description: "A quiet arrival"}},
	}
	exploration := creative.BuildBaselineExploration(storySpec)
	direction := creative.BuildBaselineDirection(storySpec, exploration)
	direction.GeneratorVersion = "creative-director-v1"
	output, _ := json.Marshal(direction)
	stub := &generatorStub{response: provider.StructuredResponse{
		Output: output, Provider: "openai", Model: "creative-model", InputTokens: 200, OutputTokens: 100,
	}}

	result, provenance, err := NewCreativeDirector(stub, "creative-model").Generate(
		context.Background(), storySpec, exploration, nil, nil,
		map[string]struct{}{"asset-1": {}}, "job-1",
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.SelectedConceptID == "" || provenance.Provider != "openai" || provenance.InputTokens != 200 {
		t.Fatalf("unexpected direction result: %#v %#v", result, provenance)
	}
}

func TestCreativeDirectorRejectsUnknownSignatureAsset(t *testing.T) {
	t.Parallel()
	storySpec := story.Spec{TitleCandidates: []string{"After Rain"}, ClimaxAssetID: "asset-1"}
	exploration := creative.BuildBaselineExploration(storySpec)
	direction := creative.BuildBaselineDirection(storySpec, exploration)
	direction.GeneratorVersion = "creative-director-v1"
	output, _ := json.Marshal(direction)
	stub := &generatorStub{response: provider.StructuredResponse{Output: output, Provider: "openai", Model: "creative-model"}}

	_, _, err := NewCreativeDirector(stub, "creative-model").Generate(
		context.Background(), storySpec, exploration, nil, nil, map[string]struct{}{}, "job-1",
	)
	if err == nil {
		t.Fatal("expected unknown asset validation error")
	}
}
