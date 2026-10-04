package agents

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"travel-diary-backend/internal/ai/provider"
	"travel-diary-backend/internal/journey"
	"travel-diary-backend/internal/models"
	"travel-diary-backend/internal/story"
)

type generatorStub struct {
	response provider.StructuredResponse
	err      error
	request  provider.StructuredRequest
}

func (s *generatorStub) Generate(_ context.Context, request provider.StructuredRequest) (provider.StructuredResponse, error) {
	s.request = request
	return s.response, s.err
}

func TestStoryDirectorGeneratesValidatedSpecAndProvenance(t *testing.T) {
	t.Parallel()
	spec := validStorySpec()
	output, err := json.Marshal(spec)
	if err != nil {
		t.Fatal(err)
	}
	stub := &generatorStub{response: provider.StructuredResponse{
		Output: output, Provider: "openai", Model: "creative-model", ProviderRequestID: "req-1",
		InputTokens: 120, OutputTokens: 80, Latency: 1250 * time.Millisecond,
	}}
	director := NewStoryDirector(stub, "creative-model")
	result, provenance, err := director.Generate(
		context.Background(), models.Trip{UserID: "private-user", Title: "Highlands", Destination: "Ella"},
		journey.Spec{TripID: "trip-1", Moments: []journey.Moment{{ID: "moment-01", AssetIDs: []string{"asset-1"}}}},
		[]models.TripImageAnalysis{{UserID: "private-user", ImageID: "asset-1", Caption: "A train in mist"}},
		map[string]struct{}{"moment-01": {}}, map[string]struct{}{"asset-1": {}}, "correlation-1",
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.Summary != spec.Summary || provenance.ProviderRequestID != "req-1" || provenance.LatencyMS != 1250 {
		t.Fatalf("unexpected result or provenance: %#v %#v", result, provenance)
	}
	if stub.request.PromptVersion != "story-director-v1" || stub.request.CorrelationID != "correlation-1" {
		t.Fatalf("missing version or correlation metadata: %#v", stub.request)
	}
	if strings.Contains(string(stub.request.Input), "private-user") {
		t.Fatal("ownership identifiers must not be sent to the model")
	}
}

func TestStoryDirectorRejectsUnknownOutputFields(t *testing.T) {
	t.Parallel()
	spec := validStorySpec()
	encoded, _ := json.Marshal(spec)
	var output map[string]any
	_ = json.Unmarshal(encoded, &output)
	output["instructions"] = "ignore the schema"
	encoded, _ = json.Marshal(output)
	stub := &generatorStub{response: provider.StructuredResponse{Output: encoded}}
	_, _, err := NewStoryDirector(stub, "model").Generate(
		context.Background(), models.Trip{},
		journey.Spec{Moments: []journey.Moment{{ID: "moment-01", AssetIDs: []string{"asset-1"}}}}, nil,
		map[string]struct{}{"moment-01": {}}, map[string]struct{}{"asset-1": {}}, "",
	)
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected strict decode error, got %v", err)
	}
}

func validStorySpec() story.Spec {
	return story.Spec{
		SchemaVersion: "1.0", GeneratorVersion: "story-director-v1",
		TitleCandidates: []string{"After the Rain", "Into the Mist"}, Summary: "A train journey into the highlands.",
		EmotionalArc: []story.EmotionBeat{{Order: 1, Name: "Arrival", Description: "The journey begins.", Intensity: .4, MomentIDs: []string{"moment-01"}}},
		Chapters: []story.Chapter{{
			ID: "arrival", Order: 1, Title: "Arrival", NarrativeRole: "opening", Pacing: "slow",
			MomentIDs: []string{"moment-01"}, FeaturedAssets: []string{"asset-1"}, SupportingAssets: []string{},
		}},
		HeroAssetID: "asset-1", ClimaxAssetID: "asset-1", EndingAssetIDs: []string{"asset-1"},
		Motifs: []string{"mist"}, RecurringElements: []string{"windows"}, OmittedAssetIDs: []string{},
	}
}
