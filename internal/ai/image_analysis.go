package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"travel-diary-backend/internal/ai/prompts"
	"travel-diary-backend/internal/models"
)

type ImageAnalyzer interface {
	Analyze(ctx context.Context, imageURL, model string) (models.TripImageAnalysis, error)
}
type OpenAIImageAnalyzer struct {
	key    string
	client *http.Client
}

func NewOpenAIImageAnalyzer(key string) *OpenAIImageAnalyzer {
	return &OpenAIImageAnalyzer{key: key, client: &http.Client{Timeout: 90 * time.Second}}
}

type chatRequest struct {
	Model          string         `json:"model"`
	Messages       []chatMessage  `json:"messages"`
	ResponseFormat responseFormat `json:"response_format"`
}
type chatMessage struct {
	Role    string        `json:"role"`
	Content []contentPart `json:"content"`
}
type contentPart struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *imageURL `json:"image_url,omitempty"`
}
type imageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail"`
}
type responseFormat struct {
	Type       string `json:"type"`
	JSONSchema schema `json:"json_schema"`
}
type schema struct {
	Name   string         `json:"name"`
	Strict bool           `json:"strict"`
	Schema map[string]any `json:"schema"`
}
type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type albumQuotesResponse struct {
	Quotes []models.AlbumQuote `json:"quotes"`
}

type AlbumPlanningImage struct {
	ID       string                   `json:"id"`
	FileName string                   `json:"fileName"`
	Width    int                      `json:"width"`
	Height   int                      `json:"height"`
	EXIF     *models.ImageEXIF        `json:"exif,omitempty"`
	Analysis models.TripImageAnalysis `json:"analysis"`
}

func GenerateSmartAlbumPlan(ctx context.Context, key, model, title, destination, mood, intention string, departure, returnDate time.Time, images []AlbumPlanningImage) (models.AlbumPlan, error) {
	if key == "" {
		return models.AlbumPlan{}, fmt.Errorf("OPENAI_KEY is not configured")
	}
	contextJSON, _ := json.Marshal(map[string]any{
		"trip": map[string]any{
			"title": title, "destination": destination, "mood": mood, "intention": intention,
			"departure": departure.Format(time.RFC3339), "return": returnDate.Format(time.RFC3339),
		},
		"images": images,
	})
	prompt := fmt.Sprintf(`You are the lead photo editor, cinematic storyteller, and travel-album art director for VoyaLoom.

Transform the trip context and analyzed travel photos into a polished, emotionally coherent, cinematic album plan. Curate like an experienced travel magazine editor: understand the story, group images into scenes, select meaningful moments, omit weak or redundant photos, and create intentional visual rhythm. Do not behave like a gallery generator and do not simply place every image into blocks.

Before returning JSON, internally reconstruct chronology, cluster images into story beats, identify cover/hero/supporting/detail/transition/closing candidates, design chapters, choose layouts from orientation and composition, write concise copy, and validate every rule below. Do not output reasoning.

CHRONOLOGY:
- Use EXIF capturedAt/DateTimeOriginal first, then other timestamps, trip dates, analysis context, and visual inference.
- With reliable dates, create one chapter per meaningful travel day and keep images chronological inside each chapter.
- Without reliable dates, infer coherent chapters from location, activity, visual similarity, subjects, environment, and narrative continuity.

CURATION:
- Do not force every uploaded image into the album. Omit blurry, redundant, weak, unrelated, screenshot, or near-duplicate images unless narrative importance justifies them.
- Select one strongest cover. The cover must be the first block in the entire album.
- Use hero images sparingly. A cover may repeat once only when editorially justified.
- Generate exactly one closing_frame for the entire album, as the final block of the final chapter.

SUPPORTED BLOCKS AND STRICT IMAGE COUNTS:
- album_cover: exactly 1 image
- full_bleed_image, image_caption, panorama, closing_frame: exactly 1 image
- full_bleed_quote: exactly 1 image when an image is used and a non-empty quote
- portrait_pair, landscape_pair: exactly 2 images
- editorial_grid: 3 to 6 images, never more than 6
- film_strip: 3 to 5 images, ideally one short sequence
- story_text: meaningful narrative text, with at most 1 supporting image
- chapter_split and chapter_transition: use only where they improve structure and flow
- Use only the supported block type names. Never invent a type.

EDITORIAL RHYTHM:
- Vary breathing space, hero moments, intimate captions, supporting sequences, narrative text, quotes, and transitions.
- Start the first chapter with the album_cover, then use a chapter_split or story_text to establish the journey before the first image sequence.
- Give every meaningful image block a concise editorial title or caption when the supplied analysis supports one.
- End the final chapter with a closing_frame followed by no additional image block.
- Prefer 2-4 clearly named chapters with distinct emotional movement instead of one long gallery chapter.
- Choose layouts using orientation, aspect ratio, faces, negative space, safe text areas, quality, scene, story importance, suitePlace, and timestamps.
- Use full_bleed_image/panorama for strong landscapes, portrait_pair for complementary vertical images, landscape_pair for related horizontal images, editorial_grid for 3-6 supporting images, and film_strip for 3-5 related chronological details.
- Use full_bleed_quote sparingly: normally 0-2 per album, maximum 3 for a very large album. Quotes must be original and never attributed.
- Captions must be concise editorial captions, not object descriptions. Story text must connect moments without inventing unsupported events.

IMAGE ID AND OUTPUT VALIDATION:
- Use only supplied image IDs. Never invent, modify, or replace IDs with filenames.
- Minimize duplicate image use.
- The album_cover must exist exactly once and be the first album block.
- The closing_frame must exist exactly once and be the final album block, with no photo blocks after it.
- Validate all block image counts, non-empty quote blocks, meaningful story text, chronology, and varied intentional rhythm before returning.
- Return only JSON matching the supplied schema. All placement and transition values must be uppercase.

Trip and analyzed image context:
%s`, contextJSON)
	reqBody := chatRequest{
		Model:    model,
		Messages: []chatMessage{{Role: "user", Content: []contentPart{{Type: "text", Text: prompt}}}},
		ResponseFormat: responseFormat{Type: "json_schema", JSONSchema: schema{
			Name: "smart_travel_album_plan", Strict: true, Schema: albumPlanSchema(),
		}},
	}
	data, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/chat/completions", bytes.NewReader(data))
	if err != nil {
		return models.AlbumPlan{}, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	res, err := (&http.Client{Timeout: 120 * time.Second}).Do(req)
	if err != nil {
		return models.AlbumPlan{}, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 2048))
		return models.AlbumPlan{}, fmt.Errorf("openai album plan request failed with status %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}
	var out chatResponse
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return models.AlbumPlan{}, err
	}
	if len(out.Choices) == 0 {
		return models.AlbumPlan{}, fmt.Errorf("openai returned no album plan choices")
	}
	var plan models.AlbumPlan
	if err := json.Unmarshal([]byte(out.Choices[0].Message.Content), &plan); err != nil {
		return models.AlbumPlan{}, fmt.Errorf("could not parse OpenAI album plan: %w", err)
	}
	return plan, nil
}

func albumPlanSchema() map[string]any {
	stringField := map[string]any{"type": "string"}
	blockVariants := make([]any, 0, 13)
	for _, variant := range []struct {
		typ       string
		minImages int
		maxImages int
		textMin   int
		quoteMin  int
	}{
		{typ: "album_cover", minImages: 1, maxImages: 1},
		{typ: "full_bleed_image", minImages: 1, maxImages: 1},
		{typ: "image_caption", minImages: 1, maxImages: 1},
		{typ: "panorama", minImages: 1, maxImages: 1},
		{typ: "closing_frame", minImages: 1, maxImages: 1},
		{typ: "portrait_pair", minImages: 2, maxImages: 2},
		{typ: "landscape_pair", minImages: 2, maxImages: 2},
		{typ: "editorial_grid", minImages: 3, maxImages: 6},
		{typ: "film_strip", minImages: 3, maxImages: 5},
		{typ: "full_bleed_quote", minImages: 0, maxImages: 1, quoteMin: 1},
		{typ: "story_text", minImages: 0, maxImages: 1, textMin: 1},
		{typ: "chapter_split", minImages: 0, maxImages: 1},
		{typ: "chapter_transition", minImages: 0, maxImages: 1, textMin: 1},
	} {
		properties := map[string]any{
			"type":         map[string]any{"type": "string", "enum": []string{variant.typ}},
			"imageIds":     map[string]any{"type": "array", "items": stringField, "minItems": variant.minImages, "maxItems": variant.maxImages},
			"textPosition": stringField,
			"eyebrow":      stringField,
			"title":        stringField,
			"text":         stringField,
			"quote":        stringField,
			"description":  stringField,
			"caption":      stringField,
		}
		if variant.textMin > 0 {
			properties["text"] = map[string]any{"type": "string", "minLength": variant.textMin}
		}
		if variant.quoteMin > 0 {
			properties["quote"] = map[string]any{"type": "string", "minLength": variant.quoteMin}
		}
		blockVariants = append(blockVariants, map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"properties":           properties,
			"required":             []string{"type", "imageIds", "textPosition", "eyebrow", "title", "text", "quote", "description", "caption"},
		})
	}
	block := map[string]any{"anyOf": blockVariants}
	chapter := map[string]any{"type": "object", "additionalProperties": false, "properties": map[string]any{
		"id": stringField, "order": map[string]any{"type": "integer"}, "eyebrow": stringField, "title": stringField, "quote": stringField, "description": stringField, "blocks": map[string]any{"type": "array", "items": block},
	}, "required": []string{"id", "order", "eyebrow", "title", "quote", "description", "blocks"}}
	quote := map[string]any{"type": "object", "additionalProperties": false, "properties": map[string]any{
		"from": stringField, "to": stringField, "text": stringField, "order": map[string]any{"type": "integer"},
	}, "required": []string{"from", "to", "text", "order"}}
	return map[string]any{"type": "object", "additionalProperties": false, "properties": map[string]any{
		"title": stringField, "subtitle": stringField, "tone": stringField, "chapters": map[string]any{"type": "array", "items": chapter}, "quotes": map[string]any{"type": "array", "items": quote},
	}, "required": []string{"title", "subtitle", "tone", "chapters", "quotes"}}
}

func GenerateAlbumQuotes(ctx context.Context, key, model, title, destination string, analyses []models.TripImageAnalysis) ([]models.AlbumQuote, error) {
	if key == "" {
		return nil, fmt.Errorf("OPENAI_KEY is not configured")
	}

	tones := make([]string, 0, len(analyses))
	for _, analysis := range analyses {
		if analysis.Story.EmotionalTone != "" {
			tones = append(tones, analysis.Story.EmotionalTone)
		}
	}
	contextJSON, _ := json.Marshal(map[string]any{
		"title": title, "destination": destination, "emotionalTones": tones,
	})
	prompt := fmt.Sprintf("Create five original, elegant, cinematic transition quotes for a travel photo album. Return only JSON. Do not quote existing copyrighted text. Each quote should be concise, poetic, emotionally restrained, and suitable for a premium editorial album. Generate exactly these transitions in order: COVER to PROLOGUE, PROLOGUE to HEADER, HEADER to HIGHLIGHTS, HIGHLIGHTS to EPILOGUE, EPILOGUE to CREDITS. Use the exact uppercase from and to values. Album context: %s", contextJSON)
	reqBody := chatRequest{
		Model:    model,
		Messages: []chatMessage{{Role: "user", Content: []contentPart{{Type: "text", Text: prompt}}}},
		ResponseFormat: responseFormat{Type: "json_schema", JSONSchema: schema{
			Name: "album_transition_quotes", Strict: true, Schema: map[string]any{
				"type": "object", "additionalProperties": false,
				"properties": map[string]any{"quotes": map[string]any{
					"type": "array", "minItems": 5, "maxItems": 5,
					"items": map[string]any{"type": "object", "additionalProperties": false, "properties": map[string]any{
						"from":  map[string]any{"type": "string", "enum": []string{"COVER", "PROLOGUE", "HEADER", "HIGHLIGHTS", "EPILOGUE"}},
						"to":    map[string]any{"type": "string", "enum": []string{"PROLOGUE", "HEADER", "HIGHLIGHTS", "EPILOGUE", "CREDITS"}},
						"text":  map[string]any{"type": "string", "minLength": 8, "maxLength": 180},
						"order": map[string]any{"type": "integer"},
					}, "required": []string{"from", "to", "text", "order"}},
				}}, "required": []string{"quotes"},
			},
		}},
	}
	data, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/chat/completions", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	res, err := (&http.Client{Timeout: 60 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 2048))
		return nil, fmt.Errorf("openai quote request failed with status %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}
	var out chatResponse
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, err
	}
	if len(out.Choices) == 0 {
		return nil, fmt.Errorf("openai returned no quote choices")
	}
	var result albumQuotesResponse
	if err := json.Unmarshal([]byte(out.Choices[0].Message.Content), &result); err != nil {
		return nil, fmt.Errorf("could not parse OpenAI album quotes: %w", err)
	}
	return result.Quotes, nil
}

func (o *OpenAIImageAnalyzer) Analyze(ctx context.Context, imageURLValue, model string) (models.TripImageAnalysis, error) {
	if o.key == "" {
		return models.TripImageAnalysis{}, fmt.Errorf("OPENAI_KEY is not configured")
	}
	reqBody := chatRequest{Model: model, Messages: []chatMessage{{Role: "user", Content: []contentPart{{Type: "text", Text: prompts.MemoryAnalystV1.Text}, {Type: "image_url", ImageURL: &imageURL{URL: imageURLValue, Detail: "low"}}}}}, ResponseFormat: responseFormat{Type: "json_schema", JSONSchema: schema{Name: "travel_image_analysis", Strict: true, Schema: imageSchema()}}}
	data, _ := json.Marshal(reqBody)
	var last error
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/chat/completions", bytes.NewReader(data))
		if err != nil {
			last = err
			break
		}
		req.Header.Set("Authorization", "Bearer "+o.key)
		req.Header.Set("Content-Type", "application/json")
		res, err := o.client.Do(req)
		if err == nil && res.StatusCode >= 200 && res.StatusCode < 300 {
			var out chatResponse
			err = json.NewDecoder(res.Body).Decode(&out)
			res.Body.Close()
			if err == nil && len(out.Choices) > 0 {
				var a models.TripImageAnalysis
				if err = json.Unmarshal([]byte(out.Choices[0].Message.Content), &a); err == nil {
					a.SuitePlace = strings.ToUpper(strings.TrimSpace(a.SuitePlace))
					if a.Memory.SchemaVersion == 0 {
						a.Memory.SchemaVersion = models.MemoryAnalysisVersion
					}
					return a, nil
				}
			}
			if err == nil {
				err = fmt.Errorf("openai returned no analysis choices")
			}
			last = fmt.Errorf("could not parse OpenAI analysis: %w", err)
		} else if res != nil {
			body, readErr := io.ReadAll(io.LimitReader(res.Body, 2048))
			res.Body.Close()
			if readErr != nil {
				last = fmt.Errorf("openai request failed with status %d (could not read response)", res.StatusCode)
			} else {
				message := strings.TrimSpace(string(body))
				last = fmt.Errorf("openai request failed with status %d: %s", res.StatusCode, message)
			}
		} else {
			last = err
		}
		if attempt < 2 {
			select {
			case <-time.After(time.Duration(1<<attempt) * time.Second):
			case <-ctx.Done():
				return models.TripImageAnalysis{}, ctx.Err()
			}
		}
	}
	return models.TripImageAnalysis{}, last
}
func imageSchema() map[string]any {
	return map[string]any{"type": "object", "additionalProperties": false, "properties": map[string]any{
		"caption": map[string]any{"type": "string"}, "scene": map[string]any{"type": "object", "additionalProperties": false, "properties": map[string]any{"primary": map[string]any{"type": "string"}, "secondary": map[string]any{"type": "array", "items": map[string]any{"type": "string"}}, "environment": map[string]any{"type": "string"}}, "required": []string{"primary", "secondary", "environment"}},
		"suitePlace":    map[string]any{"type": "string", "enum": []string{"COVER", "PROLOGUE", "HEADER", "HERO", "STORY", "GRID", "QUOTE", "TRANSITION", "HIGHLIGHTS", "EPILOGUE", "CREDITS"}},
		"visual":        map[string]any{"type": "object", "additionalProperties": false, "properties": map[string]any{"timeOfDay": map[string]any{"type": "string"}, "weather": map[string]any{"type": "string"}, "lighting": map[string]any{"type": "string"}, "dominantMood": map[string]any{"type": "string"}}, "required": []string{"timeOfDay", "weather", "lighting", "dominantMood"}},
		"content":       map[string]any{"type": "object", "additionalProperties": false, "properties": map[string]any{"subjects": map[string]any{"type": "array", "items": map[string]any{"type": "string"}}, "peopleCount": map[string]any{"type": "integer"}}, "required": []string{"subjects", "peopleCount"}},
		"story":         map[string]any{"type": "object", "additionalProperties": false, "properties": map[string]any{"importance": map[string]any{"type": "number"}, "storyRole": map[string]any{"type": "string"}, "emotionalTone": map[string]any{"type": "string"}, "keywords": map[string]any{"type": "array", "items": map[string]any{"type": "string"}}}, "required": []string{"importance", "storyRole", "emotionalTone", "keywords"}},
		"quality":       map[string]any{"type": "object", "additionalProperties": false, "properties": map[string]any{"aestheticScore": map[string]any{"type": "number"}, "sharpnessScore": map[string]any{"type": "number"}, "exposureScore": map[string]any{"type": "number"}}, "required": []string{"aestheticScore", "sharpnessScore", "exposureScore"}},
		"composition":   map[string]any{"type": "object", "additionalProperties": false, "properties": map[string]any{"subjectPosition": map[string]any{"type": "string"}, "negativeSpace": map[string]any{"type": "array", "items": map[string]any{"type": "string"}}, "textSafeAreas": map[string]any{"type": "array", "items": map[string]any{"type": "object", "additionalProperties": false, "properties": map[string]any{"position": map[string]any{"type": "string"}, "confidence": map[string]any{"type": "number"}}, "required": []string{"position", "confidence"}}}}, "required": []string{"subjectPosition", "negativeSpace", "textSafeAreas"}},
		"locationGuess": map[string]any{"type": "object", "additionalProperties": false, "properties": map[string]any{"name": map[string]any{"type": "string"}, "confidence": map[string]any{"type": "number"}}, "required": []string{"name", "confidence"}},
		"memory":        memoryAnalysisSchema(),
	}, "required": []string{"caption", "scene", "suitePlace", "visual", "content", "story", "quality", "composition", "locationGuess", "memory"}}
}

func memoryAnalysisSchema() map[string]any {
	return map[string]any{"type": "object", "additionalProperties": false, "properties": map[string]any{
		"schemaVersion":       map[string]any{"type": "integer", "const": models.MemoryAnalysisVersion},
		"description":         map[string]any{"type": "string"},
		"objects":             map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		"readableText":        map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		"dominantColors":      map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "maxItems": 6},
		"visualImportance":    map[string]any{"type": "number", "minimum": 0, "maximum": 1},
		"photographicQuality": map[string]any{"type": "number", "minimum": 0, "maximum": 1},
		"storytellingValue":   map[string]any{"type": "number", "minimum": 0, "maximum": 1},
		"candidateRoles":      map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		"confidence":          map[string]any{"type": "number", "minimum": 0, "maximum": 1},
	}, "required": []string{"schemaVersion", "description", "objects", "readableText", "dominantColors", "visualImportance", "photographicQuality", "storytellingValue", "candidateRoles", "confidence"}}
}
