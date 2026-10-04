package generation

import (
	"context"
	"errors"
	"math"
	"time"

	"travel-diary-backend/internal/models"

	"go.mongodb.org/mongo-driver/mongo"
)

var permanentFailureCodes = map[string]struct{}{
	"invalid_stage_transition":               {},
	"story_validation_failed":                {},
	"creative_exploration_validation_failed": {},
	"creative_direction_validation_failed":   {},
	"experience_spec_validation_failed":      {},
	"experience_story_coverage_failed":       {},
}

var transientFailureCodes = map[string]struct{}{
	"job_update_failed":                       {},
	"job_completion_failed":                   {},
	"journey_images_failed":                   {},
	"journey_analyses_failed":                 {},
	"journey_persistence_failed":              {},
	"story_trip_failed":                       {},
	"story_journey_failed":                    {},
	"story_analyses_failed":                   {},
	"story_persistence_failed":                {},
	"creative_story_failed":                   {},
	"creative_journey_failed":                 {},
	"creative_analyses_failed":                {},
	"creative_fingerprints_failed":            {},
	"creative_exploration_persistence_failed": {},
	"direction_story_failed":                  {},
	"direction_exploration_failed":            {},
	"direction_analyses_failed":               {},
	"direction_fingerprints_failed":           {},
	"creative_direction_persistence_failed":   {},
	"experience_story_failed":                 {},
	"experience_direction_failed":             {},
	"experience_images_failed":                {},
	"experience_analyses_failed":              {},
	"experience_persistence_failed":           {},
	"experience_validation_failed":            {},
	"experience_direction_lookup_failed":      {},
	"legacy_album_generation_failed":          {},
}

func ClassifyFailure(code string, err error) models.GenerationFailureClass {
	if _, ok := permanentFailureCodes[code]; ok {
		return models.GenerationFailurePermanent
	}
	if errors.Is(err, context.DeadlineExceeded) || mongo.IsNetworkError(err) {
		return models.GenerationFailureTransient
	}
	if _, ok := transientFailureCodes[code]; ok {
		return models.GenerationFailureTransient
	}
	return models.GenerationFailurePermanent
}

func RetryDelay(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	seconds := 2 * math.Pow(3, float64(attempt-1))
	if seconds > 30 {
		seconds = 30
	}
	return time.Duration(seconds) * time.Second
}
