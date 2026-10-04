package experience

import "time"

const (
	SourceLegacyAlbumPlan   = "legacy-album-plan"
	SourceCreativeDirection = "creative-direction"
)

type Provenance struct {
	AgentID           string `bson:"agentId" json:"agentId"`
	PromptVersion     string `bson:"promptVersion" json:"promptVersion"`
	Provider          string `bson:"provider" json:"provider"`
	Model             string `bson:"model,omitempty" json:"model,omitempty"`
	ProviderRequestID string `bson:"providerRequestId,omitempty" json:"providerRequestId,omitempty"`
	InputTokens       int    `bson:"inputTokens,omitempty" json:"inputTokens,omitempty"`
	OutputTokens      int    `bson:"outputTokens,omitempty" json:"outputTokens,omitempty"`
	LatencyMS         int64  `bson:"latencyMs,omitempty" json:"latencyMs,omitempty"`
	FallbackReason    string `bson:"fallbackReason,omitempty" json:"fallbackReason,omitempty"`
}

// Version is an immutable, renderable experience snapshot. Activation is
// stored separately so rollback never mutates the snapshot itself.
type Version struct {
	ID               string      `bson:"_id" json:"id"`
	TripID           string      `bson:"tripId" json:"tripId"`
	UserID           string      `bson:"userId" json:"-"`
	Version          int64       `bson:"version" json:"version"`
	SchemaVersion    string      `bson:"schemaVersion" json:"schemaVersion"`
	GeneratorVersion string      `bson:"generatorVersion" json:"generatorVersion"`
	SourceType       string      `bson:"sourceType" json:"sourceType"`
	SourceID         string      `bson:"sourceId,omitempty" json:"sourceId,omitempty"`
	Spec             Spec        `bson:"spec" json:"spec"`
	Provenance       *Provenance `bson:"provenance,omitempty" json:"provenance,omitempty"`
	CreatedAt        time.Time   `bson:"createdAt" json:"createdAt"`
}

type Head struct {
	ID            string    `bson:"_id"`
	TripID        string    `bson:"tripId"`
	UserID        string    `bson:"userId"`
	NextVersion   int64     `bson:"nextVersion"`
	ActiveVersion int64     `bson:"activeVersion,omitempty"`
	ActiveID      string    `bson:"activeId,omitempty"`
	UpdatedAt     time.Time `bson:"updatedAt"`
}
