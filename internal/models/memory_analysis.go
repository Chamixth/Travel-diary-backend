package models

// MemoryAnalysisVersion identifies the additive memory-intelligence contract.
// It is kept beside the legacy TripImageAnalysis fields so existing albums can
// continue rendering while newer story and creative agents adopt richer data.
const MemoryAnalysisVersion = 1

type MemoryAnalysis struct {
	SchemaVersion       int      `bson:"schemaVersion" json:"schemaVersion"`
	Description         string   `bson:"description" json:"description"`
	Objects             []string `bson:"objects" json:"objects"`
	ReadableText        []string `bson:"readableText" json:"readableText"`
	DominantColors      []string `bson:"dominantColors" json:"dominantColors"`
	VisualImportance    float64  `bson:"visualImportance" json:"visualImportance"`
	PhotographicQuality float64  `bson:"photographicQuality" json:"photographicQuality"`
	StorytellingValue   float64  `bson:"storytellingValue" json:"storytellingValue"`
	CandidateRoles      []string `bson:"candidateRoles" json:"candidateRoles"`
	Confidence          float64  `bson:"confidence" json:"confidence"`
}
