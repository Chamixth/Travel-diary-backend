package models

import "time"

type GenerationJobStatus string
type GenerationStage string
type GenerationFailureClass string

const (
	GenerationJobQueued    GenerationJobStatus = "queued"
	GenerationJobRunning   GenerationJobStatus = "running"
	GenerationJobSucceeded GenerationJobStatus = "succeeded"
	GenerationJobFailed    GenerationJobStatus = "failed"

	GenerationFailureTransient GenerationFailureClass = "transient"
	GenerationFailurePermanent GenerationFailureClass = "permanent"

	GenerationStageCreated              GenerationStage = "created"
	GenerationStageClustering           GenerationStage = "clustering"
	GenerationStageStoryBuilding        GenerationStage = "story_building"
	GenerationStageConceptExploration   GenerationStage = "concept_exploration"
	GenerationStageCreativeDirection    GenerationStage = "creative_direction"
	GenerationStageExperienceGeneration GenerationStage = "experience_generation"
	GenerationStageValidating           GenerationStage = "validating"
	GenerationStageRenderReady          GenerationStage = "render_ready"
)

type GenerationStageEvent struct {
	Stage     GenerationStage `bson:"stage" json:"stage"`
	StartedAt time.Time       `bson:"startedAt" json:"startedAt"`
}

// GenerationJob is deliberately independent from the generated artifact. A
// retry creates another attempt while an already-persisted album remains safe.
// ActiveKey is present only while a job is queued/running and is uniquely
// indexed to make starting generation idempotent across application instances.
type GenerationJob struct {
	ID             string                 `bson:"_id,omitempty" json:"id"`
	UserID         string                 `bson:"userId" json:"-"`
	TripID         string                 `bson:"tripId" json:"tripId"`
	Scope          string                 `bson:"scope" json:"scope"`
	SourceRevision string                 `bson:"sourceRevision" json:"sourceRevision"`
	ActiveKey      string                 `bson:"activeKey,omitempty" json:"-"`
	CorrelationID  string                 `bson:"correlationId" json:"correlationId"`
	Status         GenerationJobStatus    `bson:"status" json:"status"`
	Stage          GenerationStage        `bson:"stage" json:"stage"`
	Attempt        int                    `bson:"attempt" json:"attempt"`
	MaxAttempts    int                    `bson:"maxAttempts" json:"maxAttempts"`
	History        []GenerationStageEvent `bson:"history" json:"history"`
	FailureCode    string                 `bson:"failureCode,omitempty" json:"failureCode,omitempty"`
	FailureMessage string                 `bson:"failureMessage,omitempty" json:"failureMessage,omitempty"`
	FailureClass   GenerationFailureClass `bson:"failureClass,omitempty" json:"failureClass,omitempty"`
	LastErrorCode  string                 `bson:"lastErrorCode,omitempty" json:"-"`
	LastErrorAt    *time.Time             `bson:"lastErrorAt,omitempty" json:"-"`
	NextAttemptAt  *time.Time             `bson:"nextAttemptAt,omitempty" json:"nextAttemptAt,omitempty"`
	LeaseOwner     string                 `bson:"leaseOwner,omitempty" json:"-"`
	LeaseUntil     *time.Time             `bson:"leaseUntil,omitempty" json:"-"`
	CreatedAt      time.Time              `bson:"createdAt" json:"createdAt"`
	StartedAt      *time.Time             `bson:"startedAt,omitempty" json:"startedAt,omitempty"`
	FinishedAt     *time.Time             `bson:"finishedAt,omitempty" json:"finishedAt,omitempty"`
	UpdatedAt      time.Time              `bson:"updatedAt" json:"updatedAt"`
}
