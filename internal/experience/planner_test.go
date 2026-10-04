package experience

import (
	"testing"

	"travel-diary-backend/internal/models"
	"travel-diary-backend/internal/story"
)

func TestBaselinePlanCreatesRenderableResponsiveStory(t *testing.T) {
	t.Parallel()
	storySpec := story.Spec{
		TitleCandidates: []string{"After Rain"}, HeroAssetID: "asset-1", ClimaxAssetID: "asset-2",
		Chapters: []story.Chapter{
			{ID: "arrival", Title: "Arrival", NarrativeRole: "opening", FeaturedAssets: []string{"asset-1"}},
			{ID: "summit", Title: "The Summit", NarrativeRole: "climax", FeaturedAssets: []string{"asset-2"}},
		},
	}
	spec := BuildBaselinePlan(
		"experience-1", storySpec,
		ExperienceDNA{
			Title: "After Rain", CreativeThesis: "Mist reveals the journey", InteractionMetaphor: "a moving window",
			MotionLanguage: MotionLanguage{Intensity: .5}, VisualLanguage: VisualLanguage{TypographyStrategy: "documentary"},
			SignatureMoment: SignatureMoment{Concept: "The summit clears", AssetIDs: []string{"asset-2"}, MemoryTest: "The user remembers the summit clearing through mist."},
		},
		"moving-window",
		[]models.TripImage{{ID: "asset-1", Width: 1600, Height: 900}, {ID: "asset-2", Width: 900, Height: 1600}},
		[]models.TripImageAnalysis{{ImageID: "asset-1", Caption: "A train entering mist"}, {ImageID: "asset-2", Caption: "The summit above cloud"}},
	)
	knownAssets := setOf("asset-1", "asset-2")
	if err := Validate(spec, knownAssets, V1Registry()); err != nil {
		t.Fatalf("expected renderable plan: %v", err)
	}
	if err := ValidatePlanned(spec, storySpec); err != nil {
		t.Fatalf("expected story-complete plan: %v", err)
	}
	if spec.DNA.SignatureMoment.SceneID != "scene-02" {
		t.Fatalf("expected climax scene to carry signature moment, got %q", spec.DNA.SignatureMoment.SceneID)
	}
}

func TestPlannedExperienceRequiresMobileChoreography(t *testing.T) {
	t.Parallel()
	spec := validSpec()
	spec.ResponsiveRules = nil
	storySpec := story.Spec{Chapters: []story.Chapter{{ID: "climax"}}}
	if err := ValidatePlanned(spec, storySpec); err == nil {
		t.Fatal("expected missing mobile rule to fail")
	}
}

func TestBaselineCompositionChangesWithCreativeMetaphor(t *testing.T) {
	t.Parallel()
	tests := []struct {
		concept    string
		layout     string
		motion     string
		transition string
	}{
		{concept: "moving-window", layout: "horizontal-travel", motion: "horizontal-drift", transition: "wipe"},
		{concept: "unsent-notes", layout: "scattered-prints", motion: "rise-and-settle", transition: "page-turn"},
		{concept: "memory-constellation", layout: "layered-depth", motion: "depth-reveal", transition: "dissolve"},
	}
	for _, test := range tests {
		if got := baselineLayout(test.concept, 3, 0); got != test.layout {
			t.Errorf("%s layout: got %q, want %q", test.concept, got, test.layout)
		}
		if got := baselineMotion(test.concept, test.layout, "measured", .5, 0); got != test.motion {
			t.Errorf("%s motion: got %q, want %q", test.concept, got, test.motion)
		}
		if got := baselineTransition(test.concept, 1); got != test.transition {
			t.Errorf("%s transition: got %q, want %q", test.concept, got, test.transition)
		}
	}
}
