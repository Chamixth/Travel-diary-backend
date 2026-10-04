package story

import (
	"fmt"
	"sort"
	"strings"

	"travel-diary-backend/internal/journey"
	"travel-diary-backend/internal/models"
)

const BaselineGeneratorVersion = "story-director-baseline-v1"

// BuildBaseline produces an evidence-only story artifact. It is the safe
// fallback for the future model-backed Story Director and preserves the stage
// boundary without pretending that unsupported events occurred.
func BuildBaseline(trip models.Trip, journeySpec journey.Spec, analyses []models.TripImageAnalysis) Spec {
	analysisByAsset := make(map[string]models.TripImageAnalysis, len(analyses))
	for _, analysis := range analyses {
		analysisByAsset[analysis.ImageID] = analysis
	}

	chapters := make([]Chapter, 0, len(journeySpec.Moments))
	beats := make([]EmotionBeat, 0, len(journeySpec.Moments))
	motifCounts := make(map[string]int)
	var heroID, climaxID string
	bestHeroScore, bestClimaxScore := -1.0, -1.0

	for index, moment := range journeySpec.Moments {
		assets := append([]string{}, moment.AssetIDs...)
		sort.SliceStable(assets, func(i, j int) bool {
			return analysisByAsset[assets[i]].Story.Importance > analysisByAsset[assets[j]].Story.Importance
		})
		featured := []string{}
		supporting := []string{}
		if len(assets) > 0 {
			featured = append(featured, assets[0])
			supporting = append(supporting, assets[1:]...)
		}
		intensity := .35
		if len(assets) > 0 {
			total := 0.0
			for _, assetID := range assets {
				analysis := analysisByAsset[assetID]
				total += clamp01(analysis.Story.Importance)
				for _, motif := range []string{analysis.Scene.Primary, analysis.Story.EmotionalTone} {
					motif = strings.TrimSpace(motif)
					if motif != "" {
						motifCounts[motif]++
					}
				}
				score := analysis.Story.Importance + analysis.Quality.AestheticScore*.25
				if score > bestHeroScore {
					bestHeroScore, heroID = score, assetID
				}
				climaxScore := analysis.Story.Importance + float64(index)/float64(max(1, len(journeySpec.Moments)))*.15
				if climaxScore > bestClimaxScore {
					bestClimaxScore, climaxID = climaxScore, assetID
				}
			}
			intensity = clamp01(total / float64(len(assets)))
		}

		chapterID := fmt.Sprintf("chapter-%02d", index+1)
		chapters = append(chapters, Chapter{
			ID: chapterID, Order: index + 1, Title: moment.Title,
			NarrativeRole: narrativeRole(index, len(journeySpec.Moments)), Pacing: pacingFor(intensity),
			MomentIDs: []string{moment.ID}, FeaturedAssets: featured, SupportingAssets: supporting,
		})
		beats = append(beats, EmotionBeat{
			Order: index + 1, Name: nonEmpty(moment.EmotionalTone, moment.Title),
			Description: fmt.Sprintf("The journey moves through %s.", strings.ToLower(moment.Title)),
			Intensity:   intensity, MomentIDs: []string{moment.ID},
		})
	}

	ending := []string{}
	if len(journeySpec.Moments) > 0 {
		last := journeySpec.Moments[len(journeySpec.Moments)-1].AssetIDs
		start := max(0, len(last)-2)
		ending = append(ending, last[start:]...)
	}
	return Spec{
		SchemaVersion: SchemaVersionV1, GeneratorVersion: BaselineGeneratorVersion,
		TitleCandidates: titleCandidates(trip), Summary: storySummary(trip, len(journeySpec.Moments)),
		EmotionalArc: beats, Chapters: chapters, HeroAssetID: heroID, ClimaxAssetID: climaxID,
		EndingAssetIDs: ending, Motifs: topMotifs(motifCounts, 4),
		RecurringElements: []string{}, OmittedAssetIDs: append([]string{}, journeySpec.UnplacedAssetIDs...),
	}
}

func titleCandidates(trip models.Trip) []string {
	first := nonEmpty(trip.Title, "A journey remembered")
	second := "Notes from the road"
	if strings.TrimSpace(trip.Destination) != "" {
		second = "Notes from " + trip.Destination
	}
	if strings.EqualFold(first, second) {
		second = first + ", remembered"
	}
	return []string{first, second}
}

func storySummary(trip models.Trip, moments int) string {
	place := nonEmpty(trip.Destination, "the journey")
	return fmt.Sprintf("%d moments trace the trip through %s, from arrival to the final photographs.", moments, place)
}

func narrativeRole(index, total int) string {
	switch {
	case index == 0:
		return "opening"
	case index == total-1:
		return "ending"
	case index >= (total*2)/3:
		return "climax"
	default:
		return "development"
	}
}

func pacingFor(intensity float64) string {
	if intensity >= .75 {
		return "fast"
	}
	if intensity <= .35 {
		return "slow"
	}
	return "measured"
}

func topMotifs(counts map[string]int, limit int) []string {
	type counted struct {
		value string
		count int
	}
	values := make([]counted, 0, len(counts))
	for value, count := range counts {
		values = append(values, counted{value: value, count: count})
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].count == values[j].count {
			return values[i].value < values[j].value
		}
		return values[i].count > values[j].count
	})
	result := make([]string, 0, min(limit, len(values)))
	for _, value := range values[:min(limit, len(values))] {
		result = append(result, value.value)
	}
	return result
}

func clamp01(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func nonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
