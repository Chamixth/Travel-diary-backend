package creative

import (
	"testing"

	"travel-diary-backend/internal/story"
)

func TestFingerprintSimilarityRewardsOverallSameness(t *testing.T) {
	t.Parallel()
	left := Fingerprint{
		CreativeMetaphor: "Moving train window", SceneSequence: []string{"cover", "window", "tunnel"},
		TypographyPersonality: "documentary", MotionLanguage: "horizontal travel",
		NavigationConcept: "rail line", PaletteClass: "mist", ImageTreatment: "window crops",
		SignatureEffects: []string{"fog reveal"}, StoryStructure: "linear journey",
	}
	right := left
	if score := Similarity(left, right); score != 1 {
		t.Fatalf("expected identical fingerprints to score 1, got %f", score)
	}

	right.CreativeMetaphor = "unsent postcards"
	right.SceneSequence = []string{"postcard", "stamp", "letter"}
	right.MotionLanguage = "page turns"
	right.NavigationConcept = "postmarks"
	right.SignatureEffects = []string{"ink bleed"}
	if score := Similarity(left, right); score >= .7 {
		t.Fatalf("expected a materially different concept below warning threshold, got %f", score)
	}
}

func TestValidateNoveltyRequiresRepeatedStrongSimilarity(t *testing.T) {
	t.Parallel()
	exploration := BuildBaselineExploration(story.Spec{TitleCandidates: []string{"Journey"}})
	candidate := FingerprintFromConcept(exploration.Concepts[0])
	if err := ValidateNovelty(exploration, []Fingerprint{candidate}); err != nil {
		t.Fatalf("one similar prior experience should be advisory: %v", err)
	}
	if err := ValidateNovelty(exploration, []Fingerprint{candidate, candidate}); err == nil {
		t.Fatal("expected repeated extreme similarity to be rejected")
	}
}
