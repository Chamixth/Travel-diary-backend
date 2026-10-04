package experience

import "testing"

func validSpec() Spec {
	return Spec{
		SchemaVersion: SchemaVersionV1, GeneratorVersion: "experience-planner-v1", ID: "experience-1",
		DNA: ExperienceDNA{
			Title: "After the Rain", CreativeThesis: "A journey revealed as the fog lifts.",
			InteractionMetaphor: "train windows", MotionLanguage: MotionLanguage{Intensity: 0.4},
			SignatureMoment: SignatureMoment{Concept: "The summit clears", SceneID: "summit", AssetIDs: []string{"photo-1"}, MemoryTest: "The mountain appears as fog clears from the window."},
		},
		Scenes: []Scene{{
			ID: "summit", ChapterID: "climax", Layout: "cinematic-fullscreen", Motion: "slow-zoom",
			Assets:       []AssetPlacement{{AssetID: "photo-1", Role: "hero", Alt: "A mountain summit emerging from cloud", FocalX: .5, FocalY: .45}},
			LoadPriority: "eager",
			Fallback:     SceneFallback{Layout: "cinematic-fullscreen", Motion: "none"},
		}},
		Typography: Typography{Personality: "documentary", DisplayScale: "large", BodyMeasure: "readable"},
		Navigation: Navigation{Type: "chapter-marks", ShowProgress: true, Keyboard: true, Label: "Journey"},
	}
}

func TestValidateAcceptsKnownVersionAndRegistryEntries(t *testing.T) {
	t.Parallel()
	if err := Validate(validSpec(), setOf("photo-1"), V1Registry()); err != nil {
		t.Fatalf("expected valid spec: %v", err)
	}
}

func TestValidateRejectsUnknownAsset(t *testing.T) {
	t.Parallel()
	if err := Validate(validSpec(), setOf(), V1Registry()); err == nil {
		t.Fatal("expected an unknown asset error")
	}
}

func TestValidateRejectsUnknownPrimitive(t *testing.T) {
	t.Parallel()
	spec := validSpec()
	spec.Scenes[0].Layout = "invented-layout"
	if err := Validate(spec, setOf("photo-1"), V1Registry()); err == nil {
		t.Fatal("expected an unknown layout error")
	}
}

func TestValidateRequiresMemorableMoment(t *testing.T) {
	t.Parallel()
	spec := validSpec()
	spec.DNA.SignatureMoment.MemoryTest = ""
	if err := Validate(spec, setOf("photo-1"), V1Registry()); err == nil {
		t.Fatal("expected signature moment memory test to be required")
	}
}
