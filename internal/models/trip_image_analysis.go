package models

import "time"

type TripImageAnalysis struct {
	ID              string              `bson:"_id,omitempty" json:"id"`
	ImageID         string              `bson:"imageId" json:"imageId"`
	TripID          string              `bson:"tripId" json:"tripId"`
	UserID          string              `bson:"userId" json:"userId"`
	TakenAt         *time.Time          `bson:"takenAt,omitempty" json:"takenAt,omitempty"`
	Latitude        *float64            `bson:"latitude,omitempty" json:"latitude,omitempty"`
	Longitude       *float64            `bson:"longitude,omitempty" json:"longitude,omitempty"`
	SuitePlace      string              `bson:"suitePlace" json:"suitePlace"`
	Orientation     string              `bson:"orientation" json:"orientation"`
	AspectRatio     float64             `bson:"aspectRatio" json:"aspectRatio"`
	Caption         string              `bson:"caption" json:"caption"`
	Scene           SceneAnalysis       `bson:"scene" json:"scene"`
	Visual          VisualAnalysis      `bson:"visual" json:"visual"`
	Content         ContentAnalysis     `bson:"content" json:"content"`
	Story           StoryAnalysis       `bson:"story" json:"story"`
	Quality         QualityAnalysis     `bson:"quality" json:"quality"`
	Composition     CompositionAnalysis `bson:"composition" json:"composition"`
	LocationGuess   LocationGuess       `bson:"locationGuess" json:"locationGuess"`
	Memory          MemoryAnalysis      `bson:"memory" json:"memory"`
	Model           string              `bson:"model" json:"-"`
	AnalysisVersion int                 `bson:"analysisVersion" json:"analysisVersion"`
	PromptVersion   string              `bson:"promptVersion" json:"-"`
	CreatedAt       time.Time           `bson:"createdAt" json:"createdAt"`
	UpdatedAt       time.Time           `bson:"updatedAt" json:"updatedAt"`
}

const (
	SuitePlaceCover      = "COVER"
	SuitePlacePrologue   = "PROLOGUE"
	SuitePlaceHeader     = "HEADER"
	SuitePlaceHero       = "HERO"
	SuitePlaceStory      = "STORY"
	SuitePlaceGrid       = "GRID"
	SuitePlaceQuote      = "QUOTE"
	SuitePlaceTransition = "TRANSITION"
	SuitePlaceHighlights = "HIGHLIGHTS"
	SuitePlaceEpilogue   = "EPILOGUE"
	SuitePlaceCredits    = "CREDITS"
)

type SceneAnalysis struct {
	Primary     string   `bson:"primary" json:"primary"`
	Secondary   []string `bson:"secondary" json:"secondary"`
	Environment string   `bson:"environment" json:"environment"`
}
type VisualAnalysis struct {
	TimeOfDay    string `bson:"timeOfDay" json:"timeOfDay"`
	Weather      string `bson:"weather" json:"weather"`
	Lighting     string `bson:"lighting" json:"lighting"`
	DominantMood string `bson:"dominantMood" json:"dominantMood"`
}
type ContentAnalysis struct {
	Subjects    []string `bson:"subjects" json:"subjects"`
	PeopleCount int      `bson:"peopleCount" json:"peopleCount"`
}
type StoryAnalysis struct {
	Importance    float64  `bson:"importance" json:"importance"`
	StoryRole     string   `bson:"storyRole" json:"storyRole"`
	EmotionalTone string   `bson:"emotionalTone" json:"emotionalTone"`
	Keywords      []string `bson:"keywords" json:"keywords"`
}
type QualityAnalysis struct {
	AestheticScore float64 `bson:"aestheticScore" json:"aestheticScore"`
	SharpnessScore float64 `bson:"sharpnessScore" json:"sharpnessScore"`
	ExposureScore  float64 `bson:"exposureScore" json:"exposureScore"`
}
type CompositionAnalysis struct {
	SubjectPosition string         `bson:"subjectPosition" json:"subjectPosition"`
	NegativeSpace   []string       `bson:"negativeSpace" json:"negativeSpace"`
	TextSafeAreas   []TextSafeArea `bson:"textSafeAreas" json:"textSafeAreas"`
}
type TextSafeArea struct {
	Position   string  `bson:"position" json:"position"`
	Confidence float64 `bson:"confidence" json:"confidence"`
}
type LocationGuess struct {
	Name       string  `bson:"name" json:"name"`
	Confidence float64 `bson:"confidence" json:"confidence"`
}
