package generation

import (
	"context"

	"travel-diary-backend/internal/creative"
	"travel-diary-backend/internal/journey"
	"travel-diary-backend/internal/models"
	"travel-diary-backend/internal/story"
)

type creativeExplorerPort interface {
	Generate(
		context.Context,
		story.Spec,
		journey.Spec,
		[]models.TripImageAnalysis,
		[]creative.Fingerprint,
		string,
	) (creative.Exploration, creative.Provenance, error)
}

func buildExploration(
	ctx context.Context,
	modelEnabled bool,
	explorer creativeExplorerPort,
	storySpec story.Spec,
	journeySpec journey.Spec,
	analyses []models.TripImageAnalysis,
	priorFingerprints []creative.Fingerprint,
	correlationID string,
) (creative.Exploration, creative.Provenance, error) {
	baseline := creative.BuildBaselineExploration(storySpec)
	provenance := creative.Provenance{
		AgentID: "creative-explorer", PromptVersion: creative.BaselineExplorerVersion,
		Provider: "deterministic",
	}
	if !modelEnabled || explorer == nil {
		return baseline, provenance, nil
	}
	generated, generatedProvenance, err := explorer.Generate(
		ctx, storySpec, journeySpec, analyses, priorFingerprints, correlationID,
	)
	if err != nil {
		provenance.FallbackReason = "model_exploration_unavailable"
		return baseline, provenance, err
	}
	return generated, generatedProvenance, nil
}
