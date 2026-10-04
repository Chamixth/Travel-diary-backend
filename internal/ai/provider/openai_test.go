package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func TestOpenAIGenerateSendsStrictSchemaAndCapturesUsage(t *testing.T) {
	t.Parallel()
	httpClient := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.Header.Get("Authorization") != "Bearer secret" {
			t.Fatalf("missing bearer credential")
		}
		if request.Header.Get("X-Correlation-ID") != "job-1" {
			t.Fatalf("missing correlation id")
		}
		var body openAIRequest
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if !body.ResponseFormat.JSONSchema.Strict {
			t.Fatal("expected strict JSON schema")
		}
		if !strings.Contains(body.Messages[1].Content, "BEGIN_UNTRUSTED_STRUCTURED_INPUT") {
			t.Fatal("input was not delimited as untrusted")
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body: io.NopCloser(strings.NewReader(
				`{"id":"request-1","model":"model-1","choices":[{"message":{"content":"{\"result\":\"ok\"}"}}],"usage":{"prompt_tokens":12,"completion_tokens":4}}`,
			)),
		}, nil
	})}

	client := NewOpenAIWithEndpoint("secret", "https://provider.invalid/v1/chat/completions", httpClient)
	result, err := client.Generate(context.Background(), StructuredRequest{
		AgentID: "story-director", PromptVersion: "story-director-v1", Model: "model-1",
		SystemPrompt: "Direct the story.", Input: json.RawMessage(`{"trip":"one"}`),
		OutputSchema: json.RawMessage(`{"type":"object"}`), CorrelationID: "job-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ProviderRequestID != "request-1" || result.InputTokens != 12 || result.OutputTokens != 4 {
		t.Fatalf("usage metadata not captured: %#v", result)
	}
}

func TestOpenAIGenerateRejectsInvalidSchemaBeforeNetwork(t *testing.T) {
	t.Parallel()
	client := NewOpenAIWithEndpoint("secret", "http://invalid", nil)
	_, err := client.Generate(context.Background(), StructuredRequest{
		AgentID: "story", PromptVersion: "v1", Model: "model", OutputSchema: json.RawMessage(`not-json`),
	})
	if err == nil || !strings.Contains(err.Error(), "schema") {
		t.Fatalf("expected schema validation error, got %v", err)
	}
}
