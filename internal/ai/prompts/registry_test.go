package prompts

import (
	"strings"
	"testing"
)

func TestMemoryAnalystPromptIsVersionedAndTreatsImageTextAsData(t *testing.T) {
	t.Parallel()
	if MemoryAnalystV1.AgentID == "" || MemoryAnalystV1.Version == "" {
		t.Fatal("prompt identity and version are required")
	}
	if !strings.Contains(strings.ToLower(MemoryAnalystV1.Text), "untrusted") {
		t.Fatal("memory analyst prompt must explicitly distrust text found in photographs")
	}
}

func TestAllPromptsHaveUniqueVersionsAndDistrustUserContent(t *testing.T) {
	t.Parallel()
	seen := make(map[string]struct{}, len(V1))
	for _, prompt := range V1 {
		if prompt.AgentID == "" || prompt.Version == "" || strings.TrimSpace(prompt.Text) == "" {
			t.Fatalf("prompt is missing identity, version, or text: %#v", prompt)
		}
		if _, duplicate := seen[prompt.Version]; duplicate {
			t.Fatalf("duplicate prompt version %q", prompt.Version)
		}
		seen[prompt.Version] = struct{}{}
		if !strings.Contains(strings.ToLower(prompt.Text), "untrusted") {
			t.Fatalf("prompt %q does not mark user-derived content as untrusted", prompt.Version)
		}
	}
}
