package prompts

import _ "embed"

type Prompt struct {
	AgentID string
	Version string
	Text    string
}

//go:embed memory-analyst-v1.txt
var memoryAnalystV1Text string

//go:embed story-director-v1.txt
var storyDirectorV1Text string

//go:embed creative-explorer-v1.txt
var creativeExplorerV1Text string

//go:embed creative-director-v1.txt
var creativeDirectorV1Text string

//go:embed experience-planner-v1.txt
var experiencePlannerV1Text string

//go:embed creative-critic-v1.txt
var creativeCriticV1Text string

var MemoryAnalystV1 = Prompt{
	AgentID: "memory-analyst",
	Version: "memory-analyst-v1",
	Text:    memoryAnalystV1Text,
}

var StoryDirectorV1 = Prompt{AgentID: "story-director", Version: "story-director-v1", Text: storyDirectorV1Text}
var CreativeExplorerV1 = Prompt{AgentID: "creative-explorer", Version: "creative-explorer-v1", Text: creativeExplorerV1Text}
var CreativeDirectorV1 = Prompt{AgentID: "creative-director", Version: "creative-director-v1", Text: creativeDirectorV1Text}
var ExperiencePlannerV1 = Prompt{AgentID: "experience-planner", Version: "experience-planner-v1", Text: experiencePlannerV1Text}
var CreativeCriticV1 = Prompt{AgentID: "creative-critic", Version: "creative-critic-v1", Text: creativeCriticV1Text}

var V1 = []Prompt{
	MemoryAnalystV1,
	StoryDirectorV1,
	CreativeExplorerV1,
	CreativeDirectorV1,
	ExperiencePlannerV1,
	CreativeCriticV1,
}
