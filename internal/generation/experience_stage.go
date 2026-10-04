package generation

import (
	"context"

	"travel-diary-backend/internal/creative"
	"travel-diary-backend/internal/experience"
	"travel-diary-backend/internal/models"
	"travel-diary-backend/internal/story"
)

type experiencePlannerPort interface {
	Generate(
		context.Context,
		string,
		story.Spec,
		creative.Direction,
		[]models.TripImage,
		[]models.TripImageAnalysis,
		experience.Registry,
		map[string]struct{},
		string,
	) (experience.Spec, experience.Provenance, error)
}

func buildExperience(
	ctx context.Context,
	modelEnabled bool,
	planner experiencePlannerPort,
	experienceID string,
	storySpec story.Spec,
	direction creative.Direction,
	images []models.TripImage,
	analyses []models.TripImageAnalysis,
	knownAssetIDs map[string]struct{},
	correlationID string,
) (experience.Spec, experience.Provenance, error) {
	registry := experience.V1Registry()
	baseline := experience.BuildBaselinePlan(experienceID, storySpec, direction.DNA, direction.SelectedConceptID, images, analyses)
	provenance := experience.Provenance{
		AgentID: "experience-planner", PromptVersion: experience.BaselinePlannerVersion,
		Provider: "deterministic",
	}
	if !modelEnabled || planner == nil {
		return baseline, provenance, nil
	}
	generated, generatedProvenance, err := planner.Generate(
		ctx, experienceID, storySpec, direction, images, analyses, registry, knownAssetIDs, correlationID,
	)
	if err != nil {
		provenance.FallbackReason = "model_experience_unavailable"
		return baseline, provenance, err
	}
	return generated, generatedProvenance, nil
}
