package agents

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"travel-diary-backend/internal/ai/prompts"
	"travel-diary-backend/internal/ai/provider"
	"travel-diary-backend/internal/ai/schemas"
	"travel-diary-backend/internal/journey"
	"travel-diary-backend/internal/models"
	"travel-diary-backend/internal/story"
)

type StoryDirector struct {
	generator provider.StructuredGenerator
	model     string
}

type storyDirectorInput struct {
	Trip     storyTripContext `json:"trip"`
	Journey  journey.Spec     `json:"journey"`
	Analyses []memoryEvidence `json:"analyses"`
}

type storyTripContext struct {
	Title         string    `json:"title"`
	Destination   string    `json:"destination"`
	Departure     time.Time `json:"departure"`
	Return        time.Time `json:"return"`
	CinematicMood string    `json:"cinematicMood"`
	Intention     string    `json:"intention,omitempty"`
}

type memoryEvidence struct {
	ImageID       string                     `json:"imageId"`
	TakenAt       *time.Time                 `json:"takenAt,omitempty"`
	Caption       string                     `json:"caption"`
	Scene         models.SceneAnalysis       `json:"scene"`
	Visual        models.VisualAnalysis      `json:"visual"`
	Content       models.ContentAnalysis     `json:"content"`
	Story         models.StoryAnalysis       `json:"story"`
	Quality       models.QualityAnalysis     `json:"quality"`
	Composition   models.CompositionAnalysis `json:"composition"`
	LocationGuess models.LocationGuess       `json:"locationGuess"`
}

func NewStoryDirector(generator provider.StructuredGenerator, model string) *StoryDirector {
	return &StoryDirector{generator: generator, model: model}
}

func (a *StoryDirector) Generate(
	ctx context.Context,
	trip models.Trip,
	journeySpec journey.Spec,
	analyses []models.TripImageAnalysis,
	knownMomentIDs map[string]struct{},
	knownAssetIDs map[string]struct{},
	correlationID string,
) (story.Spec, story.Provenance, error) {
	if a == nil || a.generator == nil {
		return story.Spec{}, story.Provenance{}, fmt.Errorf("story director is not configured")
	}
	evidence := make([]memoryEvidence, 0, len(analyses))
	for _, analysis := range analyses {
		evidence = append(evidence, memoryEvidence{
			ImageID: analysis.ImageID, TakenAt: analysis.TakenAt, Caption: analysis.Caption,
			Scene: analysis.Scene, Visual: analysis.Visual, Content: analysis.Content,
			Story: analysis.Story, Quality: analysis.Quality, Composition: analysis.Composition,
			LocationGuess: analysis.LocationGuess,
		})
	}
	input, err := json.Marshal(storyDirectorInput{
		Trip: storyTripContext{
			Title: trip.Title, Destination: trip.Destination, Departure: trip.Departure,
			Return: trip.Return, CinematicMood: trip.CinematicMood, Intention: trip.Intention,
		},
		Journey: journeySpec, Analyses: evidence,
	})
	if err != nil {
		return story.Spec{}, story.Provenance{}, fmt.Errorf("encode story director input: %w", err)
	}
	response, err := a.generator.Generate(ctx, provider.StructuredRequest{
		AgentID: prompts.StoryDirectorV1.AgentID, PromptVersion: prompts.StoryDirectorV1.Version,
		Model: a.model, SystemPrompt: prompts.StoryDirectorV1.Text,
		Input: input, OutputSchema: schemas.StorySpecV1, CorrelationID: correlationID,
		Timeout: 90 * time.Second,
	})
	if err != nil {
		return story.Spec{}, story.Provenance{}, fmt.Errorf("generate StorySpec: %w", err)
	}

	var spec story.Spec
	decoder := json.NewDecoder(bytes.NewReader(response.Output))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&spec); err != nil {
		return story.Spec{}, story.Provenance{}, fmt.Errorf("decode StorySpec: %w", err)
	}
	if err := ensureJSONEnd(decoder); err != nil {
		return story.Spec{}, story.Provenance{}, fmt.Errorf("decode StorySpec: %w", err)
	}
	if spec.GeneratorVersion != prompts.StoryDirectorV1.Version {
		return story.Spec{}, story.Provenance{}, fmt.Errorf("StorySpec generatorVersion %q is unsupported", spec.GeneratorVersion)
	}
	if err := story.Validate(spec, knownMomentIDs, knownAssetIDs); err != nil {
		return story.Spec{}, story.Provenance{}, fmt.Errorf("validate StorySpec: %w", err)
	}
	return spec, story.Provenance{
		AgentID: prompts.StoryDirectorV1.AgentID, PromptVersion: prompts.StoryDirectorV1.Version,
		Provider: response.Provider, Model: response.Model, ProviderRequestID: response.ProviderRequestID,
		InputTokens: response.InputTokens, OutputTokens: response.OutputTokens,
		LatencyMS: response.Latency.Milliseconds(),
	}, nil
}

func ensureJSONEnd(decoder *json.Decoder) error {
	var extra any
	err := decoder.Decode(&extra)
	if err == io.EOF {
		return nil
	}
	if err == nil {
		return fmt.Errorf("multiple JSON values are not allowed")
	}
	return err
}
