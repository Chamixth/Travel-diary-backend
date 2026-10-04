package agents

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"travel-diary-backend/internal/ai/prompts"
	"travel-diary-backend/internal/ai/provider"
	"travel-diary-backend/internal/ai/schemas"
	"travel-diary-backend/internal/creative"
	"travel-diary-backend/internal/experience"
	"travel-diary-backend/internal/models"
	"travel-diary-backend/internal/story"
)

type ExperiencePlanner struct {
	generator provider.StructuredGenerator
	model     string
}

type experiencePlannerInput struct {
	ExperienceID         string                    `json:"experienceId"`
	Story                story.Spec                `json:"story"`
	Direction            creative.Direction        `json:"direction"`
	PhotographicEvidence []experiencePhotoEvidence `json:"photographicEvidence"`
	Primitives           primitiveCatalog          `json:"primitives"`
}

type experiencePhotoEvidence struct {
	AssetID       string                     `json:"assetId"`
	Width         int                        `json:"width"`
	Height        int                        `json:"height"`
	Caption       string                     `json:"caption"`
	Scene         models.SceneAnalysis       `json:"scene"`
	Visual        models.VisualAnalysis      `json:"visual"`
	Story         models.StoryAnalysis       `json:"story"`
	Composition   models.CompositionAnalysis `json:"composition"`
	LocationGuess models.LocationGuess       `json:"locationGuess"`
}

type primitiveCatalog struct {
	Layouts     []string `json:"layouts"`
	Motions     []string `json:"motions"`
	Transitions []string `json:"transitions"`
	Effects     []string `json:"effects"`
	Typography  []string `json:"typography"`
	Navigation  []string `json:"navigation"`
}

func NewExperiencePlanner(generator provider.StructuredGenerator, model string) *ExperiencePlanner {
	return &ExperiencePlanner{generator: generator, model: model}
}

func (a *ExperiencePlanner) Generate(
	ctx context.Context,
	experienceID string,
	storySpec story.Spec,
	direction creative.Direction,
	images []models.TripImage,
	analyses []models.TripImageAnalysis,
	registry experience.Registry,
	knownAssetIDs map[string]struct{},
	correlationID string,
) (experience.Spec, experience.Provenance, error) {
	if a == nil || a.generator == nil {
		return experience.Spec{}, experience.Provenance{}, fmt.Errorf("experience planner is not configured")
	}
	analysisByAsset := make(map[string]models.TripImageAnalysis, len(analyses))
	for _, analysis := range analyses {
		analysisByAsset[analysis.ImageID] = analysis
	}
	evidence := make([]experiencePhotoEvidence, 0, len(images))
	for _, image := range images {
		analysis := analysisByAsset[image.ID]
		evidence = append(evidence, experiencePhotoEvidence{
			AssetID: image.ID, Width: image.Width, Height: image.Height, Caption: analysis.Caption,
			Scene: analysis.Scene, Visual: analysis.Visual, Story: analysis.Story,
			Composition: analysis.Composition, LocationGuess: analysis.LocationGuess,
		})
	}
	input, err := json.Marshal(experiencePlannerInput{
		ExperienceID: experienceID, Story: storySpec, Direction: direction,
		PhotographicEvidence: evidence, Primitives: catalogFromRegistry(registry),
	})
	if err != nil {
		return experience.Spec{}, experience.Provenance{}, fmt.Errorf("encode experience planner input: %w", err)
	}
	response, err := a.generator.Generate(ctx, provider.StructuredRequest{
		AgentID: prompts.ExperiencePlannerV1.AgentID, PromptVersion: prompts.ExperiencePlannerV1.Version,
		Model: a.model, SystemPrompt: prompts.ExperiencePlannerV1.Text, Input: input,
		OutputSchema: schemas.ExperienceSpecV1, CorrelationID: correlationID, Timeout: 120 * time.Second,
	})
	if err != nil {
		return experience.Spec{}, experience.Provenance{}, fmt.Errorf("generate ExperienceSpec: %w", err)
	}
	var spec experience.Spec
	decoder := json.NewDecoder(bytes.NewReader(response.Output))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&spec); err != nil {
		return experience.Spec{}, experience.Provenance{}, fmt.Errorf("decode ExperienceSpec: %w", err)
	}
	if err := ensureJSONEnd(decoder); err != nil {
		return experience.Spec{}, experience.Provenance{}, fmt.Errorf("decode ExperienceSpec: %w", err)
	}
	if spec.ID != experienceID {
		return experience.Spec{}, experience.Provenance{}, fmt.Errorf("ExperienceSpec id %q does not match reserved id", spec.ID)
	}
	if spec.GeneratorVersion != prompts.ExperiencePlannerV1.Version {
		return experience.Spec{}, experience.Provenance{}, fmt.Errorf("ExperienceSpec generatorVersion %q is unsupported", spec.GeneratorVersion)
	}
	// Experience Planner places the approved signature moment in a scene but
	// cannot silently rewrite the Creative Director's artistic decision.
	signatureSceneID := spec.DNA.SignatureMoment.SceneID
	spec.DNA = direction.DNA
	spec.DNA.SignatureMoment.SceneID = signatureSceneID
	if err := experience.Validate(spec, knownAssetIDs, registry); err != nil {
		return experience.Spec{}, experience.Provenance{}, fmt.Errorf("validate ExperienceSpec: %w", err)
	}
	if err := experience.ValidatePlanned(spec, storySpec); err != nil {
		return experience.Spec{}, experience.Provenance{}, fmt.Errorf("validate planned experience: %w", err)
	}
	return spec, experience.Provenance{
		AgentID: prompts.ExperiencePlannerV1.AgentID, PromptVersion: prompts.ExperiencePlannerV1.Version,
		Provider: response.Provider, Model: response.Model, ProviderRequestID: response.ProviderRequestID,
		InputTokens: response.InputTokens, OutputTokens: response.OutputTokens, LatencyMS: response.Latency.Milliseconds(),
	}, nil
}

func catalogFromRegistry(registry experience.Registry) primitiveCatalog {
	return primitiveCatalog{
		Layouts: keys(registry.Layouts), Motions: keys(registry.Motions),
		Transitions: keys(registry.Transitions), Effects: keys(registry.Effects),
		Typography: keys(registry.Typography), Navigation: keys(registry.Navigation),
	}
}

func keys(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
