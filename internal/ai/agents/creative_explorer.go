package agents

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"travel-diary-backend/internal/ai/prompts"
	"travel-diary-backend/internal/ai/provider"
	"travel-diary-backend/internal/ai/schemas"
	"travel-diary-backend/internal/creative"
	"travel-diary-backend/internal/journey"
	"travel-diary-backend/internal/models"
	"travel-diary-backend/internal/story"
)

type CreativeExplorer struct {
	generator provider.StructuredGenerator
	model     string
}

type creativeExplorerInput struct {
	Story                story.Spec              `json:"story"`
	Journey              journey.Spec            `json:"journey"`
	PhotographicEvidence []creativePhotoEvidence `json:"photographicEvidence"`
	PriorFingerprints    []creative.Fingerprint  `json:"priorFingerprints"`
}

type creativePhotoEvidence struct {
	AssetID       string                 `json:"assetId"`
	Scene         models.SceneAnalysis   `json:"scene"`
	Visual        models.VisualAnalysis  `json:"visual"`
	Story         models.StoryAnalysis   `json:"story"`
	Quality       models.QualityAnalysis `json:"quality"`
	LocationGuess models.LocationGuess   `json:"locationGuess"`
}

func NewCreativeExplorer(generator provider.StructuredGenerator, model string) *CreativeExplorer {
	return &CreativeExplorer{generator: generator, model: model}
}

func (a *CreativeExplorer) Generate(
	ctx context.Context,
	storySpec story.Spec,
	journeySpec journey.Spec,
	analyses []models.TripImageAnalysis,
	priorFingerprints []creative.Fingerprint,
	correlationID string,
) (creative.Exploration, creative.Provenance, error) {
	if a == nil || a.generator == nil {
		return creative.Exploration{}, creative.Provenance{}, fmt.Errorf("creative explorer is not configured")
	}
	evidence := make([]creativePhotoEvidence, 0, len(analyses))
	for _, analysis := range analyses {
		evidence = append(evidence, creativePhotoEvidence{
			AssetID: analysis.ImageID, Scene: analysis.Scene, Visual: analysis.Visual,
			Story: analysis.Story, Quality: analysis.Quality, LocationGuess: analysis.LocationGuess,
		})
	}
	input, err := json.Marshal(creativeExplorerInput{
		Story: storySpec, Journey: journeySpec, PhotographicEvidence: evidence,
		PriorFingerprints: priorFingerprints,
	})
	if err != nil {
		return creative.Exploration{}, creative.Provenance{}, fmt.Errorf("encode creative explorer input: %w", err)
	}
	response, err := a.generator.Generate(ctx, provider.StructuredRequest{
		AgentID: prompts.CreativeExplorerV1.AgentID, PromptVersion: prompts.CreativeExplorerV1.Version,
		Model: a.model, SystemPrompt: prompts.CreativeExplorerV1.Text, Input: input,
		OutputSchema: schemas.CreativeExplorationV1, CorrelationID: correlationID, Timeout: 90 * time.Second,
	})
	if err != nil {
		return creative.Exploration{}, creative.Provenance{}, fmt.Errorf("generate CreativeExploration: %w", err)
	}
	var exploration creative.Exploration
	decoder := json.NewDecoder(bytes.NewReader(response.Output))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&exploration); err != nil {
		return creative.Exploration{}, creative.Provenance{}, fmt.Errorf("decode CreativeExploration: %w", err)
	}
	if err := ensureJSONEnd(decoder); err != nil {
		return creative.Exploration{}, creative.Provenance{}, fmt.Errorf("decode CreativeExploration: %w", err)
	}
	if exploration.GeneratorVersion != prompts.CreativeExplorerV1.Version {
		return creative.Exploration{}, creative.Provenance{}, fmt.Errorf("CreativeExploration generatorVersion %q is unsupported", exploration.GeneratorVersion)
	}
	if err := creative.ValidateExploration(exploration); err != nil {
		return creative.Exploration{}, creative.Provenance{}, fmt.Errorf("validate CreativeExploration: %w", err)
	}
	if err := creative.ValidateNovelty(exploration, priorFingerprints); err != nil {
		return creative.Exploration{}, creative.Provenance{}, fmt.Errorf("validate CreativeExploration novelty: %w", err)
	}
	return exploration, creative.Provenance{
		AgentID: prompts.CreativeExplorerV1.AgentID, PromptVersion: prompts.CreativeExplorerV1.Version,
		Provider: response.Provider, Model: response.Model, ProviderRequestID: response.ProviderRequestID,
		InputTokens: response.InputTokens, OutputTokens: response.OutputTokens, LatencyMS: response.Latency.Milliseconds(),
	}, nil
}
