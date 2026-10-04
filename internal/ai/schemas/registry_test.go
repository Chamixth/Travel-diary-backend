package schemas

import (
	"encoding/json"
	"testing"
)

func TestEmbeddedSchemasAreValidJSON(t *testing.T) {
	t.Parallel()
	if !json.Valid(StorySpecV1) {
		t.Fatal("StorySpec v1 schema is not valid JSON")
	}
	if !json.Valid(CreativeExplorationV1) {
		t.Fatal("CreativeExploration v1 schema is not valid JSON")
	}
	if !json.Valid(CreativeDirectionV1) {
		t.Fatal("CreativeDirection v1 schema is not valid JSON")
	}
	if !json.Valid(ExperienceSpecV1) {
		t.Fatal("ExperienceSpec v1 AI schema is not valid JSON")
	}
}

func TestStorySchemaUsesStrictStructuredOutputObjects(t *testing.T) {
	t.Parallel()
	var schema map[string]any
	if err := json.Unmarshal(StorySpecV1, &schema); err != nil {
		t.Fatal(err)
	}
	assertStrictObjects(t, schema, "root")
	if err := json.Unmarshal(CreativeExplorationV1, &schema); err != nil {
		t.Fatal(err)
	}
	assertStrictObjects(t, schema, "creative-root")
	if err := json.Unmarshal(CreativeDirectionV1, &schema); err != nil {
		t.Fatal(err)
	}
	assertStrictObjects(t, schema, "direction-root")
	if err := json.Unmarshal(ExperienceSpecV1, &schema); err != nil {
		t.Fatal(err)
	}
	assertStrictObjects(t, schema, "experience-root")
}

func assertStrictObjects(t *testing.T, node any, path string) {
	t.Helper()
	object, ok := node.(map[string]any)
	if !ok {
		return
	}
	if properties, hasProperties := object["properties"].(map[string]any); hasProperties {
		if object["additionalProperties"] != false {
			t.Fatalf("%s must set additionalProperties to false", path)
		}
		requiredValues, _ := object["required"].([]any)
		required := make(map[string]struct{}, len(requiredValues))
		for _, value := range requiredValues {
			if name, ok := value.(string); ok {
				required[name] = struct{}{}
			}
		}
		for name := range properties {
			if _, ok := required[name]; !ok {
				t.Fatalf("%s.%s must be required for strict structured output", path, name)
			}
		}
	}
	for key, value := range object {
		switch nested := value.(type) {
		case map[string]any:
			assertStrictObjects(t, nested, path+"."+key)
		case []any:
			for _, item := range nested {
				assertStrictObjects(t, item, path+"."+key)
			}
		}
	}
}
