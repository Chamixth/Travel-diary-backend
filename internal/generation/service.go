package generation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"travel-diary-backend/internal/ai/agents"
	"travel-diary-backend/internal/ai/provider"
	"travel-diary-backend/internal/config"
	"travel-diary-backend/internal/creative"
	"travel-diary-backend/internal/dao"
	"travel-diary-backend/internal/experience"
	"travel-diary-backend/internal/journey"
	"travel-diary-backend/internal/models"
	legacyservice "travel-diary-backend/internal/service"
	"travel-diary-backend/internal/story"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/mongo"
)

const (
	defaultScope       = "whole-experience"
	leaseLength        = 90 * time.Second
	heartbeatInterval  = 30 * time.Second
	executionTimeout   = 12 * time.Minute
	defaultMaxAttempts = 3
)

var ErrImagesNotReady = errors.New("trip photographs are not ready for generation")

type Service struct {
	jobs        dao.GenerationJobDAO
	trips       dao.TripDAO
	images      dao.TripImageDAO
	wake        chan struct{}
	maxAttempts int
}

func NewService(db *mongo.Database, wake chan struct{}, maxAttempts int) *Service {
	return &Service{
		jobs: dao.NewGenerationJobDAO(db), trips: dao.NewTripDAO(db),
		images: dao.NewTripImageDAO(db), wake: wake, maxAttempts: normalizedMaxAttempts(maxAttempts),
	}
}

func (s *Service) Start(ctx context.Context, userID, tripID string) (models.GenerationJob, bool, error) {
	if _, err := s.trips.FindByIDAndUserID(ctx, tripID, userID); err != nil {
		return models.GenerationJob{}, false, err
	}
	images, err := s.images.ListByTripID(ctx, userID, tripID)
	if err != nil {
		return models.GenerationJob{}, false, err
	}
	if len(images) == 0 {
		return models.GenerationJob{}, false, ErrImagesNotReady
	}
	for _, image := range images {
		if image.AnalysisStatus != "ANALYZED" && image.AnalysisStatus != "FAILED" {
			return models.GenerationJob{}, false, ErrImagesNotReady
		}
	}

	now := time.Now().UTC()
	revision := sourceRevision(images)
	activeKey := strings.Join([]string{userID, tripID, defaultScope, revision}, ":")
	job := models.GenerationJob{
		ID: uuid.NewString(), UserID: userID, TripID: tripID, Scope: defaultScope,
		SourceRevision: revision, ActiveKey: activeKey, CorrelationID: uuid.NewString(),
		Status: models.GenerationJobQueued, Stage: models.GenerationStageCreated,
		MaxAttempts: s.maxAttempts,
		History:     []models.GenerationStageEvent{{Stage: models.GenerationStageCreated, StartedAt: now}},
		CreatedAt:   now, UpdatedAt: now,
	}
	job, created, err := s.jobs.CreateOrGetActive(ctx, job)
	if err != nil {
		return models.GenerationJob{}, false, err
	}
	if created {
		select {
		case s.wake <- struct{}{}:
		default:
		}
	}
	return job, created, nil
}

func (s *Service) Get(ctx context.Context, userID, tripID, jobID string) (models.GenerationJob, error) {
	return s.jobs.FindByIDAndUserID(ctx, jobID, userID, tripID)
}

func sourceRevision(images []models.TripImage) string {
	parts := make([]string, 0, len(images))
	for _, image := range images {
		parts = append(parts, fmt.Sprintf("%s:%d:%d", image.ID, image.UpdatedAt.UnixNano(), image.AnalysisVersion))
	}
	sort.Strings(parts)
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:12])
}

type Queue struct {
	jobs                    dao.GenerationJobDAO
	album                   *legacyservice.AlbumService
	images                  dao.TripImageDAO
	analyses                dao.TripImageAnalysisDAO
	mediaAssets             dao.MediaAssetDAO
	journeys                dao.JourneyDAO
	stories                 dao.StoryDAO
	creative                dao.CreativeDAO
	trips                   dao.TripDAO
	experience              *legacyservice.ExperienceService
	storyDirector           storyDirectorPort
	creativeExplorer        creativeExplorerPort
	creativeDirector        creativeDirectorPort
	experiencePlanner       experiencePlannerPort
	creativePipelineEnabled bool
	wake                    chan struct{}
	workers                 int
	maxAttempts             int
	workerID                string
	claimContext            context.Context
	stopClaims              context.CancelFunc
	executionContext        context.Context
	cancelExecutions        context.CancelFunc
	workerGroup             sync.WaitGroup
	startOnce               sync.Once
	shutdownOnce            sync.Once
	shutdownDone            chan struct{}
}

func NewQueue(cfg config.Config, db *mongo.Database) *Queue {
	workers := cfg.GenerationWorkers
	if workers < 1 {
		workers = 1
	}
	claimContext, stopClaims := context.WithCancel(context.Background())
	executionContext, cancelExecutions := context.WithCancel(context.Background())
	return &Queue{
		jobs: dao.NewGenerationJobDAO(db), images: dao.NewTripImageDAO(db),
		analyses: dao.NewTripImageAnalysisDAO(db), mediaAssets: dao.NewMediaAssetDAO(db), journeys: dao.NewJourneyDAO(db),
		stories: dao.NewStoryDAO(db), creative: dao.NewCreativeDAO(db), trips: dao.NewTripDAO(db),
		album: legacyservice.NewAlbumService(
			dao.NewTripDAO(db), dao.NewTripImageDAO(db), dao.NewTripImageAnalysisDAO(db),
			dao.NewAlbumPlanDAO(db), cfg,
		),
		experience: legacyservice.NewExperienceService(
			dao.NewTripDAO(db), dao.NewTripImageDAO(db), dao.NewAlbumPlanDAO(db), dao.NewExperienceVersionDAO(db),
		),
		storyDirector:           agents.NewStoryDirector(provider.NewOpenAI(cfg.OpenAIKey), cfg.OpenAIModelCreative),
		creativeExplorer:        agents.NewCreativeExplorer(provider.NewOpenAI(cfg.OpenAIKey), cfg.OpenAIModelCreative),
		creativeDirector:        agents.NewCreativeDirector(provider.NewOpenAI(cfg.OpenAIKey), cfg.OpenAIModelCreative),
		experiencePlanner:       agents.NewExperiencePlanner(provider.NewOpenAI(cfg.OpenAIKey), cfg.OpenAIModelCreative),
		creativePipelineEnabled: cfg.CreativePipelineEnabled,
		wake:                    make(chan struct{}, 1), workers: workers,
		maxAttempts: normalizedMaxAttempts(cfg.GenerationMaxAttempts), workerID: uuid.NewString(),
		claimContext: claimContext, stopClaims: stopClaims,
		executionContext: executionContext, cancelExecutions: cancelExecutions,
		shutdownDone: make(chan struct{}),
	}
}

func (q *Queue) Start() {
	q.startOnce.Do(func() {
		for worker := 0; worker < q.workers; worker++ {
			q.workerGroup.Add(1)
			go q.runWorker(fmt.Sprintf("%s-%d", q.workerID, worker+1))
		}
		q.Notify()
	})
}

func (q *Queue) Notify() {
	select {
	case q.wake <- struct{}{}:
	default:
	}
}

func (q *Queue) Service(db *mongo.Database) *Service {
	return NewService(db, q.wake, q.maxAttempts)
}

func normalizedMaxAttempts(value int) int {
	if value < 1 || value > 10 {
		return defaultMaxAttempts
	}
	return value
}

func (q *Queue) Shutdown(ctx context.Context) error {
	q.shutdownOnce.Do(func() {
		q.stopClaims()
		q.Notify()
		go func() {
			q.workerGroup.Wait()
			q.cancelExecutions()
			close(q.shutdownDone)
		}()
	})
	select {
	case <-q.shutdownDone:
		return nil
	case <-ctx.Done():
		q.cancelExecutions()
		// Give canceled workers a short opportunity to return their leases
		// before infrastructure clients are closed by the application hook.
		forceTimer := time.NewTimer(5 * time.Second)
		defer forceTimer.Stop()
		select {
		case <-q.shutdownDone:
		case <-forceTimer.C:
		}
		return ctx.Err()
	}
}

func (q *Queue) runWorker(workerID string) {
	defer q.workerGroup.Done()
	for {
		if q.claimContext.Err() != nil {
			return
		}
		if expired, err := q.jobs.FailExpiredAttempts(q.claimContext, q.maxAttempts); err != nil && !errors.Is(err, context.Canceled) {
			log.Printf(`{"event":"generation_retry_sweep_failed","workerId":%q,"error":%q}`, workerID, err.Error())
		} else if expired > 0 {
			log.Printf(`{"event":"generation_retries_exhausted","workerId":%q,"jobs":%d}`, workerID, expired)
		}
		job, err := q.jobs.ClaimNext(q.claimContext, workerID, leaseLength, q.maxAttempts)
		if errors.Is(err, mongo.ErrNoDocuments) {
			select {
			case <-q.claimContext.Done():
				return
			case <-q.wake:
			case <-time.After(2 * time.Second):
			}
			continue
		}
		if errors.Is(err, context.Canceled) {
			return
		}
		if err != nil {
			log.Printf(`{"event":"generation_claim_failed","workerId":%q,"error":%q}`, workerID, err.Error())
			select {
			case <-q.claimContext.Done():
				return
			case <-time.After(time.Second):
			}
			continue
		}
		q.execute(q.executionContext, workerID, job)
	}
}

func (q *Queue) execute(parent context.Context, workerID string, job models.GenerationJob) {
	ctx, cancel := context.WithTimeout(parent, executionTimeout)
	heartbeatContext, stopHeartbeatContext := context.WithCancel(ctx)
	heartbeatDone := make(chan struct{})
	go q.maintainLease(heartbeatContext, cancel, heartbeatDone, workerID, job)
	var stopHeartbeatOnce sync.Once
	stopHeartbeat := func() {
		stopHeartbeatOnce.Do(func() {
			stopHeartbeatContext()
			<-heartbeatDone
		})
	}
	defer func() {
		stopHeartbeat()
		cancel()
	}()
	started := time.Now()
	log.Printf(`{"event":"generation_started","jobId":%q,"tripId":%q,"correlationId":%q,"attempt":%d}`,
		job.ID, job.TripID, job.CorrelationID, job.Attempt)

	fail := func(code string, err error) {
		stopHeartbeat()
		writeContext, writeCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer writeCancel()
		if parent.Err() != nil {
			if releaseErr := q.jobs.Release(writeContext, job.ID, workerID); releaseErr != nil && !errors.Is(releaseErr, dao.ErrGenerationLeaseLost) {
				log.Printf(`{"event":"generation_release_failed","jobId":%q,"correlationId":%q,"error":%q}`,
					job.ID, job.CorrelationID, releaseErr.Error())
			}
			log.Printf(`{"event":"generation_interrupted","jobId":%q,"correlationId":%q,"stage":%q,"reason":"worker shutdown"}`,
				job.ID, job.CorrelationID, job.Stage)
			return
		}
		class := ClassifyFailure(code, err)
		maxAttempts := job.MaxAttempts
		if maxAttempts < 1 {
			maxAttempts = q.maxAttempts
		}
		if class == models.GenerationFailureTransient && job.Attempt < maxAttempts {
			delay := RetryDelay(job.Attempt)
			nextAttempt := time.Now().UTC().Add(delay)
			if retryErr := q.jobs.Retry(writeContext, job.ID, workerID, code, err.Error(), nextAttempt); retryErr == nil {
				q.Notify()
				log.Printf(`{"event":"generation_retry_scheduled","jobId":%q,"correlationId":%q,"stage":%q,"attempt":%d,"maxAttempts":%d,"delayMs":%d,"failureCode":%q}`,
					job.ID, job.CorrelationID, job.Stage, job.Attempt, maxAttempts, delay.Milliseconds(), code)
				return
			} else if !errors.Is(retryErr, dao.ErrGenerationLeaseLost) {
				log.Printf(`{"event":"generation_retry_schedule_failed","jobId":%q,"correlationId":%q,"error":%q}`,
					job.ID, job.CorrelationID, retryErr.Error())
			}
		}
		if failErr := q.jobs.Fail(writeContext, job.ID, workerID, code, err.Error(), class); failErr != nil && !errors.Is(failErr, dao.ErrGenerationLeaseLost) {
			log.Printf(`{"event":"generation_failure_write_failed","jobId":%q,"correlationId":%q,"error":%q}`,
				job.ID, job.CorrelationID, failErr.Error())
		}
		log.Printf(`{"event":"generation_failed","jobId":%q,"correlationId":%q,"stage":%q,"attempt":%d,"maxAttempts":%d,"failureClass":%q,"failureCode":%q,"durationMs":%d,"error":%q}`,
			job.ID, job.CorrelationID, job.Stage, job.Attempt, maxAttempts, class, code,
			time.Since(started).Milliseconds(), err.Error())
	}

	if job.Stage == models.GenerationStageCreated {
		if err := ValidateStageTransition(job.Stage, models.GenerationStageClustering); err != nil {
			fail("invalid_stage_transition", err)
			return
		}
		if err := q.jobs.Advance(ctx, job.ID, workerID, job.Stage, models.GenerationStageClustering); err != nil {
			fail("job_update_failed", err)
			return
		}
		job.Stage = models.GenerationStageClustering
	}

	if job.Stage == models.GenerationStageClustering {
		images, err := q.images.ListByTripID(ctx, job.UserID, job.TripID)
		if err != nil {
			fail("journey_images_failed", err)
			return
		}
		analyses, err := q.analyses.ListByTripID(ctx, job.UserID, job.TripID)
		if err != nil {
			fail("journey_analyses_failed", err)
			return
		}
		assets, mediaErr := q.mediaAssets.ListByTripID(ctx, job.UserID, job.TripID)
		if mediaErr != nil {
			fail("journey_media_failed", mediaErr)
			return
		}
		journeySpec := journey.ReconstructWithMedia(job.TripID, images, analyses, assets)
		if _, err := q.journeys.Upsert(ctx, job.UserID, job.TripID, journeySpec); err != nil {
			fail("journey_persistence_failed", err)
			return
		}
		if err := q.jobs.Advance(ctx, job.ID, workerID, job.Stage, models.GenerationStageStoryBuilding); err != nil {
			fail("job_update_failed", err)
			return
		}
		job.Stage = models.GenerationStageStoryBuilding
	}

	if job.Stage == models.GenerationStageStoryBuilding {
		trip, err := q.trips.FindByIDAndUserID(ctx, job.TripID, job.UserID)
		if err != nil {
			fail("story_trip_failed", err)
			return
		}
		journeyRecord, err := q.journeys.FindByTripID(ctx, job.UserID, job.TripID)
		if err != nil {
			fail("story_journey_failed", err)
			return
		}
		analyses, err := q.analyses.ListByTripID(ctx, job.UserID, job.TripID)
		if err != nil {
			fail("story_analyses_failed", err)
			return
		}
		if len(journeyRecord.Spec.Moments) > 0 {
			knownMoments := make(map[string]struct{}, len(journeyRecord.Spec.Moments))
			knownAssets := make(map[string]struct{})
			for _, moment := range journeyRecord.Spec.Moments {
				knownMoments[moment.ID] = struct{}{}
				for _, assetID := range moment.AssetIDs {
					knownAssets[assetID] = struct{}{}
				}
			}
			for _, assetID := range journeyRecord.Spec.UnplacedAssetIDs {
				knownAssets[assetID] = struct{}{}
			}
			storySpec, provenance, generationErr := buildStory(
				ctx, q.creativePipelineEnabled, q.storyDirector, trip, journeyRecord.Spec,
				analyses, knownMoments, knownAssets, job.CorrelationID,
			)
			if generationErr != nil {
				log.Printf(`{"event":"story_director_fallback","jobId":%q,"correlationId":%q,"error":%q}`,
					job.ID, job.CorrelationID, generationErr.Error())
			} else if provenance.Provider != "deterministic" {
				log.Printf(`{"event":"story_director_completed","jobId":%q,"correlationId":%q,"provider":%q,"model":%q,"latencyMs":%d,"inputTokens":%d,"outputTokens":%d}`,
					job.ID, job.CorrelationID, provenance.Provider, provenance.Model, provenance.LatencyMS,
					provenance.InputTokens, provenance.OutputTokens)
			}
			if err := story.Validate(storySpec, knownMoments, knownAssets); err != nil {
				fail("story_validation_failed", err)
				return
			}
			if _, err := q.stories.CreateOrGet(ctx, job.UserID, job.TripID, job.SourceRevision, storySpec, provenance); err != nil {
				fail("story_persistence_failed", err)
				return
			}
		} else {
			log.Printf(`{"event":"story_skipped","jobId":%q,"correlationId":%q,"reason":"no analyzed moments"}`, job.ID, job.CorrelationID)
		}
		if err := q.jobs.Advance(ctx, job.ID, workerID, job.Stage, models.GenerationStageConceptExploration); err != nil {
			fail("job_update_failed", err)
			return
		}
		job.Stage = models.GenerationStageConceptExploration
	}

	if job.Stage == models.GenerationStageConceptExploration {
		storyRecord, err := q.stories.FindLatestBySourceRevision(ctx, job.UserID, job.TripID, job.SourceRevision)
		if errors.Is(err, mongo.ErrNoDocuments) {
			log.Printf(`{"event":"concept_exploration_skipped","jobId":%q,"correlationId":%q,"reason":"no story artifact"}`,
				job.ID, job.CorrelationID)
		} else if err != nil {
			fail("creative_story_failed", err)
			return
		} else {
			journeyRecord, err := q.journeys.FindByTripID(ctx, job.UserID, job.TripID)
			if err != nil {
				fail("creative_journey_failed", err)
				return
			}
			analyses, err := q.analyses.ListByTripID(ctx, job.UserID, job.TripID)
			if err != nil {
				fail("creative_analyses_failed", err)
				return
			}
			fingerprints, err := q.creative.ListRecentFingerprints(ctx, job.UserID, job.TripID, 12)
			if err != nil {
				fail("creative_fingerprints_failed", err)
				return
			}
			exploration, provenance, generationErr := buildExploration(
				ctx, q.creativePipelineEnabled, q.creativeExplorer, storyRecord.Spec,
				journeyRecord.Spec, analyses, fingerprints, job.CorrelationID,
			)
			if generationErr != nil {
				log.Printf(`{"event":"creative_explorer_fallback","jobId":%q,"correlationId":%q,"error":%q}`,
					job.ID, job.CorrelationID, generationErr.Error())
			} else if provenance.Provider != "deterministic" {
				log.Printf(`{"event":"creative_explorer_completed","jobId":%q,"correlationId":%q,"provider":%q,"model":%q,"concepts":%d,"latencyMs":%d,"inputTokens":%d,"outputTokens":%d}`,
					job.ID, job.CorrelationID, provenance.Provider, provenance.Model, len(exploration.Concepts),
					provenance.LatencyMS, provenance.InputTokens, provenance.OutputTokens)
			}
			if err := creative.ValidateExploration(exploration); err != nil {
				fail("creative_exploration_validation_failed", err)
				return
			}
			if _, err := q.creative.CreateOrGetExploration(
				ctx, job.UserID, job.TripID, job.SourceRevision, exploration, fingerprints, provenance,
			); err != nil {
				fail("creative_exploration_persistence_failed", err)
				return
			}
		}
		if err := q.jobs.Advance(ctx, job.ID, workerID, job.Stage, models.GenerationStageCreativeDirection); err != nil {
			fail("job_update_failed", err)
			return
		}
		job.Stage = models.GenerationStageCreativeDirection
	}

	if job.Stage == models.GenerationStageCreativeDirection {
		storyRecord, storyErr := q.stories.FindLatestBySourceRevision(ctx, job.UserID, job.TripID, job.SourceRevision)
		explorationRecord, explorationErr := q.creative.FindLatestExplorationBySourceRevision(ctx, job.UserID, job.TripID, job.SourceRevision)
		if errors.Is(storyErr, mongo.ErrNoDocuments) || errors.Is(explorationErr, mongo.ErrNoDocuments) {
			log.Printf(`{"event":"creative_direction_skipped","jobId":%q,"correlationId":%q,"reason":"missing story or exploration artifact"}`,
				job.ID, job.CorrelationID)
		} else if storyErr != nil {
			fail("direction_story_failed", storyErr)
			return
		} else if explorationErr != nil {
			fail("direction_exploration_failed", explorationErr)
			return
		} else {
			analyses, err := q.analyses.ListByTripID(ctx, job.UserID, job.TripID)
			if err != nil {
				fail("direction_analyses_failed", err)
				return
			}
			knownAssetIDs := story.IncludedAssetIDs(storyRecord.Spec)
			fingerprints, err := q.creative.ListRecentFingerprints(ctx, job.UserID, job.TripID, 12)
			if err != nil {
				fail("direction_fingerprints_failed", err)
				return
			}
			direction, provenance, generationErr := buildDirection(
				ctx, q.creativePipelineEnabled, q.creativeDirector, storyRecord.Spec,
				explorationRecord.Exploration, analyses, fingerprints, knownAssetIDs, job.CorrelationID,
			)
			if generationErr != nil {
				log.Printf(`{"event":"creative_director_fallback","jobId":%q,"correlationId":%q,"error":%q}`,
					job.ID, job.CorrelationID, generationErr.Error())
			} else if provenance.Provider != "deterministic" {
				log.Printf(`{"event":"creative_director_completed","jobId":%q,"correlationId":%q,"provider":%q,"model":%q,"selectedConceptId":%q,"latencyMs":%d,"inputTokens":%d,"outputTokens":%d}`,
					job.ID, job.CorrelationID, provenance.Provider, provenance.Model, direction.SelectedConceptID,
					provenance.LatencyMS, provenance.InputTokens, provenance.OutputTokens)
			}
			if err := creative.ValidateDirection(direction, explorationRecord.Exploration, knownAssetIDs); err != nil {
				fail("creative_direction_validation_failed", err)
				return
			}
			fingerprint := creative.FingerprintFromDirection(direction, storyRecord.Spec)
			if _, err := q.creative.CreateOrGetDirection(
				ctx, job.UserID, job.TripID, job.SourceRevision, explorationRecord.ID,
				direction, fingerprint, provenance,
			); err != nil {
				fail("creative_direction_persistence_failed", err)
				return
			}
		}
		if err := q.jobs.Advance(ctx, job.ID, workerID, job.Stage, models.GenerationStageExperienceGeneration); err != nil {
			fail("job_update_failed", err)
			return
		}
		job.Stage = models.GenerationStageExperienceGeneration
	}

	if job.Stage == models.GenerationStageExperienceGeneration {
		storyRecord, storyErr := q.stories.FindLatestBySourceRevision(ctx, job.UserID, job.TripID, job.SourceRevision)
		directionRecord, directionErr := q.creative.FindLatestDirectionBySourceRevision(ctx, job.UserID, job.TripID, job.SourceRevision)
		if errors.Is(storyErr, mongo.ErrNoDocuments) || errors.Is(directionErr, mongo.ErrNoDocuments) {
			// Compatibility path for old or analysis-empty jobs that entered the
			// lifecycle before native story artifacts could be created.
			if _, err := q.album.Generate(ctx, job.UserID, job.TripID); err != nil {
				fail("legacy_album_generation_failed", err)
				return
			}
		} else if storyErr != nil {
			fail("experience_story_failed", storyErr)
			return
		} else if directionErr != nil {
			fail("experience_direction_failed", directionErr)
			return
		} else {
			images, err := q.images.ListByTripID(ctx, job.UserID, job.TripID)
			if err != nil {
				fail("experience_images_failed", err)
				return
			}
			analyses, err := q.analyses.ListByTripID(ctx, job.UserID, job.TripID)
			if err != nil {
				fail("experience_analyses_failed", err)
				return
			}
			knownAssetIDs := make(map[string]struct{}, len(images))
			for _, image := range images {
				knownAssetIDs[image.ID] = struct{}{}
			}
			spec, provenance, generationErr := buildExperience(
				ctx, q.creativePipelineEnabled, q.experiencePlanner, uuid.NewString(),
				storyRecord.Spec, directionRecord.Direction, images, analyses, knownAssetIDs, job.CorrelationID,
			)
			if generationErr != nil {
				log.Printf(`{"event":"experience_planner_fallback","jobId":%q,"correlationId":%q,"error":%q}`,
					job.ID, job.CorrelationID, generationErr.Error())
			} else if provenance.Provider != "deterministic" {
				log.Printf(`{"event":"experience_planner_completed","jobId":%q,"correlationId":%q,"provider":%q,"model":%q,"scenes":%d,"latencyMs":%d,"inputTokens":%d,"outputTokens":%d}`,
					job.ID, job.CorrelationID, provenance.Provider, provenance.Model, len(spec.Scenes),
					provenance.LatencyMS, provenance.InputTokens, provenance.OutputTokens)
			}
			if err := experience.Validate(spec, knownAssetIDs, experience.V1Registry()); err != nil {
				fail("experience_spec_validation_failed", err)
				return
			}
			if err := experience.ValidatePlanned(spec, storyRecord.Spec); err != nil {
				fail("experience_story_coverage_failed", err)
				return
			}
			if _, err := q.experience.PersistPlanned(
				ctx, job.UserID, job.TripID, directionRecord.ID, spec, provenance,
			); err != nil {
				fail("experience_persistence_failed", err)
				return
			}
		}
		if err := q.jobs.Advance(ctx, job.ID, workerID, job.Stage, models.GenerationStageValidating); err != nil {
			fail("job_update_failed", err)
			return
		}
		job.Stage = models.GenerationStageValidating
	}

	if job.Stage == models.GenerationStageValidating {
		// Re-read the immutable artifact so validation also covers BSON
		// serialization and retry/resume behavior.
		directionRecord, directionErr := q.creative.FindLatestDirectionBySourceRevision(ctx, job.UserID, job.TripID, job.SourceRevision)
		if directionErr == nil {
			if _, err := q.experience.ValidatePlannedSource(ctx, job.UserID, job.TripID, directionRecord.ID); errors.Is(err, mongo.ErrNoDocuments) {
				// A job may have crossed this deployment boundary after producing a
				// legacy plan but before native planning was introduced.
				if _, legacyErr := q.experience.MaterializeLegacy(ctx, job.UserID, job.TripID); legacyErr != nil {
					fail("experience_validation_failed", legacyErr)
					return
				}
			} else if err != nil {
				fail("experience_validation_failed", err)
				return
			}
		} else if errors.Is(directionErr, mongo.ErrNoDocuments) {
			if _, err := q.experience.MaterializeLegacy(ctx, job.UserID, job.TripID); err != nil {
				fail("experience_validation_failed", err)
				return
			}
		} else {
			fail("experience_direction_lookup_failed", directionErr)
			return
		}
		if err := q.jobs.Advance(ctx, job.ID, workerID, job.Stage, models.GenerationStageRenderReady); err != nil {
			fail("job_update_failed", err)
			return
		}
		job.Stage = models.GenerationStageRenderReady
	}

	stopHeartbeat()
	if err := q.jobs.Complete(ctx, job.ID, workerID); err != nil {
		fail("job_completion_failed", err)
		return
	}
	log.Printf(`{"event":"generation_completed","jobId":%q,"tripId":%q,"correlationId":%q,"durationMs":%d}`,
		job.ID, job.TripID, job.CorrelationID, time.Since(started).Milliseconds())
}

func (q *Queue) maintainLease(
	ctx context.Context,
	cancelExecution context.CancelFunc,
	done chan<- struct{},
	workerID string,
	job models.GenerationJob,
) {
	defer close(done)
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()
	consecutiveFailures := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			err := q.jobs.Heartbeat(ctx, job.ID, workerID, leaseLength)
			if err == nil {
				consecutiveFailures = 0
				continue
			}
			if errors.Is(err, dao.ErrGenerationLeaseLost) {
				log.Printf(`{"event":"generation_lease_lost","jobId":%q,"correlationId":%q,"workerId":%q}`,
					job.ID, job.CorrelationID, workerID)
				cancelExecution()
				return
			}
			consecutiveFailures++
			log.Printf(`{"event":"generation_heartbeat_failed","jobId":%q,"correlationId":%q,"workerId":%q,"consecutiveFailures":%d,"error":%q}`,
				job.ID, job.CorrelationID, workerID, consecutiveFailures, err.Error())
			if consecutiveFailures >= 2 {
				cancelExecution()
				return
			}
		}
	}
}
