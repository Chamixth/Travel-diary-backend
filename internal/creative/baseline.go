package creative

import (
	"fmt"
	"strings"

	"travel-diary-backend/internal/story"
)

const BaselineExplorerVersion = "creative-explorer-baseline-v1"

func BuildBaselineExploration(storySpec story.Spec) Exploration {
	title := "the journey"
	if len(storySpec.TitleCandidates) > 0 && strings.TrimSpace(storySpec.TitleCandidates[0]) != "" {
		title = storySpec.TitleCandidates[0]
	}
	motifs := strings.Join(storySpec.Motifs, ", ")
	if motifs == "" {
		motifs = "the journey's recurring places and light"
	}
	return Exploration{
		SchemaVersion: SchemaVersionV1, GeneratorVersion: BaselineExplorerVersion,
		Concepts: []Concept{
			{
				ID: "moving-window", Title: "The Moving Window", Thesis: "The trip unfolds as a continuous passage through changing views.",
				InteractionMetaphor: "a journey seen through a moving window", EmotionalInterpretation: "anticipation gives way to arrival",
				VisualWorld: "wide photographic apertures, passing frames, and intervals of darkness", MotionWorld: "measured horizontal travel with still pauses",
				TypographyPersonality: "documentary", PhotographicTreatment: "natural widescreen crops with window-like masks",
				NavigationConcept: "a route line marking each chapter", SignatureInteraction: "the climax clears from a dark passage into the hero photograph",
				Climax: "the strongest photograph fills the view after a compressed transition", EndingConcept: "the final view recedes as the route reaches its last mark",
				Risks: []string{"horizontal choreography needs a vertical mobile equivalent"}, TechnicalFeasibility: "DOM and CSS motion with a reduced-motion still sequence",
				WhyItFits: fmt.Sprintf("%s has a chronological arc and recurring evidence of %s.", title, motifs),
			},
			{
				ID: "unsent-notes", Title: "Notes Never Sent", Thesis: "Each chapter feels like a private field note discovered after the trip.",
				InteractionMetaphor: "an unfolding set of unsent field notes", EmotionalInterpretation: "observation accumulates into affection and reflection",
				VisualWorld: "paper fields, precise captions, image edges, and generous quiet space", MotionWorld: "restrained page shifts and deliberate reveals",
				TypographyPersonality: "journal", PhotographicTreatment: "intimate prints balanced by occasional full-bleed interruptions",
				NavigationConcept: "dated note markers", SignatureInteraction: "a small annotated photograph expands beyond the page at the emotional peak",
				Climax: "the journal structure breaks open for the defining image", EndingConcept: "a final short note rests beside the last photograph",
				Risks: []string{"paper styling must avoid scrapbook cliché"}, TechnicalFeasibility: "responsive editorial layouts with lightweight transforms",
				WhyItFits: fmt.Sprintf("The chapter structure of %s supports intimate observation while keeping photographs primary.", title),
			},
			{
				ID: "memory-constellation", Title: "A Constellation of Moments", Thesis: "Memories begin as fragments and resolve into the shape of the journey.",
				InteractionMetaphor: "a field of memories connected by recurring motifs", EmotionalInterpretation: "fragmented recall becomes a coherent emotional map",
				VisualWorld: "floating photographs, fine connective paths, darkness, and isolated pools of color", MotionWorld: "depth-based gathering followed by calm fullscreen focus",
				TypographyPersonality: "contemporary", PhotographicTreatment: "layered scales with undistorted fullscreen focal images",
				NavigationConcept: "selectable memory nodes", SignatureInteraction: "scattered fragments align around the climax photograph",
				Climax: "the visual field resolves into one clear memory", EndingConcept: "the nodes dim until only the ending photographs remain",
				Risks: []string{"free placement requires strict collision and performance limits"}, TechnicalFeasibility: "progressive DOM/SVG enhancement with a linear Tier A fallback",
				WhyItFits: fmt.Sprintf("Recurring motifs—%s—can connect distinct moments without flattening their differences.", motifs),
			},
		},
	}
}
