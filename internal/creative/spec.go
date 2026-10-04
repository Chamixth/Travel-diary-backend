package creative

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"travel-diary-backend/internal/experience"
)

const SchemaVersionV1 = "1.0"

type Record struct {
	ID                  string        `bson:"_id" json:"id"`
	UserID              string        `bson:"userId" json:"-"`
	TripID              string        `bson:"tripId" json:"tripId"`
	SourceRevision      string        `bson:"sourceRevision" json:"sourceRevision"`
	Exploration         Exploration   `bson:"exploration" json:"exploration"`
	AvoidedFingerprints []Fingerprint `bson:"avoidedFingerprints" json:"avoidedFingerprints"`
	Provenance          Provenance    `bson:"provenance" json:"provenance"`
	CreatedAt           time.Time     `bson:"createdAt" json:"createdAt"`
}

type FingerprintRecord struct {
	ID        string      `bson:"_id" json:"id"`
	UserID    string      `bson:"userId" json:"-"`
	TripID    string      `bson:"tripId" json:"tripId"`
	VersionID string      `bson:"versionId" json:"versionId"`
	Value     Fingerprint `bson:"value" json:"value"`
	CreatedAt time.Time   `bson:"createdAt" json:"createdAt"`
}

// DirectionRecord is the immutable output of Creative Director. The selected
// fingerprint is stored beside the direction so the artistic decision and the
// anti-template history cannot drift apart.
type DirectionRecord struct {
	ID             string      `bson:"_id" json:"id"`
	UserID         string      `bson:"userId" json:"-"`
	TripID         string      `bson:"tripId" json:"tripId"`
	SourceRevision string      `bson:"sourceRevision" json:"sourceRevision"`
	ExplorationID  string      `bson:"explorationId" json:"explorationId"`
	Direction      Direction   `bson:"direction" json:"direction"`
	Fingerprint    Fingerprint `bson:"fingerprint" json:"fingerprint"`
	Provenance     Provenance  `bson:"provenance" json:"provenance"`
	CreatedAt      time.Time   `bson:"createdAt" json:"createdAt"`
}

type Provenance struct {
	AgentID           string `bson:"agentId" json:"agentId"`
	PromptVersion     string `bson:"promptVersion" json:"promptVersion"`
	Provider          string `bson:"provider" json:"provider"`
	Model             string `bson:"model,omitempty" json:"model,omitempty"`
	ProviderRequestID string `bson:"providerRequestId,omitempty" json:"providerRequestId,omitempty"`
	InputTokens       int    `bson:"inputTokens,omitempty" json:"inputTokens,omitempty"`
	OutputTokens      int    `bson:"outputTokens,omitempty" json:"outputTokens,omitempty"`
	LatencyMS         int64  `bson:"latencyMs,omitempty" json:"latencyMs,omitempty"`
	FallbackReason    string `bson:"fallbackReason,omitempty" json:"fallbackReason,omitempty"`
}

type Concept struct {
	ID                      string   `bson:"id" json:"id"`
	Title                   string   `bson:"title" json:"title"`
	Thesis                  string   `bson:"thesis" json:"thesis"`
	InteractionMetaphor     string   `bson:"interactionMetaphor" json:"interactionMetaphor"`
	EmotionalInterpretation string   `bson:"emotionalInterpretation" json:"emotionalInterpretation"`
	VisualWorld             string   `bson:"visualWorld" json:"visualWorld"`
	MotionWorld             string   `bson:"motionWorld" json:"motionWorld"`
	TypographyPersonality   string   `bson:"typographyPersonality" json:"typographyPersonality"`
	PhotographicTreatment   string   `bson:"photographicTreatment" json:"photographicTreatment"`
	NavigationConcept       string   `bson:"navigationConcept" json:"navigationConcept"`
	SignatureInteraction    string   `bson:"signatureInteraction" json:"signatureInteraction"`
	Climax                  string   `bson:"climax" json:"climax"`
	EndingConcept           string   `bson:"endingConcept" json:"endingConcept"`
	Risks                   []string `bson:"risks" json:"risks"`
	TechnicalFeasibility    string   `bson:"technicalFeasibility" json:"technicalFeasibility"`
	WhyItFits               string   `bson:"whyItFits" json:"whyItFits"`
}

type Exploration struct {
	SchemaVersion    string    `bson:"schemaVersion" json:"schemaVersion"`
	GeneratorVersion string    `bson:"generatorVersion" json:"generatorVersion"`
	Concepts         []Concept `bson:"concepts" json:"concepts"`
}

// Direction deliberately stops at art direction. Scene and primitive choices
// belong to Experience Planner, which can regenerate independently.
type Direction struct {
	SchemaVersion      string                   `bson:"schemaVersion" json:"schemaVersion"`
	GeneratorVersion   string                   `bson:"generatorVersion" json:"generatorVersion"`
	SelectedConceptID  string                   `bson:"selectedConceptId" json:"selectedConceptId"`
	SelectionRationale string                   `bson:"selectionRationale" json:"selectionRationale"`
	DNA                experience.ExperienceDNA `bson:"dna" json:"dna"`
}

func ValidateExploration(exploration Exploration) error {
	if exploration.SchemaVersion != SchemaVersionV1 {
		return fmt.Errorf("unsupported creative exploration schema version %q", exploration.SchemaVersion)
	}
	if strings.TrimSpace(exploration.GeneratorVersion) == "" {
		return errors.New("creative exploration requires generatorVersion")
	}
	if len(exploration.Concepts) < 3 || len(exploration.Concepts) > 7 {
		return fmt.Errorf("creative exploration requires 3-7 concepts, got %d", len(exploration.Concepts))
	}
	ids := make(map[string]struct{}, len(exploration.Concepts))
	identities := make(map[string]struct{}, len(exploration.Concepts))
	for _, concept := range exploration.Concepts {
		if strings.TrimSpace(concept.ID) == "" {
			return errors.New("creative concept requires an id")
		}
		if _, duplicate := ids[concept.ID]; duplicate {
			return fmt.Errorf("duplicate creative concept id %q", concept.ID)
		}
		ids[concept.ID] = struct{}{}
		required := []string{
			concept.Title, concept.Thesis, concept.InteractionMetaphor, concept.EmotionalInterpretation,
			concept.VisualWorld, concept.MotionWorld, concept.TypographyPersonality,
			concept.PhotographicTreatment, concept.NavigationConcept, concept.SignatureInteraction,
			concept.Climax, concept.EndingConcept, concept.TechnicalFeasibility, concept.WhyItFits,
		}
		for _, value := range required {
			if strings.TrimSpace(value) == "" {
				return fmt.Errorf("creative concept %q has an empty required field", concept.ID)
			}
		}
		identity := normalize(concept.InteractionMetaphor) + "|" + normalize(concept.NavigationConcept) + "|" + normalize(concept.SignatureInteraction)
		if _, duplicate := identities[identity]; duplicate {
			return fmt.Errorf("creative concept %q repeats another concept's overall experience", concept.ID)
		}
		identities[identity] = struct{}{}
	}
	return nil
}

func ValidateDirection(direction Direction, exploration Exploration, knownAssetIDs map[string]struct{}) error {
	if direction.SchemaVersion != SchemaVersionV1 {
		return fmt.Errorf("unsupported creative direction schema version %q", direction.SchemaVersion)
	}
	if strings.TrimSpace(direction.GeneratorVersion) == "" {
		return errors.New("creative direction requires generatorVersion")
	}
	selected := false
	for _, concept := range exploration.Concepts {
		if concept.ID == direction.SelectedConceptID {
			selected = true
			break
		}
	}
	if !selected {
		return fmt.Errorf("creative direction selected unknown concept %q", direction.SelectedConceptID)
	}
	if strings.TrimSpace(direction.SelectionRationale) == "" {
		return errors.New("creative direction requires a selection rationale")
	}

	dna := direction.DNA
	required := map[string]string{
		"title": dna.Title, "creativeThesis": dna.CreativeThesis,
		"interactionMetaphor":      dna.InteractionMetaphor,
		"emotionalArc.opening":     dna.EmotionalArc.Opening,
		"emotionalArc.development": dna.EmotionalArc.Development,
		"emotionalArc.climax":      dna.EmotionalArc.Climax, "emotionalArc.ending": dna.EmotionalArc.Ending,
		"visualLanguage.paletteStrategy":    dna.VisualLanguage.PaletteStrategy,
		"visualLanguage.typographyStrategy": dna.VisualLanguage.TypographyStrategy,
		"visualLanguage.imageTreatment":     dna.VisualLanguage.ImageTreatment,
		"motionLanguage.personality":        dna.MotionLanguage.Personality,
		"motionLanguage.pacing":             dna.MotionLanguage.Pacing,
		"motionLanguage.physics":            dna.MotionLanguage.Physics,
		"motionLanguage.scrollBehavior":     dna.MotionLanguage.ScrollBehavior,
		"navigationConcept":                 dna.NavigationConcept,
		"signatureMoment.concept":           dna.SignatureMoment.Concept,
		"signatureMoment.memoryTest":        dna.SignatureMoment.MemoryTest,
	}
	for name, value := range required {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("creative direction requires %s", name)
		}
	}
	if len(dna.VisualLanguage.Personality) == 0 || len(dna.VisualLanguage.Atmosphere) == 0 {
		return errors.New("creative direction requires visual personality and atmosphere")
	}
	if dna.MotionLanguage.Intensity < 0 || dna.MotionLanguage.Intensity > 1 {
		return errors.New("creative direction motion intensity must be between 0 and 1")
	}
	if dna.SignatureMoment.SceneID != "" {
		return errors.New("creative direction cannot reference a scene before experience planning")
	}
	if len(dna.SignatureMoment.AssetIDs) == 0 {
		return errors.New("creative direction signature moment requires at least one asset")
	}
	for _, assetID := range dna.SignatureMoment.AssetIDs {
		if _, ok := knownAssetIDs[assetID]; !ok {
			return fmt.Errorf("creative direction references unknown signature asset %q", assetID)
		}
	}
	if len(strings.Fields(dna.SignatureMoment.MemoryTest)) < 6 {
		return errors.New("creative direction must explicitly explain what the user will remember tomorrow")
	}
	if len(dna.Constraints.Avoid) == 0 || len(dna.Constraints.AccessibilityNotes) == 0 || len(dna.Constraints.PerformanceNotes) == 0 {
		return errors.New("creative direction requires avoidance, accessibility, and performance constraints")
	}
	return nil
}

type Fingerprint struct {
	CreativeMetaphor      string   `bson:"creativeMetaphor" json:"creativeMetaphor"`
	SceneSequence         []string `bson:"sceneSequence" json:"sceneSequence"`
	TypographyPersonality string   `bson:"typographyPersonality" json:"typographyPersonality"`
	MotionLanguage        string   `bson:"motionLanguage" json:"motionLanguage"`
	NavigationConcept     string   `bson:"navigationConcept" json:"navigationConcept"`
	PaletteClass          string   `bson:"paletteClass" json:"paletteClass"`
	ImageTreatment        string   `bson:"imageTreatment" json:"imageTreatment"`
	SignatureEffects      []string `bson:"signatureEffects" json:"signatureEffects"`
	StoryStructure        string   `bson:"storyStructure" json:"storyStructure"`
}
