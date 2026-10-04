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
	"travel-diary-backend/internal/models"
	"travel-diary-backend/internal/story"
)

type CreativeDirector struct {
	generator provider.StructuredGenerator
	model     string
}

type creativeDirectorInput struct {
	Story                story.Spec              `json:"story"`
	Exploration          creative.Exploration    `json:"exploration"`
	PhotographicEvidence []creativePhotoEvidence `json:"photographicEvidence"`
	PriorFingerprints    []creative.Fingerprint  `json:"priorFingerprints"`
}

func NewCreativeDirector(generator provider.StructuredGenerator, model string) *CreativeDirector {
	return &CreativeDirector{generator: generator, model: model}
}

func (a *CreativeDirector) Generate(
	ctx context.Context,
	storySpec story.Spec,
	exploration creative.Exploration,
	analyses []models.TripImageAnalysis,
	priorFingerprints []creative.Fingerprint,
	knownAssetIDs map[string]struct{},
	correlationID string,
) (creative.Direction, creative.Provenance, error) {
	if a == nil || a.generator == nil {
		return creative.Direction{}, creative.Provenance{}, fmt.Errorf("creative director is not configured")
	}
	evidence := make([]creativePhotoEvidence, 0, len(analyses))
	for _, analysis := range analyses {
		evidence = append(evidence, creativePhotoEvidence{
			AssetID: analysis.ImageID, Scene: analysis.Scene, Visual: analysis.Visual,
			Story: analysis.Story, Quality: analysis.Quality, LocationGuess: analysis.LocationGuess,
		})
	}
	input, err := json.Marshal(creativeDirectorInput{
		Story: storySpec, Exploration: exploration, PhotographicEvidence: evidence,
		PriorFingerprints: priorFingerprints,
	})
	if err != nil {
		return creative.Direction{}, creative.Provenance{}, fmt.Errorf("encode creative director input: %w", err)
	}
	response, err := a.generator.Generate(ctx, provider.StructuredRequest{
		AgentID: prompts.CreativeDirectorV1.AgentID, PromptVersion: prompts.CreativeDirectorV1.Version,
		Model: a.model, SystemPrompt: prompts.CreativeDirectorV1.Text, Input: input,
		OutputSchema: schemas.CreativeDirectionV1, CorrelationID: correlationID, Timeout: 90 * time.Second,
	})
	if err != nil {
		return creative.Direction{}, creative.Provenance{}, fmt.Errorf("generate CreativeDirection: %w", err)
	}
	var direction creative.Direction
	decoder := json.NewDecoder(bytes.NewReader(response.Output))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&direction); err != nil {
		return creative.Direction{}, creative.Provenance{}, fmt.Errorf("decode CreativeDirection: %w", err)
	}
	if err := ensureJSONEnd(decoder); err != nil {
		return creative.Direction{}, creative.Provenance{}, fmt.Errorf("decode CreativeDirection: %w", err)
	}
	if direction.GeneratorVersion != prompts.CreativeDirectorV1.Version {
		return creative.Direction{}, creative.Provenance{}, fmt.Errorf("CreativeDirection generatorVersion %q is unsupported", direction.GeneratorVersion)
	}
	if err := creative.ValidateDirection(direction, exploration, knownAssetIDs); err != nil {
		return creative.Direction{}, creative.Provenance{}, fmt.Errorf("validate CreativeDirection: %w", err)
	}
	return direction, creative.Provenance{
		AgentID: prompts.CreativeDirectorV1.AgentID, PromptVersion: prompts.CreativeDirectorV1.Version,
		Provider: response.Provider, Model: response.Model, ProviderRequestID: response.ProviderRequestID,
		InputTokens: response.InputTokens, OutputTokens: response.OutputTokens, LatencyMS: response.Latency.Milliseconds(),
	}, nil
}
