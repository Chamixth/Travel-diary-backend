package story

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const SchemaVersionV1 = "1.0"

type Record struct {
	ID             string     `bson:"_id,omitempty" json:"id"`
	UserID         string     `bson:"userId" json:"-"`
	TripID         string     `bson:"tripId" json:"tripId"`
	SourceRevision string     `bson:"sourceRevision" json:"sourceRevision"`
	Spec           Spec       `bson:"spec" json:"spec"`
	Provenance     Provenance `bson:"provenance" json:"provenance"`
	CreatedAt      time.Time  `bson:"createdAt" json:"createdAt"`
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

type Spec struct {
	SchemaVersion     string        `bson:"schemaVersion" json:"schemaVersion"`
	GeneratorVersion  string        `bson:"generatorVersion" json:"generatorVersion"`
	TitleCandidates   []string      `bson:"titleCandidates" json:"titleCandidates"`
	Summary           string        `bson:"summary" json:"summary"`
	EmotionalArc      []EmotionBeat `bson:"emotionalArc" json:"emotionalArc"`
	Chapters          []Chapter     `bson:"chapters" json:"chapters"`
	HeroAssetID       string        `bson:"heroAssetId" json:"heroAssetId"`
	ClimaxAssetID     string        `bson:"climaxAssetId" json:"climaxAssetId"`
	EndingAssetIDs    []string      `bson:"endingAssetIds" json:"endingAssetIds"`
	Motifs            []string      `bson:"motifs" json:"motifs"`
	RecurringElements []string      `bson:"recurringElements" json:"recurringElements"`
	OmittedAssetIDs   []string      `bson:"omittedAssetIds" json:"omittedAssetIds"`
}

type EmotionBeat struct {
	Order       int      `bson:"order" json:"order"`
	Name        string   `bson:"name" json:"name"`
	Description string   `bson:"description" json:"description"`
	Intensity   float64  `bson:"intensity" json:"intensity"`
	MomentIDs   []string `bson:"momentIds" json:"momentIds"`
}

type Chapter struct {
	ID               string   `bson:"id" json:"id"`
	Order            int      `bson:"order" json:"order"`
	Title            string   `bson:"title" json:"title"`
	NarrativeRole    string   `bson:"narrativeRole" json:"narrativeRole"`
	Pacing           string   `bson:"pacing" json:"pacing"`
	MomentIDs        []string `bson:"momentIds" json:"momentIds"`
	FeaturedAssets   []string `bson:"featuredAssets" json:"featuredAssets"`
	SupportingAssets []string `bson:"supportingAssets" json:"supportingAssets"`
}

func Validate(spec Spec, knownMomentIDs, knownAssetIDs map[string]struct{}) error {
	if spec.SchemaVersion != SchemaVersionV1 {
		return fmt.Errorf("unsupported story schema version %q", spec.SchemaVersion)
	}
	if strings.TrimSpace(spec.GeneratorVersion) == "" || strings.TrimSpace(spec.Summary) == "" {
		return errors.New("story requires generatorVersion and summary")
	}
	if len(spec.TitleCandidates) < 2 {
		return errors.New("story requires at least two title candidates")
	}
	if len(spec.Chapters) == 0 || len(spec.EmotionalArc) == 0 {
		return errors.New("story requires chapters and an emotional arc")
	}
	chapterIDs := make(map[string]struct{}, len(spec.Chapters))
	for _, chapter := range spec.Chapters {
		if chapter.ID == "" {
			return errors.New("story chapter requires an id")
		}
		if _, duplicate := chapterIDs[chapter.ID]; duplicate {
			return fmt.Errorf("duplicate story chapter %q", chapter.ID)
		}
		chapterIDs[chapter.ID] = struct{}{}
		for _, momentID := range chapter.MomentIDs {
			if _, ok := knownMomentIDs[momentID]; !ok {
				return fmt.Errorf("chapter %q references unknown moment %q", chapter.ID, momentID)
			}
		}
		for _, assetID := range append(append([]string{}, chapter.FeaturedAssets...), chapter.SupportingAssets...) {
			if _, ok := knownAssetIDs[assetID]; !ok {
				return fmt.Errorf("chapter %q references unknown asset %q", chapter.ID, assetID)
			}
		}
	}
	for _, assetID := range append(append([]string{spec.HeroAssetID, spec.ClimaxAssetID}, spec.EndingAssetIDs...), spec.OmittedAssetIDs...) {
		if assetID == "" {
			continue
		}
		if _, ok := knownAssetIDs[assetID]; !ok {
			return fmt.Errorf("story references unknown asset %q", assetID)
		}
	}
	for _, beat := range spec.EmotionalArc {
		if beat.Intensity < 0 || beat.Intensity > 1 {
			return fmt.Errorf("emotion beat %q intensity must be between 0 and 1", beat.Name)
		}
		for _, momentID := range beat.MomentIDs {
			if _, ok := knownMomentIDs[momentID]; !ok {
				return fmt.Errorf("emotion beat %q references unknown moment %q", beat.Name, momentID)
			}
		}
	}
	return nil
}
