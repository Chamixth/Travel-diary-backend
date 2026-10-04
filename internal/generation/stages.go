package generation

import (
	"fmt"

	"travel-diary-backend/internal/models"
)

var stageOrder = map[models.GenerationStage]int{
	models.GenerationStageCreated:              0,
	models.GenerationStageClustering:           1,
	models.GenerationStageStoryBuilding:        2,
	models.GenerationStageConceptExploration:   3,
	models.GenerationStageCreativeDirection:    4,
	models.GenerationStageExperienceGeneration: 5,
	models.GenerationStageValidating:           6,
	models.GenerationStageRenderReady:          7,
}

// ValidateStageTransition prevents workers and retries from moving a job
// backwards or inventing a stage that the current executor does not perform.
func ValidateStageTransition(from, to models.GenerationStage) error {
	fromOrder, fromOK := stageOrder[from]
	toOrder, toOK := stageOrder[to]
	if !fromOK {
		return fmt.Errorf("unknown generation stage %q", from)
	}
	if !toOK {
		return fmt.Errorf("unknown generation stage %q", to)
	}
	if toOrder != fromOrder+1 {
		return fmt.Errorf("invalid generation stage transition %q -> %q", from, to)
	}
	return nil
}

func ProductStageLabel(stage models.GenerationStage) string {
	switch stage {
	case models.GenerationStageCreated:
		return "Preparing your memories…"
	case models.GenerationStageClustering:
		return "Finding the moments that belong together…"
	case models.GenerationStageStoryBuilding:
		return "Discovering the story…"
	case models.GenerationStageConceptExploration:
		return "Exploring a few creative directions…"
	case models.GenerationStageCreativeDirection:
		return "Shaping the idea that belongs to this journey…"
	case models.GenerationStageExperienceGeneration:
		return "Designing your experience…"
	case models.GenerationStageValidating:
		return "Adding the finishing touches…"
	case models.GenerationStageRenderReady:
		return "Your story is ready."
	default:
		return "Working with your memories…"
	}
}
