package creative

import (
	"fmt"
	"strings"
)

func FingerprintFromConcept(concept Concept) Fingerprint {
	return Fingerprint{
		CreativeMetaphor:      concept.InteractionMetaphor,
		TypographyPersonality: concept.TypographyPersonality,
		MotionLanguage:        concept.MotionWorld,
		NavigationConcept:     concept.NavigationConcept,
		PaletteClass:          concept.VisualWorld,
		ImageTreatment:        concept.PhotographicTreatment,
		SignatureEffects:      []string{concept.SignatureInteraction},
		StoryStructure:        concept.EmotionalInterpretation + " " + concept.EndingConcept,
	}
}

// ValidateNovelty only blocks a concept when it is strongly similar to more
// than one prior album. Reuse of an individual primitive remains acceptable.
func ValidateNovelty(exploration Exploration, prior []Fingerprint) error {
	for _, concept := range exploration.Concepts {
		candidate := FingerprintFromConcept(concept)
		extremeMatches := 0
		for _, previous := range prior {
			if Similarity(candidate, previous) >= .80 {
				extremeMatches++
			}
		}
		if extremeMatches >= 2 {
			return fmt.Errorf("creative concept %q is extremely similar to %d prior experiences", concept.ID, extremeMatches)
		}
	}
	return nil
}

// Similarity intentionally remains explainable. It warns Creative Exploration
// about repeated overall experiences without banning individual primitives.
func Similarity(left, right Fingerprint) float64 {
	weightedMatches := 0.0
	totalWeight := 0.0
	compare := func(leftValue, rightValue string, weight float64) {
		totalWeight += weight
		if normalize(leftValue) != "" && normalize(leftValue) == normalize(rightValue) {
			weightedMatches += weight
		}
	}
	compare(left.CreativeMetaphor, right.CreativeMetaphor, 3)
	compare(left.TypographyPersonality, right.TypographyPersonality, 1)
	compare(left.MotionLanguage, right.MotionLanguage, 2)
	compare(left.NavigationConcept, right.NavigationConcept, 2)
	compare(left.PaletteClass, right.PaletteClass, 1)
	compare(left.ImageTreatment, right.ImageTreatment, 1)
	compare(left.StoryStructure, right.StoryStructure, 2)

	totalWeight += 3
	weightedMatches += 3 * jaccard(left.SceneSequence, right.SceneSequence)
	totalWeight += 2
	weightedMatches += 2 * jaccard(left.SignatureEffects, right.SignatureEffects)
	if totalWeight == 0 {
		return 0
	}
	return weightedMatches / totalWeight
}

func jaccard(left, right []string) float64 {
	union := make(map[string]struct{}, len(left)+len(right))
	intersection := 0
	leftSet := make(map[string]struct{}, len(left))
	for _, value := range left {
		value = normalize(value)
		if value != "" {
			leftSet[value] = struct{}{}
			union[value] = struct{}{}
		}
	}
	seenIntersection := make(map[string]struct{})
	for _, value := range right {
		value = normalize(value)
		if value == "" {
			continue
		}
		union[value] = struct{}{}
		if _, exists := leftSet[value]; exists {
			if _, counted := seenIntersection[value]; !counted {
				intersection++
				seenIntersection[value] = struct{}{}
			}
		}
	}
	if len(union) == 0 {
		return 0
	}
	return float64(intersection) / float64(len(union))
}

func normalize(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(value), " "))
}
