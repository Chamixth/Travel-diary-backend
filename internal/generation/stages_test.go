package generation

import (
	"testing"

	"travel-diary-backend/internal/models"
)

func TestValidateStageTransition(t *testing.T) {
	t.Parallel()

	valid := [][2]models.GenerationStage{
		{models.GenerationStageCreated, models.GenerationStageClustering},
		{models.GenerationStageClustering, models.GenerationStageStoryBuilding},
		{models.GenerationStageStoryBuilding, models.GenerationStageConceptExploration},
		{models.GenerationStageConceptExploration, models.GenerationStageCreativeDirection},
		{models.GenerationStageCreativeDirection, models.GenerationStageExperienceGeneration},
		{models.GenerationStageExperienceGeneration, models.GenerationStageValidating},
		{models.GenerationStageValidating, models.GenerationStageRenderReady},
	}
	for _, transition := range valid {
		if err := ValidateStageTransition(transition[0], transition[1]); err != nil {
			t.Fatalf("expected %s -> %s to be valid: %v", transition[0], transition[1], err)
		}
	}

	invalid := [][2]models.GenerationStage{
		{models.GenerationStageCreated, models.GenerationStageExperienceGeneration},
		{models.GenerationStageValidating, models.GenerationStageExperienceGeneration},
		{models.GenerationStageRenderReady, models.GenerationStageRenderReady},
		{"unknown", models.GenerationStageCreated},
	}
	for _, transition := range invalid {
		if err := ValidateStageTransition(transition[0], transition[1]); err == nil {
			t.Fatalf("expected %s -> %s to be invalid", transition[0], transition[1])
		}
	}
}

func TestEveryStageHasProductLanguage(t *testing.T) {
	t.Parallel()
	for stage := range stageOrder {
		if label := ProductStageLabel(stage); label == "" {
			t.Fatalf("stage %s has no product label", stage)
		}
	}
}
