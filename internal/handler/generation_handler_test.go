package handler

import (
	"strings"
	"testing"
	"time"

	"travel-diary-backend/internal/models"
)

func TestPublicGenerationFailureDoesNotExposeProviderDetails(t *testing.T) {
	t.Parallel()
	job := models.GenerationJob{
		Status: models.GenerationJobFailed, FailureCode: "legacy_album_generation_failed",
		FailureMessage: "provider rejected secret request details",
	}
	message := publicGenerationFailure(job)
	if strings.Contains(message, "provider") || strings.Contains(message, "secret") {
		t.Fatalf("public failure exposed internal details: %q", message)
	}
}

func TestGenerationResponseIncludesRetryTimingWithoutInternalError(t *testing.T) {
	t.Parallel()
	nextAttempt := time.Date(2026, time.September, 21, 10, 30, 0, 0, time.UTC)
	response := toGenerationJobResponse(models.GenerationJob{
		Status: models.GenerationJobQueued, Stage: models.GenerationStageStoryBuilding,
		Attempt: 1, MaxAttempts: 3, NextAttemptAt: &nextAttempt,
		FailureMessage: "private provider response", CreatedAt: nextAttempt, UpdatedAt: nextAttempt,
	})
	if response.MaxAttempts != 3 || response.NextAttemptAt == "" {
		t.Fatalf("expected retry metadata, got %#v", response)
	}
	if response.FailureMessage != "" {
		t.Fatal("queued retry must not expose an internal provider error")
	}
}
