package service

import (
	"context"
	"errors"
	"fmt"
	"time"
	"travel-diary-backend/internal/dao"
	"travel-diary-backend/internal/experience"
	"travel-diary-backend/internal/models"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/mongo"
)

type ExperienceService struct {
	trips    dao.TripDAO
	images   dao.TripImageDAO
	plans    dao.AlbumPlanDAO
	versions dao.ExperienceVersionDAO
}

func NewExperienceService(trips dao.TripDAO, images dao.TripImageDAO, plans dao.AlbumPlanDAO, versions dao.ExperienceVersionDAO) *ExperienceService {
	return &ExperienceService{trips: trips, images: images, plans: plans, versions: versions}
}

func (s *ExperienceService) GetActive(ctx context.Context, userID, tripID string) (experience.Version, error) {
	if _, err := s.trips.FindByIDAndUserID(ctx, tripID, userID); err != nil {
		return experience.Version{}, err
	}
	active, activeErr := s.versions.FindActive(ctx, userID, tripID)
	if activeErr == nil && active.SourceType != experience.SourceLegacyAlbumPlan {
		return active, nil
	}
	plan, err := s.plans.FindByTripID(ctx, userID, tripID)
	if err != nil {
		if activeErr == nil {
			return active, nil
		}
		return experience.Version{}, err
	}
	sourceID := legacySourceID(plan.ID, plan.UpdatedAt)
	if activeErr == nil && active.SourceType == experience.SourceLegacyAlbumPlan && active.SourceID == sourceID {
		return active, nil
	}
	return s.materializeLegacy(ctx, userID, tripID, sourceID, plan.Plan)
}

func (s *ExperienceService) MaterializeLegacy(ctx context.Context, userID, tripID string) (experience.Version, error) {
	if _, err := s.trips.FindByIDAndUserID(ctx, tripID, userID); err != nil {
		return experience.Version{}, err
	}
	plan, err := s.plans.FindByTripID(ctx, userID, tripID)
	if err != nil {
		return experience.Version{}, err
	}
	sourceID := legacySourceID(plan.ID, plan.UpdatedAt)
	if existing, err := s.versions.FindBySource(ctx, userID, tripID, experience.SourceLegacyAlbumPlan, sourceID); err == nil {
		return existing, nil
	} else if !errors.Is(err, mongo.ErrNoDocuments) {
		return experience.Version{}, err
	}
	return s.materializeLegacy(ctx, userID, tripID, sourceID, plan.Plan)
}

func (s *ExperienceService) PersistPlanned(
	ctx context.Context,
	userID, tripID, directionID string,
	spec experience.Spec,
	provenance experience.Provenance,
) (experience.Version, error) {
	if _, err := s.trips.FindByIDAndUserID(ctx, tripID, userID); err != nil {
		return experience.Version{}, err
	}
	if existing, err := s.versions.FindBySource(ctx, userID, tripID, experience.SourceCreativeDirection, directionID); err == nil {
		return s.versions.Activate(ctx, userID, tripID, existing.Version)
	} else if !errors.Is(err, mongo.ErrNoDocuments) {
		return experience.Version{}, err
	}
	if err := s.validateSpecAssets(ctx, userID, tripID, spec); err != nil {
		return experience.Version{}, err
	}
	return s.versions.CreateAndActivate(ctx, experience.Version{
		ID: uuid.NewString(), TripID: tripID, UserID: userID,
		SchemaVersion: spec.SchemaVersion, GeneratorVersion: spec.GeneratorVersion,
		SourceType: experience.SourceCreativeDirection, SourceID: directionID,
		Spec: spec, Provenance: &provenance,
	})
}

func (s *ExperienceService) ValidatePlannedSource(
	ctx context.Context, userID, tripID, directionID string,
) (experience.Version, error) {
	version, err := s.versions.FindBySource(ctx, userID, tripID, experience.SourceCreativeDirection, directionID)
	if err != nil {
		return experience.Version{}, err
	}
	if err := s.validateSpecAssets(ctx, userID, tripID, version.Spec); err != nil {
		return experience.Version{}, err
	}
	return version, nil
}

func (s *ExperienceService) ListVersions(ctx context.Context, userID, tripID string) ([]experience.Version, error) {
	if _, err := s.trips.FindByIDAndUserID(ctx, tripID, userID); err != nil {
		return nil, err
	}
	return s.versions.ListByTripID(ctx, userID, tripID, 25)
}

func (s *ExperienceService) ActivateVersion(ctx context.Context, userID, tripID string, versionNumber int64) (experience.Version, error) {
	if versionNumber < 1 {
		return experience.Version{}, fmt.Errorf("experience version must be positive")
	}
	if _, err := s.trips.FindByIDAndUserID(ctx, tripID, userID); err != nil {
		return experience.Version{}, err
	}
	return s.versions.Activate(ctx, userID, tripID, versionNumber)
}

func (s *ExperienceService) materializeLegacy(ctx context.Context, userID, tripID, sourceID string, plan models.AlbumPlan) (experience.Version, error) {
	images, err := s.images.ListByTripID(ctx, userID, tripID)
	if err != nil {
		return experience.Version{}, err
	}
	spec := experience.FromLegacyAlbum(uuid.NewString(), plan, images)
	knownAssets := make(map[string]struct{}, len(images))
	for _, image := range images {
		knownAssets[image.ID] = struct{}{}
	}
	if err := experience.Validate(spec, knownAssets, experience.V1Registry()); err != nil {
		return experience.Version{}, fmt.Errorf("adapt legacy album: %w", err)
	}
	return s.versions.CreateAndActivate(ctx, experience.Version{
		ID: uuid.NewString(), TripID: tripID, UserID: userID,
		SchemaVersion: spec.SchemaVersion, GeneratorVersion: spec.GeneratorVersion,
		SourceType: experience.SourceLegacyAlbumPlan, SourceID: sourceID, Spec: spec,
	})
}

func (s *ExperienceService) validateSpecAssets(ctx context.Context, userID, tripID string, spec experience.Spec) error {
	images, err := s.images.ListByTripID(ctx, userID, tripID)
	if err != nil {
		return err
	}
	knownAssets := make(map[string]struct{}, len(images))
	for _, image := range images {
		knownAssets[image.ID] = struct{}{}
	}
	if err := experience.Validate(spec, knownAssets, experience.V1Registry()); err != nil {
		return fmt.Errorf("validate planned experience: %w", err)
	}
	return nil
}

func legacySourceID(planID string, updatedAt time.Time) string {
	return fmt.Sprintf("%s:%d", planID, updatedAt.UTC().UnixNano())
}
