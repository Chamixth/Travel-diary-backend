package generation

import (
	"context"
	"testing"
	"time"

	"travel-diary-backend/internal/models"
)

func TestClassifyFailureSeparatesValidationFromInfrastructure(t *testing.T) {
	t.Parallel()
	if class := ClassifyFailure("experience_spec_validation_failed", context.DeadlineExceeded); class != models.GenerationFailurePermanent {
		t.Fatalf("schema failure must remain permanent, got %q", class)
	}
	if class := ClassifyFailure("experience_persistence_failed", context.DeadlineExceeded); class != models.GenerationFailureTransient {
		t.Fatalf("persistence timeout must be transient, got %q", class)
	}
	if class := ClassifyFailure("unknown_failure", context.Canceled); class != models.GenerationFailurePermanent {
		t.Fatalf("unknown failures must not loop automatically, got %q", class)
	}
}

func TestRetryDelayIsBoundedExponentialBackoff(t *testing.T) {
	t.Parallel()
	want := []time.Duration{2 * time.Second, 6 * time.Second, 18 * time.Second, 30 * time.Second}
	for index, expected := range want {
		if actual := RetryDelay(index + 1); actual != expected {
			t.Fatalf("attempt %d: expected %s, got %s", index+1, expected, actual)
		}
	}
}
