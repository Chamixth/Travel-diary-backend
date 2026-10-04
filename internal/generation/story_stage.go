package generation

import (
	"context"

	"travel-diary-backend/internal/journey"
	"travel-diary-backend/internal/models"
	"travel-diary-backend/internal/story"
)

type storyDirectorPort interface {
	Generate(
		ctx context.Context,
		trip models.Trip,
		journeySpec journey.Spec,
		analyses []models.TripImageAnalysis,
		knownMomentIDs map[string]struct{},
		knownAssetIDs map[string]struct{},
		correlationID string,
	) (story.Spec, story.Provenance, error)
}

func buildStory(
	ctx context.Context,
	modelEnabled bool,
	director storyDirectorPort,
	trip models.Trip,
	journeySpec journey.Spec,
	analyses []models.TripImageAnalysis,
	knownMomentIDs map[string]struct{},
	knownAssetIDs map[string]struct{},
	correlationID string,
) (story.Spec, story.Provenance, error) {
	baseline := story.BuildBaseline(trip, journeySpec, analyses)
	baselineProvenance := story.Provenance{
		AgentID: "story-director", PromptVersion: story.BaselineGeneratorVersion,
		Provider: "deterministic",
	}
	if !modelEnabled || director == nil {
		return baseline, baselineProvenance, nil
	}
	generated, provenance, err := director.Generate(
		ctx, trip, journeySpec, analyses, knownMomentIDs, knownAssetIDs, correlationID,
	)
	if err != nil {
		baselineProvenance.FallbackReason = "model_story_unavailable"
		return baseline, baselineProvenance, err
	}
	return generated, provenance, nil
}
