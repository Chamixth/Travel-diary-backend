package service

import (
	"context"
	"fmt"

	"travel-diary-backend/internal/dao"
	"travel-diary-backend/internal/integrations"
)

const MaxMediaBackfillBatch = 50

type MediaBackfillResult struct {
	Requested int      `json:"requested"`
	Processed int      `json:"processed"`
	Created   int      `json:"created"`
	Skipped   int      `json:"skipped"`
	Failed    int      `json:"failed"`
	Errors    []string `json:"errors,omitempty"`
	Remaining int      `json:"remaining"`
}

type MediaBackfillService struct {
	images dao.TripImageDAO
	assets dao.MediaAssetDAO
	s3     *integrations.S3Client
}

func NewMediaBackfillService(images dao.TripImageDAO, assets dao.MediaAssetDAO, s3 *integrations.S3Client) *MediaBackfillService {
	return &MediaBackfillService{images: images, assets: assets, s3: s3}
}

// BackfillTrip is idempotent and deliberately bounded. Existing assets are
// skipped, so callers can safely retry until Remaining reaches zero.
func (s *MediaBackfillService) BackfillTrip(ctx context.Context, userID, tripID string, limit int) (MediaBackfillResult, error) {
	if limit <= 0 || limit > MaxMediaBackfillBatch {
		limit = MaxMediaBackfillBatch
	}
	images, err := s.images.ListByTripID(ctx, userID, tripID)
	if err != nil {
		return MediaBackfillResult{}, err
	}
	existing, err := s.assets.ListByTripID(ctx, userID, tripID)
	if err != nil {
		return MediaBackfillResult{}, err
	}
	have := make(map[string]struct{}, len(existing))
	for _, asset := range existing {
		have[asset.ImageID] = struct{}{}
	}
	missing := 0
	for _, image := range images {
		if _, ok := have[image.ID]; !ok {
			missing++
		}
	}
	result := MediaBackfillResult{Requested: limit}
	for _, image := range images {
		if result.Processed >= limit {
			break
		}
		if _, ok := have[image.ID]; ok {
			result.Skipped++
			continue
		}
		result.Processed++
		if s.s3 == nil {
			result.Failed++
			result.Errors = append(result.Errors, image.ID+": storage unavailable")
			continue
		}
		data, getErr := s.s3.GetObject(ctx, image.S3Key)
		if getErr != nil {
			result.Failed++
			result.Errors = append(result.Errors, image.ID+": "+getErr.Error())
			continue
		}
		asset, buildErr := BuildMediaAssetMetadata(data, image.ID, tripID, userID, image.S3Key, image.ContentType, image.Width, image.Height)
		if buildErr != nil {
			result.Failed++
			result.Errors = append(result.Errors, image.ID+": "+buildErr.Error())
			continue
		}
		if err := s.assets.Upsert(ctx, asset); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, image.ID+": "+err.Error())
			continue
		}
		result.Created++
	}
	result.Remaining = missing - result.Created
	if result.Remaining < 0 {
		result.Remaining = 0
	}
	return result, nil
}

func (r MediaBackfillResult) Validate() error {
	if r.Requested < 1 || r.Requested > MaxMediaBackfillBatch {
		return fmt.Errorf("invalid backfill batch")
	}
	return nil
}
