package agents

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"travel-diary-backend/internal/ai/provider"
	"travel-diary-backend/internal/creative"
	"travel-diary-backend/internal/experience"
	"travel-diary-backend/internal/models"
	"travel-diary-backend/internal/story"
)

func TestExperiencePlannerReturnsValidatedNativeSpec(t *testing.T) {
	t.Parallel()
	storySpec, direction, images, analyses := plannerFixture()
	spec := experience.BuildBaselinePlan("experience-1", storySpec, direction.DNA, direction.SelectedConceptID, images, analyses)
	spec.GeneratorVersion = "experience-planner-v1"
	output, _ := json.Marshal(spec)
	stub := &generatorStub{response: provider.StructuredResponse{
		Output: output, Provider: "openai", Model: "creative-model", InputTokens: 500, OutputTokens: 300,
	}}
	result, provenance, err := NewExperiencePlanner(stub, "creative-model").Generate(
		context.Background(), "experience-1", storySpec, direction, images, analyses,
		experience.V1Registry(), map[string]struct{}{"asset-1": {}, "asset-2": {}}, "job-1",
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.ID != "experience-1" || result.DNA.SignatureMoment.SceneID == "" || provenance.Provider != "openai" {
		t.Fatalf("unexpected planned experience: %#v %#v", result, provenance)
	}
	if !strings.Contains(string(stub.request.Input), `"layouts"`) {
		t.Fatal("planner input must include the registered primitive catalog")
	}
}

func TestExperiencePlannerRejectsUnreservedExperienceID(t *testing.T) {
	t.Parallel()
	storySpec, direction, images, analyses := plannerFixture()
	spec := experience.BuildBaselinePlan("invented-id", storySpec, direction.DNA, direction.SelectedConceptID, images, analyses)
	spec.GeneratorVersion = "experience-planner-v1"
	output, _ := json.Marshal(spec)
	stub := &generatorStub{response: provider.StructuredResponse{Output: output}}
	_, _, err := NewExperiencePlanner(stub, "model").Generate(
		context.Background(), "reserved-id", storySpec, direction, images, analyses,
		experience.V1Registry(), map[string]struct{}{"asset-1": {}, "asset-2": {}}, "job-1",
	)
	if err == nil {
		t.Fatal("expected experience id mismatch")
	}
}

func plannerFixture() (story.Spec, creative.Direction, []models.TripImage, []models.TripImageAnalysis) {
	storySpec := story.Spec{
		TitleCandidates: []string{"After Rain"}, HeroAssetID: "asset-1", ClimaxAssetID: "asset-2",
		Chapters: []story.Chapter{
			{ID: "arrival", Title: "Arrival", NarrativeRole: "opening", FeaturedAssets: []string{"asset-1"}},
			{ID: "summit", Title: "Summit", NarrativeRole: "climax", FeaturedAssets: []string{"asset-2"}},
		},
	}
	exploration := creative.BuildBaselineExploration(storySpec)
	direction := creative.BuildBaselineDirection(storySpec, exploration)
	images := []models.TripImage{{ID: "asset-1", Width: 1600, Height: 900}, {ID: "asset-2", Width: 900, Height: 1600}}
	analyses := []models.TripImageAnalysis{{ImageID: "asset-1", Caption: "A train in mist"}, {ImageID: "asset-2", Caption: "A summit above cloud"}}
	return storySpec, direction, images, analyses
}
