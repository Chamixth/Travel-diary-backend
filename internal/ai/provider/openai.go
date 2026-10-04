package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultOpenAIEndpoint = "https://api.openai.com/v1/chat/completions"

type OpenAI struct {
	key      string
	endpoint string
	client   *http.Client
}

func NewOpenAI(key string) *OpenAI {
	return &OpenAI{
		key: key, endpoint: defaultOpenAIEndpoint,
		client: &http.Client{Timeout: 120 * time.Second},
	}
}

// NewOpenAIWithEndpoint supports deterministic adapter tests and compatible
// gateways without leaking provider URL concerns into domain agents.
func NewOpenAIWithEndpoint(key, endpoint string, client *http.Client) *OpenAI {
	if client == nil {
		client = &http.Client{Timeout: 120 * time.Second}
	}
	return &OpenAI{key: key, endpoint: endpoint, client: client}
}

type openAIRequest struct {
	Model          string               `json:"model"`
	Messages       []openAIMessage      `json:"messages"`
	ResponseFormat openAIResponseFormat `json:"response_format"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponseFormat struct {
	Type       string           `json:"type"`
	JSONSchema openAIJSONSchema `json:"json_schema"`
}

type openAIJSONSchema struct {
	Name   string          `json:"name"`
	Strict bool            `json:"strict"`
	Schema json.RawMessage `json:"schema"`
}

type openAIResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

func (o *OpenAI) Generate(ctx context.Context, request StructuredRequest) (StructuredResponse, error) {
	if strings.TrimSpace(o.key) == "" {
		return StructuredResponse{}, fmt.Errorf("OpenAI API key is not configured")
	}
	if request.AgentID == "" || request.PromptVersion == "" || request.Model == "" {
		return StructuredResponse{}, fmt.Errorf("structured request requires agent, prompt version, and model")
	}
	if len(request.OutputSchema) == 0 || !json.Valid(request.OutputSchema) {
		return StructuredResponse{}, fmt.Errorf("structured request output schema is invalid")
	}
	if len(request.Input) > 0 && !json.Valid(request.Input) {
		return StructuredResponse{}, fmt.Errorf("structured request input is invalid JSON")
	}

	timeout := request.Timeout
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	requestContext, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	body := openAIRequest{
		Model: request.Model,
		Messages: []openAIMessage{
			{Role: "system", Content: request.SystemPrompt},
			{Role: "user", Content: "BEGIN_UNTRUSTED_STRUCTURED_INPUT\n" + string(request.Input) + "\nEND_UNTRUSTED_STRUCTURED_INPUT"},
		},
		ResponseFormat: openAIResponseFormat{Type: "json_schema", JSONSchema: openAIJSONSchema{
			Name:   request.AgentID + "_" + strings.ReplaceAll(request.PromptVersion, "-", "_"),
			Strict: true, Schema: request.OutputSchema,
		}},
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return StructuredResponse{}, err
	}
	httpRequest, err := http.NewRequestWithContext(requestContext, http.MethodPost, o.endpoint, bytes.NewReader(encoded))
	if err != nil {
		return StructuredResponse{}, err
	}
	httpRequest.Header.Set("Authorization", "Bearer "+o.key)
	httpRequest.Header.Set("Content-Type", "application/json")
	if request.CorrelationID != "" {
		httpRequest.Header.Set("X-Correlation-ID", request.CorrelationID)
	}

	started := time.Now()
	response, err := o.client.Do(httpRequest)
	latency := time.Since(started)
	if err != nil {
		return StructuredResponse{}, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		message, _ := io.ReadAll(io.LimitReader(response.Body, 2048))
		return StructuredResponse{}, fmt.Errorf("OpenAI structured request failed with status %d: %s", response.StatusCode, strings.TrimSpace(string(message)))
	}

	var decoded openAIResponse
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return StructuredResponse{}, fmt.Errorf("decode OpenAI structured response: %w", err)
	}
	if len(decoded.Choices) == 0 {
		return StructuredResponse{}, fmt.Errorf("OpenAI returned no structured response choices")
	}
	output := json.RawMessage(decoded.Choices[0].Message.Content)
	if !json.Valid(output) {
		return StructuredResponse{}, fmt.Errorf("OpenAI returned invalid structured JSON")
	}
	return StructuredResponse{
		Output: output, Provider: "openai", Model: decoded.Model,
		ProviderRequestID: decoded.ID, InputTokens: decoded.Usage.PromptTokens,
		OutputTokens: decoded.Usage.CompletionTokens, Latency: latency,
	}, nil
}
