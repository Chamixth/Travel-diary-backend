package schemas

import _ "embed"

//go:embed story-spec-v1.schema.json
var StorySpecV1 []byte

//go:embed creative-exploration-v1.schema.json
var CreativeExplorationV1 []byte

//go:embed creative-direction-v1.schema.json
var CreativeDirectionV1 []byte

//go:embed experience-spec-v1.schema.json
var ExperienceSpecV1 []byte
