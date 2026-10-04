package creative

import (
	"fmt"
	"strings"

	"travel-diary-backend/internal/experience"
	"travel-diary-backend/internal/story"
)

const BaselineDirectorVersion = "creative-director-baseline-v1"

func BuildBaselineDirection(storySpec story.Spec, exploration Exploration, priorFingerprints ...Fingerprint) Direction {
	if len(exploration.Concepts) == 0 {
		return Direction{SchemaVersion: SchemaVersionV1, GeneratorVersion: BaselineDirectorVersion}
	}
	concept := selectBaselineConcept(storySpec, exploration.Concepts, priorFingerprints)
	title := concept.Title
	if len(storySpec.TitleCandidates) > 0 && strings.TrimSpace(storySpec.TitleCandidates[0]) != "" {
		title = storySpec.TitleCandidates[0]
	}
	opening, development := storySpec.Summary, concept.EmotionalInterpretation
	if len(storySpec.EmotionalArc) > 0 {
		opening = nonEmpty(storySpec.EmotionalArc[0].Description, storySpec.EmotionalArc[0].Name, opening)
		middle := storySpec.EmotionalArc[len(storySpec.EmotionalArc)/2]
		development = nonEmpty(middle.Description, middle.Name, development)
	}
	signatureAssets := directionAssets(storySpec)
	memoryTest := fmt.Sprintf("The user remembers how %s", strings.TrimSuffix(concept.SignatureInteraction, "."))

	return Direction{
		SchemaVersion: SchemaVersionV1, GeneratorVersion: BaselineDirectorVersion,
		SelectedConceptID:  concept.ID,
		SelectionRationale: fmt.Sprintf("%s connects the story's emotional progression to a feasible, photograph-led interaction and preserves a clear reduced-motion path.", concept.Title),
		DNA: experience.ExperienceDNA{
			Title: title, Subtitle: concept.Title, CreativeThesis: concept.Thesis,
			InteractionMetaphor: concept.InteractionMetaphor,
			EmotionalArc: experience.EmotionalArc{
				Opening: opening, Development: development, Climax: concept.Climax, Ending: concept.EndingConcept,
			},
			VisualLanguage: experience.VisualLanguage{
				Personality:        []string{concept.TypographyPersonality, "photograph-led"},
				Atmosphere:         []string{concept.EmotionalInterpretation},
				PaletteStrategy:    "derive restrained scene colors from the hero and climax photographs with accessible contrast",
				TextureStrategy:    "use texture only when it belongs to the interaction metaphor",
				TypographyStrategy: concept.TypographyPersonality,
				ImageTreatment:     concept.PhotographicTreatment,
			},
			MotionLanguage: experience.MotionLanguage{
				Personality: concept.MotionWorld, Intensity: .5, Pacing: "follow each story chapter's authored pace",
				Physics: "measured movement with calm settling", ScrollBehavior: concept.InteractionMetaphor,
			},
			NavigationConcept: concept.NavigationConcept,
			SignatureMoment: experience.SignatureMoment{
				Concept: concept.SignatureInteraction, AssetIDs: signatureAssets, MemoryTest: memoryTest,
			},
			Constraints: experience.Constraints{
				Avoid:              append([]string{"generic card grids", "motion unrelated to the story", "photography reduced to decoration"}, concept.Risks...),
				AccessibilityNotes: []string{"Preserve the signature memory as a still composition with reduced motion.", "Keep all narrative information available without animation."},
				PerformanceNotes:   []string{"Load only imminent scene media and cap simultaneous animated layers.", "Use DOM and SVG as the baseline rendering tier."},
			},
		},
	}
}

func selectBaselineConcept(storySpec story.Spec, concepts []Concept, prior []Fingerprint) Concept {
	bestIndex, bestScore := 0, -2.0
	for index, concept := range concepts {
		fit := 0.0
		switch concept.ID {
		case "moving-window":
			if len(storySpec.Chapters) >= 4 {
				fit += .35
			}
			if len(storySpec.EmotionalArc) >= 4 {
				fit += .15
			}
		case "unsent-notes":
			if len(storySpec.Chapters) <= 3 {
				fit += .35
			}
		case "memory-constellation":
			if len(storySpec.Motifs) >= 2 {
				fit += .35
			}
			if len(storySpec.RecurringElements) >= 2 {
				fit += .15
			}
		}
		candidate := FingerprintFromConcept(concept)
		maximumSimilarity := 0.0
		for _, previous := range prior {
			maximumSimilarity = max(maximumSimilarity, Similarity(candidate, previous))
		}
		score := fit - maximumSimilarity*.8
		if score > bestScore {
			bestIndex, bestScore = index, score
		}
	}
	return concepts[bestIndex]
}

func FingerprintFromDirection(direction Direction, storySpec story.Spec) Fingerprint {
	sceneSequence := make([]string, 0, len(storySpec.Chapters))
	for _, chapter := range storySpec.Chapters {
		sceneSequence = append(sceneSequence, nonEmpty(chapter.NarrativeRole, chapter.Title))
	}
	dna := direction.DNA
	return Fingerprint{
		CreativeMetaphor: dna.InteractionMetaphor, SceneSequence: sceneSequence,
		TypographyPersonality: dna.VisualLanguage.TypographyStrategy,
		MotionLanguage:        strings.Join([]string{dna.MotionLanguage.Personality, dna.MotionLanguage.Pacing, dna.MotionLanguage.ScrollBehavior}, " | "),
		NavigationConcept:     dna.NavigationConcept, PaletteClass: dna.VisualLanguage.PaletteStrategy,
		ImageTreatment:   dna.VisualLanguage.ImageTreatment,
		SignatureEffects: []string{dna.SignatureMoment.Concept},
		StoryStructure:   strings.Join([]string{dna.EmotionalArc.Opening, dna.EmotionalArc.Development, dna.EmotionalArc.Climax, dna.EmotionalArc.Ending}, " | "),
	}
}

func directionAssets(spec story.Spec) []string {
	candidates := append([]string{spec.ClimaxAssetID, spec.HeroAssetID}, spec.EndingAssetIDs...)
	for _, chapter := range spec.Chapters {
		candidates = append(candidates, chapter.FeaturedAssets...)
	}
	result := make([]string, 0, 2)
	seen := make(map[string]struct{})
	for _, assetID := range candidates {
		assetID = strings.TrimSpace(assetID)
		if assetID == "" {
			continue
		}
		if _, exists := seen[assetID]; exists {
			continue
		}
		seen[assetID] = struct{}{}
		result = append(result, assetID)
		if len(result) == 2 {
			break
		}
	}
	return result
}

func nonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
