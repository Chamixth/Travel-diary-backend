package experience

import (
	"encoding/json"
	"os"
	"testing"
)

func TestCanonicalSchemaMatchesGoContractVersion(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile("../../contracts/experience-spec-v1.schema.json")
	if err != nil {
		t.Fatalf("read canonical schema: %v", err)
	}
	var schema struct {
		Properties struct {
			SchemaVersion struct {
				Const string `json:"const"`
			} `json:"schemaVersion"`
		} `json:"properties"`
		Required []string `json:"required"`
	}
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("parse canonical schema: %v", err)
	}
	if schema.Properties.SchemaVersion.Const != SchemaVersionV1 {
		t.Fatalf("schema version %q does not match Go version %q", schema.Properties.SchemaVersion.Const, SchemaVersionV1)
	}
	if !containsString(schema.Required, "schemaVersion") || !containsString(schema.Required, "generatorVersion") {
		t.Fatal("canonical schema must require its schema and generator versions")
	}
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
