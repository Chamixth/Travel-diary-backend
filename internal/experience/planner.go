package experience

import (
	"fmt"
	"strings"

	"travel-diary-backend/internal/models"
	"travel-diary-backend/internal/story"
)

const BaselinePlannerVersion = "experience-planner-baseline-v1"

// BuildBaselinePlan is the deterministic planning path. It preserves the
// approved direction and maps story chapters onto the same registered runtime
// vocabulary available to the model-backed planner.
func BuildBaselinePlan(
	id string,
	storySpec story.Spec,
	dna ExperienceDNA,
	selectedConceptID string,
	images []models.TripImage,
	analyses []models.TripImageAnalysis,
) Spec {
	analysisByAsset := make(map[string]models.TripImageAnalysis, len(analyses))
	for _, analysis := range analyses {
		analysisByAsset[analysis.ImageID] = analysis
	}
	imageByAsset := make(map[string]models.TripImage, len(images))
	for _, image := range images {
		imageByAsset[image.ID] = image
	}

	scenes := make([]Scene, 0, len(storySpec.Chapters))
	for index, chapter := range storySpec.Chapters {
		assetIDs := sceneAssetIDs(chapter, dna.SignatureMoment.AssetIDs, 4)
		placements := make([]AssetPlacement, 0, min(4, len(assetIDs)))
		for _, assetID := range assetIDs {
			analysis := analysisByAsset[assetID]
			image := imageByAsset[assetID]
			focalX, focalY := focalPoint(analysis.Composition.SubjectPosition)
			placements = append(placements, AssetPlacement{
				AssetID: assetID, Role: assetRole(assetID, chapter, storySpec),
				Alt: baselineAlt(analysis, image, chapter.Title), FocalX: focalX, FocalY: focalY,
			})
		}
		layout := baselineLayout(selectedConceptID, len(placements), index)
		motion := baselineMotion(selectedConceptID, layout, chapter.Pacing, dna.MotionLanguage.Intensity, index)
		scenes = append(scenes, Scene{
			ID: fmt.Sprintf("scene-%02d", index+1), ChapterID: chapter.ID,
			Layout: layout, Motion: motion, Assets: placements,
			Content:      SceneContent{Eyebrow: chapter.NarrativeRole, Title: chapter.Title},
			Effects:      baselineEffects(selectedConceptID, layout),
			LoadPriority: loadPriority(index),
			Fallback:     SceneFallback{Layout: fallbackLayout(layout), Motion: "none"},
		})
	}

	dna.SignatureMoment.SceneID = signatureSceneID(scenes, dna.SignatureMoment.AssetIDs)
	transitions := make([]Transition, 0, max(0, len(scenes)-1))
	for index := 1; index < len(scenes); index++ {
		transitionType := baselineTransition(selectedConceptID, index)
		if scenes[index].ID == dna.SignatureMoment.SceneID {
			transitionType = "dip-to-black"
		}
		transitions = append(transitions, Transition{
			FromSceneID: scenes[index-1].ID, ToSceneID: scenes[index].ID,
			Type: transitionType, DurationMS: 650,
		})
	}
	responsive := make([]ResponsiveRule, 0, len(scenes)*2)
	for _, scene := range scenes {
		responsive = append(responsive,
			ResponsiveRule{Target: scene.ID, Breakpoint: "mobile", Layout: mobileLayout(scene.Layout), Motion: mobileMotion(scene.Motion), EffectTier: "A"},
			ResponsiveRule{Target: scene.ID, Breakpoint: "tablet", Layout: scene.Layout, Motion: scene.Motion, EffectTier: "A"},
		)
	}

	return Spec{
		SchemaVersion: SchemaVersionV1, GeneratorVersion: BaselinePlannerVersion, ID: id, DNA: dna,
		Scenes: scenes, Transitions: transitions, GlobalEffects: []Effect{},
		Typography:      Typography{Personality: approvedTypography(dna.VisualLanguage.TypographyStrategy), DisplayScale: "responsive", BodyMeasure: "readable"},
		Navigation:      Navigation{Type: baselineNavigation(selectedConceptID), ShowProgress: true, Keyboard: true, Label: "Journey chapters"},
		ResponsiveRules: responsive,
	}
}

// ValidatePlanned adds story coverage and intentional fallback requirements to
// the general render-contract validation used by legacy versions.
func ValidatePlanned(spec Spec, storySpec story.Spec) error {
	knownChapters := make(map[string]struct{}, len(storySpec.Chapters))
	for _, chapter := range storySpec.Chapters {
		knownChapters[chapter.ID] = struct{}{}
	}
	covered := make(map[string]struct{}, len(spec.Scenes))
	mobile := make(map[string]struct{}, len(spec.ResponsiveRules))
	for _, scene := range spec.Scenes {
		if _, ok := knownChapters[scene.ChapterID]; !ok {
			return fmt.Errorf("scene %q references unknown story chapter %q", scene.ID, scene.ChapterID)
		}
		covered[scene.ChapterID] = struct{}{}
		if len(scene.Assets) == 0 {
			return fmt.Errorf("planned scene %q must keep photography present", scene.ID)
		}
		if scene.Fallback.Motion != "none" {
			return fmt.Errorf("planned scene %q requires a reduced-motion fallback", scene.ID)
		}
	}
	for _, chapter := range storySpec.Chapters {
		if _, ok := covered[chapter.ID]; !ok {
			return fmt.Errorf("story chapter %q has no planned scene", chapter.ID)
		}
	}
	for _, rule := range spec.ResponsiveRules {
		if rule.Breakpoint == "mobile" {
			mobile[rule.Target] = struct{}{}
		}
	}
	for _, scene := range spec.Scenes {
		if _, ok := mobile[scene.ID]; !ok {
			return fmt.Errorf("planned scene %q has no intentional mobile rule", scene.ID)
		}
	}
	if spec.DNA.SignatureMoment.SceneID == "" {
		return fmt.Errorf("planned experience must place the signature moment in a scene")
	}
	signatureAssets := make(map[string]struct{}, len(spec.DNA.SignatureMoment.AssetIDs))
	for _, assetID := range spec.DNA.SignatureMoment.AssetIDs {
		signatureAssets[assetID] = struct{}{}
	}
	signaturePlaced := false
	for _, scene := range spec.Scenes {
		if scene.ID != spec.DNA.SignatureMoment.SceneID {
			continue
		}
		for _, asset := range scene.Assets {
			if _, ok := signatureAssets[asset.AssetID]; ok {
				signaturePlaced = true
			}
		}
	}
	if !signaturePlaced {
		return fmt.Errorf("signature scene must contain a selected signature asset")
	}
	return nil
}

func sceneAssetIDs(chapter story.Chapter, signatureAssetIDs []string, limit int) []string {
	all := append(append([]string{}, chapter.FeaturedAssets...), chapter.SupportingAssets...)
	selected := append([]string{}, all[:min(limit, len(all))]...)
	selectedSet := make(map[string]struct{}, len(selected))
	for _, assetID := range selected {
		selectedSet[assetID] = struct{}{}
	}
	chapterSet := make(map[string]struct{}, len(all))
	for _, assetID := range all {
		chapterSet[assetID] = struct{}{}
	}
	for _, assetID := range signatureAssetIDs {
		if _, belongs := chapterSet[assetID]; !belongs {
			continue
		}
		if _, included := selectedSet[assetID]; included {
			continue
		}
		if len(selected) < limit {
			selected = append(selected, assetID)
		} else if len(selected) > 0 {
			delete(selectedSet, selected[len(selected)-1])
			selected[len(selected)-1] = assetID
		}
		selectedSet[assetID] = struct{}{}
	}
	return selected
}

func baselineLayout(conceptID string, assets, index int) string {
	if assets <= 1 {
		return "cinematic-fullscreen"
	}
	switch conceptID {
	case "moving-window":
		if index%2 == 0 {
			return "horizontal-travel"
		}
		return "film-strip"
	case "unsent-notes":
		if assets == 2 {
			return "editorial-split"
		}
		return "scattered-prints"
	case "memory-constellation":
		if index%2 == 0 {
			return "layered-depth"
		}
		return "timeline"
	default:
		if assets == 2 {
			return "editorial-split"
		}
		return "contact-sheet"
	}
}

func baselineMotion(conceptID, layout, pacing string, intensity float64, index int) string {
	if intensity <= .2 || strings.EqualFold(pacing, "slow") {
		return "gentle-reveal"
	}
	switch conceptID {
	case "moving-window":
		return "horizontal-drift"
	case "unsent-notes":
		return "rise-and-settle"
	case "memory-constellation":
		if layout == "layered-depth" {
			return "depth-reveal"
		}
		return "blur-to-focus"
	}
	if intensity >= .65 && strings.EqualFold(pacing, "fast") {
		return "camera-flash"
	}
	if index == 0 {
		return "slow-zoom"
	}
	return "exposure-reveal"
}

func baselineEffects(conceptID, layout string) []Effect {
	switch conceptID {
	case "moving-window":
		return []Effect{{ID: "soft-haze", Version: "1", Fallback: "none"}}
	case "unsent-notes":
		return []Effect{{ID: "grain", Version: "1", Fallback: "none"}, {ID: "light-leak", Version: "1", Fallback: "none"}}
	case "memory-constellation":
		return []Effect{{ID: "color-wash", Version: "1", Fallback: "none", Params: map[string]any{"tone": "blue"}}}
	}
	if layout == "cinematic-fullscreen" {
		return []Effect{{ID: "vignette", Version: "1", Fallback: "none"}}
	}
	return []Effect{}
}

func baselineTransition(conceptID string, index int) string {
	switch conceptID {
	case "moving-window":
		return "wipe"
	case "unsent-notes":
		return "page-turn"
	case "memory-constellation":
		return "dissolve"
	default:
		if index%2 == 0 {
			return "slide-over"
		}
		return "crossfade"
	}
}

func baselineNavigation(conceptID string) string {
	switch conceptID {
	case "moving-window":
		return "scene-progress"
	case "memory-constellation":
		return "minimal-progress"
	default:
		return "chapter-marks"
	}
}

func signatureSceneID(scenes []Scene, assetIDs []string) string {
	wanted := make(map[string]struct{}, len(assetIDs))
	for _, assetID := range assetIDs {
		wanted[assetID] = struct{}{}
	}
	for _, scene := range scenes {
		for _, asset := range scene.Assets {
			if _, ok := wanted[asset.AssetID]; ok {
				return scene.ID
			}
		}
	}
	return ""
}

func focalPoint(position string) (float64, float64) {
	x, y := .5, .5
	lower := strings.ToLower(position)
	if strings.Contains(lower, "left") {
		x = .35
	}
	if strings.Contains(lower, "right") {
		x = .65
	}
	if strings.Contains(lower, "top") {
		y = .35
	}
	if strings.Contains(lower, "bottom") {
		y = .65
	}
	return x, y
}

func baselineAlt(analysis models.TripImageAnalysis, image models.TripImage, chapterTitle string) string {
	if value := strings.TrimSpace(analysis.Caption); value != "" {
		return value
	}
	if scene := strings.TrimSpace(analysis.Scene.Primary); scene != "" {
		return fmt.Sprintf("%s during %s", scene, chapterTitle)
	}
	if image.Width > image.Height {
		return fmt.Sprintf("A wide travel photograph from %s", chapterTitle)
	}
	return fmt.Sprintf("A travel photograph from %s", chapterTitle)
}

func assetRole(assetID string, chapter story.Chapter, spec story.Spec) string {
	if assetID == spec.ClimaxAssetID {
		return "climax"
	}
	if assetID == spec.HeroAssetID {
		return "hero"
	}
	for _, featured := range chapter.FeaturedAssets {
		if assetID == featured {
			return "featured"
		}
	}
	return "supporting"
}

func loadPriority(index int) string {
	if index == 0 {
		return "eager"
	}
	if index == 1 {
		return "nearby"
	}
	return "lazy"
}

func fallbackLayout(layout string) string {
	if layout == "cinematic-fullscreen" || layout == "layered-depth" {
		return "cinematic-fullscreen"
	}
	if layout == "editorial-split" {
		return layout
	}
	return "minimal-gallery"
}

func mobileLayout(layout string) string {
	if layout == "cinematic-fullscreen" || layout == "layered-depth" {
		return "cinematic-fullscreen"
	}
	if layout == "editorial-split" || layout == "timeline" {
		return "editorial-split"
	}
	return "minimal-gallery"
}

func mobileMotion(motion string) string {
	if motion == "slow-zoom" || motion == "horizontal-drift" || motion == "depth-reveal" || motion == "camera-flash" {
		return "gentle-reveal"
	}
	return motion
}

func approvedTypography(strategy string) string {
	lower := strings.ToLower(strategy)
	for _, personality := range []string{
		"cinematic", "editorial", "documentary", "quiet", "contemporary", "journal",
		"playful", "brutalist", "nostalgic", "technical", "handwritten-accent",
	} {
		if strings.Contains(lower, personality) {
			return personality
		}
	}
	return "documentary"
}
