package creative

import (
	"testing"

	"travel-diary-backend/internal/story"
)

func TestBaselineDirectionIsValidAndProducesFingerprint(t *testing.T) {
	t.Parallel()
	storySpec := story.Spec{
		TitleCandidates: []string{"After Rain"}, ClimaxAssetID: "asset-1", HeroAssetID: "asset-2",
		EmotionalArc: []story.EmotionBeat{{Name: "Opening", Description: "Mist gathers"}},
		Chapters:     []story.Chapter{{Title: "Arrival", NarrativeRole: "establishing"}},
	}
	exploration := BuildBaselineExploration(storySpec)
	direction := BuildBaselineDirection(storySpec, exploration)
	if err := ValidateDirection(direction, exploration, map[string]struct{}{"asset-1": {}, "asset-2": {}}); err != nil {
		t.Fatalf("expected valid baseline direction: %v", err)
	}
	fingerprint := FingerprintFromDirection(direction, storySpec)
	if fingerprint.CreativeMetaphor == "" || len(fingerprint.SceneSequence) != 1 || len(fingerprint.SignatureEffects) != 1 {
		t.Fatalf("unexpected fingerprint: %#v", fingerprint)
	}
}

func TestDirectionRequiresConcreteMemorableMoment(t *testing.T) {
	t.Parallel()
	storySpec := story.Spec{TitleCandidates: []string{"Journey"}, ClimaxAssetID: "asset-1"}
	exploration := BuildBaselineExploration(storySpec)
	direction := BuildBaselineDirection(storySpec, exploration)
	direction.DNA.SignatureMoment.MemoryTest = "Looks nice"
	if err := ValidateDirection(direction, exploration, map[string]struct{}{"asset-1": {}}); err == nil {
		t.Fatal("expected a weak memory test to be rejected")
	}
}

func TestBaselineDirectionAvoidsARepeatedDirection(t *testing.T) {
	t.Parallel()
	storySpec := story.Spec{
		TitleCandidates: []string{"Long Road"}, ClimaxAssetID: "asset-1",
		Chapters: []story.Chapter{{}, {}, {}, {}, {}}, EmotionalArc: []story.EmotionBeat{{}, {}, {}, {}, {}},
	}
	exploration := BuildBaselineExploration(storySpec)
	repeated := FingerprintFromConcept(exploration.Concepts[0])
	direction := BuildBaselineDirection(storySpec, exploration, repeated)
	if direction.SelectedConceptID == exploration.Concepts[0].ID {
		t.Fatal("expected the deterministic director to avoid the repeated overall direction")
	}
}
