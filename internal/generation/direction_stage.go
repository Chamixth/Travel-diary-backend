package generation

import (
	"context"

	"travel-diary-backend/internal/creative"
	"travel-diary-backend/internal/models"
	"travel-diary-backend/internal/story"
)

type creativeDirectorPort interface {
	Generate(
		context.Context,
		story.Spec,
		creative.Exploration,
		[]models.TripImageAnalysis,
		[]creative.Fingerprint,
		map[string]struct{},
		string,
	) (creative.Direction, creative.Provenance, error)
}

func buildDirection(
	ctx context.Context,
	modelEnabled bool,
	director creativeDirectorPort,
	storySpec story.Spec,
	exploration creative.Exploration,
	analyses []models.TripImageAnalysis,
	priorFingerprints []creative.Fingerprint,
	knownAssetIDs map[string]struct{},
	correlationID string,
) (creative.Direction, creative.Provenance, error) {
	baseline := creative.BuildBaselineDirection(storySpec, exploration, priorFingerprints...)
	provenance := creative.Provenance{
		AgentID: "creative-director", PromptVersion: creative.BaselineDirectorVersion,
		Provider: "deterministic",
	}
	if !modelEnabled || director == nil {
		return baseline, provenance, nil
	}
	generated, generatedProvenance, err := director.Generate(
		ctx, storySpec, exploration, analyses, priorFingerprints, knownAssetIDs, correlationID,
	)
	if err != nil {
		provenance.FallbackReason = "model_direction_unavailable"
		return baseline, provenance, err
	}
	return generated, generatedProvenance, nil
}
