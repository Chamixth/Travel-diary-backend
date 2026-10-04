package agents

import (
	"context"
	"encoding/json"
	"testing"

	"travel-diary-backend/internal/ai/provider"
	"travel-diary-backend/internal/creative"
	"travel-diary-backend/internal/journey"
	"travel-diary-backend/internal/story"
)

func TestCreativeExplorerReturnsDistinctValidatedConcepts(t *testing.T) {
	t.Parallel()
	exploration := creative.BuildBaselineExploration(story.Spec{TitleCandidates: []string{"Night Train"}})
	exploration.GeneratorVersion = "creative-explorer-v1"
	output, _ := json.Marshal(exploration)
	stub := &generatorStub{response: provider.StructuredResponse{Output: output, Provider: "openai", Model: "creative-model"}}
	result, provenance, err := NewCreativeExplorer(stub, "creative-model").Generate(
		context.Background(), story.Spec{}, journey.Spec{}, nil, []creative.Fingerprint{{CreativeMetaphor: "postcards"}}, "job-1",
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Concepts) != 3 || provenance.Provider != "openai" {
		t.Fatalf("unexpected exploration result: %#v %#v", result, provenance)
	}
}
