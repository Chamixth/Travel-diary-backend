package creative

import (
	"testing"
	"travel-diary-backend/internal/story"
)

func TestBaselineExplorationContainsDistinctValidConcepts(t *testing.T) {
	t.Parallel()
	exploration := BuildBaselineExploration(story.Spec{TitleCandidates: []string{"After Rain"}, Motifs: []string{"mist", "rail"}})
	if err := ValidateExploration(exploration); err != nil {
		t.Fatalf("expected valid exploration: %v", err)
	}
	if exploration.Concepts[0].InteractionMetaphor == exploration.Concepts[1].InteractionMetaphor {
		t.Fatal("baseline concepts must be conceptually distinct")
	}
}
