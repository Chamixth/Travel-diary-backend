package provider

import (
	"context"
	"encoding/json"
	"time"
)

// StructuredGenerator is the only capability generation agents depend on.
// Provider credentials, wire formats, retries, and usage accounting belong in
// adapters implementing this interface rather than in domain orchestration.
type StructuredGenerator interface {
	Generate(ctx context.Context, request StructuredRequest) (StructuredResponse, error)
}

type StructuredRequest struct {
	AgentID       string
	PromptVersion string
	Model         string
	SystemPrompt  string
	Input         json.RawMessage
	OutputSchema  json.RawMessage
	CorrelationID string
	Timeout       time.Duration
}

type StructuredResponse struct {
	Output            json.RawMessage
	Provider          string
	Model             string
	ProviderRequestID string
	InputTokens       int
	OutputTokens      int
	Latency           time.Duration
}
