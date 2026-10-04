package experience

import (
	"errors"
	"fmt"
	"strings"
)

const SchemaVersionV1 = "1.0"

type Spec struct {
	SchemaVersion    string           `bson:"schemaVersion" json:"schemaVersion"`
	GeneratorVersion string           `bson:"generatorVersion" json:"generatorVersion"`
	ID               string           `bson:"id" json:"id"`
	DNA              ExperienceDNA    `bson:"dna" json:"dna"`
	Scenes           []Scene          `bson:"scenes" json:"scenes"`
	Transitions      []Transition     `bson:"transitions" json:"transitions"`
	GlobalEffects    []Effect         `bson:"globalEffects" json:"globalEffects"`
	Typography       Typography       `bson:"typography" json:"typography"`
	Navigation       Navigation       `bson:"navigation" json:"navigation"`
	ResponsiveRules  []ResponsiveRule `bson:"responsiveRules" json:"responsiveRules"`
}

type ExperienceDNA struct {
	Title               string          `bson:"title" json:"title"`
	Subtitle            string          `bson:"subtitle,omitempty" json:"subtitle,omitempty"`
	CreativeThesis      string          `bson:"creativeThesis" json:"creativeThesis"`
	InteractionMetaphor string          `bson:"interactionMetaphor" json:"interactionMetaphor"`
	EmotionalArc        EmotionalArc    `bson:"emotionalArc" json:"emotionalArc"`
	VisualLanguage      VisualLanguage  `bson:"visualLanguage" json:"visualLanguage"`
	MotionLanguage      MotionLanguage  `bson:"motionLanguage" json:"motionLanguage"`
	NavigationConcept   string          `bson:"navigationConcept" json:"navigationConcept"`
	SignatureMoment     SignatureMoment `bson:"signatureMoment" json:"signatureMoment"`
	Constraints         Constraints     `bson:"constraints" json:"constraints"`
}

type EmotionalArc struct {
	Opening     string `bson:"opening" json:"opening"`
	Development string `bson:"development" json:"development"`
	Climax      string `bson:"climax" json:"climax"`
	Ending      string `bson:"ending" json:"ending"`
}

type VisualLanguage struct {
	Personality        []string `bson:"personality" json:"personality"`
	Atmosphere         []string `bson:"atmosphere" json:"atmosphere"`
	PaletteStrategy    string   `bson:"paletteStrategy" json:"paletteStrategy"`
	TextureStrategy    string   `bson:"textureStrategy,omitempty" json:"textureStrategy,omitempty"`
	TypographyStrategy string   `bson:"typographyStrategy" json:"typographyStrategy"`
	ImageTreatment     string   `bson:"imageTreatment" json:"imageTreatment"`
}

type MotionLanguage struct {
	Personality    string  `bson:"personality" json:"personality"`
	Intensity      float64 `bson:"intensity" json:"intensity"`
	Pacing         string  `bson:"pacing" json:"pacing"`
	Physics        string  `bson:"physics" json:"physics"`
	ScrollBehavior string  `bson:"scrollBehavior" json:"scrollBehavior"`
}

type SignatureMoment struct {
	Concept    string   `bson:"concept" json:"concept"`
	SceneID    string   `bson:"sceneId,omitempty" json:"sceneId,omitempty"`
	AssetIDs   []string `bson:"assetIds" json:"assetIds"`
	MemoryTest string   `bson:"memoryTest" json:"memoryTest"`
}

type Constraints struct {
	Avoid              []string `bson:"avoid" json:"avoid"`
	AccessibilityNotes []string `bson:"accessibilityNotes" json:"accessibilityNotes"`
	PerformanceNotes   []string `bson:"performanceNotes" json:"performanceNotes"`
}

type Scene struct {
	ID           string           `bson:"id" json:"id"`
	ChapterID    string           `bson:"chapterId" json:"chapterId"`
	Layout       string           `bson:"layout" json:"layout"`
	Motion       string           `bson:"motion" json:"motion"`
	Assets       []AssetPlacement `bson:"assets" json:"assets"`
	Content      SceneContent     `bson:"content" json:"content"`
	Effects      []Effect         `bson:"effects" json:"effects"`
	LoadPriority string           `bson:"loadPriority" json:"loadPriority"`
	Fallback     SceneFallback    `bson:"fallback" json:"fallback"`
}

type AssetPlacement struct {
	AssetID string  `bson:"assetId" json:"assetId"`
	Role    string  `bson:"role" json:"role"`
	Alt     string  `bson:"alt" json:"alt"`
	FocalX  float64 `bson:"focalX" json:"focalX"`
	FocalY  float64 `bson:"focalY" json:"focalY"`
}

type SceneContent struct {
	Eyebrow string `bson:"eyebrow,omitempty" json:"eyebrow,omitempty"`
	Title   string `bson:"title,omitempty" json:"title,omitempty"`
	Body    string `bson:"body,omitempty" json:"body,omitempty"`
	Caption string `bson:"caption,omitempty" json:"caption,omitempty"`
}

type SceneFallback struct {
	Layout string `bson:"layout" json:"layout"`
	Motion string `bson:"motion" json:"motion"`
}

type Effect struct {
	ID       string         `bson:"id" json:"id"`
	Version  string         `bson:"version" json:"version"`
	Params   map[string]any `bson:"params,omitempty" json:"params,omitempty"`
	Fallback string         `bson:"fallback" json:"fallback"`
}

type Transition struct {
	FromSceneID string `bson:"fromSceneId" json:"fromSceneId"`
	ToSceneID   string `bson:"toSceneId" json:"toSceneId"`
	Type        string `bson:"type" json:"type"`
	DurationMS  int    `bson:"durationMs" json:"durationMs"`
}

type Typography struct {
	Personality  string `bson:"personality" json:"personality"`
	DisplayScale string `bson:"displayScale" json:"displayScale"`
	BodyMeasure  string `bson:"bodyMeasure" json:"bodyMeasure"`
}

type Navigation struct {
	Type         string `bson:"type" json:"type"`
	ShowProgress bool   `bson:"showProgress" json:"showProgress"`
	Keyboard     bool   `bson:"keyboard" json:"keyboard"`
	Label        string `bson:"label" json:"label"`
}

type ResponsiveRule struct {
	Target     string `bson:"target" json:"target"`
	Breakpoint string `bson:"breakpoint" json:"breakpoint"`
	Layout     string `bson:"layout" json:"layout"`
	Motion     string `bson:"motion" json:"motion"`
	EffectTier string `bson:"effectTier" json:"effectTier"`
}

type Registry struct {
	Layouts     map[string]struct{}
	Motions     map[string]struct{}
	Transitions map[string]struct{}
	Effects     map[string]struct{}
	Typography  map[string]struct{}
	Navigation  map[string]struct{}
}

func V1Registry() Registry {
	return Registry{
		Layouts: setOf(
			"cinematic-fullscreen", "editorial-split", "minimal-gallery", "film-strip",
			"contact-sheet", "scattered-prints", "timeline", "layered-depth", "horizontal-travel",
		),
		Motions: setOf(
			"none", "gentle-reveal", "slow-zoom", "blur-to-focus", "depth-reveal",
			"horizontal-drift", "camera-flash", "exposure-reveal", "clip-reveal", "rise-and-settle",
		),
		Transitions: setOf(
			"cut", "crossfade", "dip-to-black", "wipe", "dissolve", "slide-over", "flash", "page-turn",
		),
		Effects: setOf(
			"grain", "vignette", "none", "soft-haze", "light-leak", "letterbox", "monochrome", "color-wash",
		),
		Typography: setOf(
			"cinematic", "editorial", "documentary", "quiet", "contemporary", "journal",
			"playful", "brutalist", "nostalgic", "technical", "handwritten-accent",
		),
		Navigation: setOf("chapter-marks", "scene-progress", "minimal-progress"),
	}
}

func Validate(spec Spec, knownAssetIDs map[string]struct{}, registry Registry) error {
	if spec.SchemaVersion != SchemaVersionV1 {
		return fmt.Errorf("unsupported experience schema version %q", spec.SchemaVersion)
	}
	if strings.TrimSpace(spec.GeneratorVersion) == "" {
		return errors.New("generatorVersion is required")
	}
	if strings.TrimSpace(spec.DNA.Title) == "" || strings.TrimSpace(spec.DNA.CreativeThesis) == "" {
		return errors.New("experience DNA requires a title and creative thesis")
	}
	if strings.TrimSpace(spec.DNA.SignatureMoment.Concept) == "" || strings.TrimSpace(spec.DNA.SignatureMoment.MemoryTest) == "" {
		return errors.New("experience DNA requires a memorable signature moment")
	}
	if spec.DNA.MotionLanguage.Intensity < 0 || spec.DNA.MotionLanguage.Intensity > 1 {
		return errors.New("motion intensity must be between 0 and 1")
	}
	if len(spec.Scenes) == 0 {
		return errors.New("experience requires at least one scene")
	}

	sceneIDs := make(map[string]struct{}, len(spec.Scenes))
	for index, scene := range spec.Scenes {
		if scene.ID == "" {
			return fmt.Errorf("scene %d has no id", index)
		}
		if _, duplicate := sceneIDs[scene.ID]; duplicate {
			return fmt.Errorf("duplicate scene id %q", scene.ID)
		}
		sceneIDs[scene.ID] = struct{}{}
		if _, ok := registry.Layouts[scene.Layout]; !ok {
			return fmt.Errorf("scene %q uses unknown layout %q", scene.ID, scene.Layout)
		}
		if _, ok := registry.Motions[scene.Motion]; !ok {
			return fmt.Errorf("scene %q uses unknown motion %q", scene.ID, scene.Motion)
		}
		if _, ok := registry.Layouts[scene.Fallback.Layout]; !ok {
			return fmt.Errorf("scene %q uses unknown fallback layout %q", scene.ID, scene.Fallback.Layout)
		}
		if _, ok := registry.Motions[scene.Fallback.Motion]; !ok {
			return fmt.Errorf("scene %q uses unknown fallback motion %q", scene.ID, scene.Fallback.Motion)
		}
		for _, asset := range scene.Assets {
			if _, ok := knownAssetIDs[asset.AssetID]; !ok {
				return fmt.Errorf("scene %q references unknown asset %q", scene.ID, asset.AssetID)
			}
			if strings.TrimSpace(asset.Alt) == "" {
				return fmt.Errorf("scene %q asset %q has no alt text", scene.ID, asset.AssetID)
			}
			if asset.FocalX < 0 || asset.FocalX > 1 || asset.FocalY < 0 || asset.FocalY > 1 {
				return fmt.Errorf("scene %q asset %q has an invalid focal point", scene.ID, asset.AssetID)
			}
		}
		for _, effect := range scene.Effects {
			if _, ok := registry.Effects[effect.ID]; !ok {
				return fmt.Errorf("scene %q uses unknown effect %q", scene.ID, effect.ID)
			}
			if effect.Fallback != "" {
				if _, ok := registry.Effects[effect.Fallback]; !ok {
					return fmt.Errorf("scene %q effect %q uses unknown fallback %q", scene.ID, effect.ID, effect.Fallback)
				}
			}
		}
		if scene.LoadPriority != "eager" && scene.LoadPriority != "nearby" && scene.LoadPriority != "lazy" {
			return fmt.Errorf("scene %q has unknown load priority %q", scene.ID, scene.LoadPriority)
		}
	}

	if id := spec.DNA.SignatureMoment.SceneID; id != "" {
		if _, ok := sceneIDs[id]; !ok {
			return fmt.Errorf("signature moment references unknown scene %q", id)
		}
	}
	for _, assetID := range spec.DNA.SignatureMoment.AssetIDs {
		if _, ok := knownAssetIDs[assetID]; !ok {
			return fmt.Errorf("signature moment references unknown asset %q", assetID)
		}
	}
	for _, transition := range spec.Transitions {
		if _, ok := sceneIDs[transition.FromSceneID]; !ok {
			return fmt.Errorf("transition references unknown source scene %q", transition.FromSceneID)
		}
		if _, ok := sceneIDs[transition.ToSceneID]; !ok {
			return fmt.Errorf("transition references unknown destination scene %q", transition.ToSceneID)
		}
		if _, ok := registry.Transitions[transition.Type]; !ok {
			return fmt.Errorf("unknown transition %q", transition.Type)
		}
		if transition.DurationMS < 0 || transition.DurationMS > 10000 {
			return fmt.Errorf("transition duration must be between 0 and 10000 milliseconds")
		}
	}
	for _, effect := range spec.GlobalEffects {
		if _, ok := registry.Effects[effect.ID]; !ok {
			return fmt.Errorf("experience uses unknown global effect %q", effect.ID)
		}
		if effect.Fallback != "" {
			if _, ok := registry.Effects[effect.Fallback]; !ok {
				return fmt.Errorf("global effect %q uses unknown fallback %q", effect.ID, effect.Fallback)
			}
		}
	}
	if _, ok := registry.Typography[spec.Typography.Personality]; !ok {
		return fmt.Errorf("experience uses unknown typography personality %q", spec.Typography.Personality)
	}
	if _, ok := registry.Navigation[spec.Navigation.Type]; !ok {
		return fmt.Errorf("experience uses unknown navigation type %q", spec.Navigation.Type)
	}
	for _, rule := range spec.ResponsiveRules {
		if _, ok := sceneIDs[rule.Target]; !ok {
			return fmt.Errorf("responsive rule references unknown scene %q", rule.Target)
		}
		if _, ok := registry.Layouts[rule.Layout]; !ok {
			return fmt.Errorf("responsive rule for %q uses unknown layout %q", rule.Target, rule.Layout)
		}
		if _, ok := registry.Motions[rule.Motion]; !ok {
			return fmt.Errorf("responsive rule for %q uses unknown motion %q", rule.Target, rule.Motion)
		}
		if rule.Breakpoint != "mobile" && rule.Breakpoint != "tablet" && rule.Breakpoint != "desktop" {
			return fmt.Errorf("responsive rule for %q has unknown breakpoint %q", rule.Target, rule.Breakpoint)
		}
		if rule.EffectTier != "A" && rule.EffectTier != "B" && rule.EffectTier != "C" {
			return fmt.Errorf("responsive rule for %q has unknown effect tier %q", rule.Target, rule.EffectTier)
		}
	}
	return nil
}

func setOf(values ...string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[value] = struct{}{}
	}
	return result
}
