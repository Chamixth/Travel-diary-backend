package experience

import (
	"fmt"
	"path/filepath"
	"strings"
	"travel-diary-backend/internal/models"
)

const LegacyAdapterGeneratorVersion = "legacy-album-adapter-v1"

// FromLegacyAlbum preserves existing generated albums while moving rendering
// onto the versioned runtime. It is deterministic for a given plan and image
// set and does not invent new story facts.
func FromLegacyAlbum(id string, plan models.AlbumPlan, images []models.TripImage) Spec {
	imageByID := make(map[string]models.TripImage, len(images))
	for _, image := range images {
		imageByID[image.ID] = image
	}

	scenes := make([]Scene, 0)
	for chapterIndex, chapter := range plan.Chapters {
		chapterID := strings.TrimSpace(chapter.ID)
		if chapterID == "" {
			chapterID = fmt.Sprintf("chapter-%d", chapterIndex+1)
		}
		for blockIndex, block := range chapter.Blocks {
			placements := make([]AssetPlacement, 0, len(block.ImageIDs))
			for imageIndex, imageID := range block.ImageIDs {
				image, ok := imageByID[imageID]
				if !ok {
					continue
				}
				name := strings.TrimSuffix(image.FileName, filepath.Ext(image.FileName))
				if strings.TrimSpace(name) == "" {
					name = "Travel photograph"
				}
				placements = append(placements, AssetPlacement{
					AssetID: imageID, Role: legacyAssetRole(block.Type, imageIndex),
					Alt: fmt.Sprintf("%s — %s", name, nonEmpty(chapter.Title, plan.Title)), FocalX: .5, FocalY: .5,
				})
			}

			sceneIndex := len(scenes)
			scene := Scene{
				ID: fmt.Sprintf("scene-%d-%d", chapterIndex+1, blockIndex+1), ChapterID: chapterID,
				Layout: legacyLayout(block.Type), Motion: legacyMotion(block.Type), Assets: placements,
				Content: SceneContent{
					Eyebrow: nonEmpty(block.Eyebrow, chapter.Eyebrow),
					Title:   nonEmpty(block.Title, chapter.Title),
					Body:    nonEmpty(block.Text, block.Description, block.Quote, chapter.Description, chapter.Quote),
					Caption: block.Caption,
				},
				Effects: []Effect{}, LoadPriority: legacyLoadPriority(sceneIndex),
				Fallback: SceneFallback{Layout: "minimal-gallery", Motion: "none"},
			}
			if block.Type == "album_cover" || block.Type == "closing_frame" {
				scene.Effects = []Effect{{ID: "vignette", Version: "1", Fallback: "none"}}
				scene.Fallback.Layout = "cinematic-fullscreen"
			}
			scenes = append(scenes, scene)
		}
	}

	transitions := make([]Transition, 0, max(0, len(scenes)-1))
	for index := 1; index < len(scenes); index++ {
		transitionType := "crossfade"
		if scenes[index-1].ChapterID != scenes[index].ChapterID {
			transitionType = "dip-to-black"
		}
		transitions = append(transitions, Transition{
			FromSceneID: scenes[index-1].ID, ToSceneID: scenes[index].ID,
			Type: transitionType, DurationMS: 650,
		})
	}

	signatureIndex := len(scenes) - 1
	if signatureIndex > 0 {
		signatureIndex--
	}
	var signature SignatureMoment
	if signatureIndex >= 0 {
		signatureScene := scenes[signatureIndex]
		assetIDs := make([]string, 0, len(signatureScene.Assets))
		for _, asset := range signatureScene.Assets {
			assetIDs = append(assetIDs, asset.AssetID)
		}
		signature = SignatureMoment{
			Concept: nonEmpty(signatureScene.Content.Title, "The journey's defining memory"),
			SceneID: signatureScene.ID, AssetIDs: assetIDs,
			MemoryTest: "The photographs expand into the defining moment before the journey closes.",
		}
	}

	tone := nonEmpty(plan.Tone, "documentary")
	return Spec{
		SchemaVersion: SchemaVersionV1, GeneratorVersion: LegacyAdapterGeneratorVersion, ID: id,
		DNA: ExperienceDNA{
			Title: nonEmpty(plan.Title, "A journey remembered"), Subtitle: plan.Subtitle,
			CreativeThesis:      "The original album story is retold as a paced, photograph-led journey.",
			InteractionMetaphor: "a cinematic travel journal",
			EmotionalArc:        EmotionalArc{Opening: "arrival", Development: "discovery", Climax: "the defining memory", Ending: "reflection"},
			VisualLanguage: VisualLanguage{
				Personality: []string{tone}, Atmosphere: []string{tone}, PaletteStrategy: "photo-derived",
				TypographyStrategy: "documentary", ImageTreatment: "natural",
			},
			MotionLanguage:    MotionLanguage{Personality: "restrained", Intensity: .35, Pacing: "measured", Physics: "soft", ScrollBehavior: "vertical"},
			NavigationConcept: "chapter progress", SignatureMoment: signature,
			Constraints: Constraints{
				Avoid:              []string{"generic cards", "decorative motion"},
				AccessibilityNotes: []string{"All photographs retain text alternatives.", "Reduced motion uses still compositions."},
				PerformanceNotes:   []string{"Only the opening scene loads eagerly."},
			},
		},
		Scenes: scenes, Transitions: transitions, GlobalEffects: []Effect{},
		Typography:      Typography{Personality: "documentary", DisplayScale: "large", BodyMeasure: "readable"},
		Navigation:      Navigation{Type: "chapter-marks", ShowProgress: true, Keyboard: true, Label: "Journey scenes"},
		ResponsiveRules: []ResponsiveRule{},
	}
}

func legacyLayout(blockType string) string {
	switch blockType {
	case "album_cover", "full_bleed_image", "full_bleed_quote", "panorama", "closing_frame":
		return "cinematic-fullscreen"
	case "portrait_pair", "landscape_pair", "image_caption", "chapter_split", "chapter_transition", "story_text":
		return "editorial-split"
	default:
		return "minimal-gallery"
	}
}

func legacyMotion(blockType string) string {
	switch blockType {
	case "album_cover", "panorama":
		return "slow-zoom"
	case "story_text", "chapter_split":
		return "none"
	default:
		return "gentle-reveal"
	}
}

func legacyAssetRole(blockType string, index int) string {
	if blockType == "album_cover" || blockType == "closing_frame" || index == 0 {
		return "hero"
	}
	return "supporting"
}

func legacyLoadPriority(index int) string {
	if index == 0 {
		return "eager"
	}
	if index == 1 {
		return "nearby"
	}
	return "lazy"
}

func nonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
