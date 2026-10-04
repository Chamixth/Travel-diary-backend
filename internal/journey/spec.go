package journey

import "time"

const SchemaVersionV1 = "1.0"

type Spec struct {
	SchemaVersion    string   `bson:"schemaVersion" json:"schemaVersion"`
	GeneratorVersion string   `bson:"generatorVersion" json:"generatorVersion"`
	TripID           string   `bson:"tripId" json:"tripId"`
	Moments          []Moment `bson:"moments" json:"moments"`
	UnplacedAssetIDs []string `bson:"unplacedAssetIds" json:"unplacedAssetIds"`
}

type Record struct {
	ID        string    `bson:"_id,omitempty" json:"id"`
	UserID    string    `bson:"userId" json:"-"`
	TripID    string    `bson:"tripId" json:"tripId"`
	Spec      Spec      `bson:"spec" json:"spec"`
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}

type Moment struct {
	ID               string    `bson:"id" json:"id"`
	Order            int       `bson:"order" json:"order"`
	Title            string    `bson:"title" json:"title"`
	StartAt          time.Time `bson:"startAt" json:"startAt"`
	EndAt            time.Time `bson:"endAt" json:"endAt"`
	AssetIDs         []string  `bson:"assetIds" json:"assetIds"`
	PrimaryScene     string    `bson:"primaryScene" json:"primaryScene"`
	Location         string    `bson:"location,omitempty" json:"location,omitempty"`
	EmotionalTone    string    `bson:"emotionalTone,omitempty" json:"emotionalTone,omitempty"`
	Confidence       float64   `bson:"confidence" json:"confidence"`
	BoundaryEvidence []string  `bson:"boundaryEvidence" json:"boundaryEvidence"`
	Latitude         *float64  `bson:"latitude,omitempty" json:"latitude,omitempty"`
	Longitude        *float64  `bson:"longitude,omitempty" json:"longitude,omitempty"`
	PerceptualHash   string    `bson:"perceptualHash,omitempty" json:"-"`
	ContentHash      string    `bson:"contentHash,omitempty" json:"-"`
}
