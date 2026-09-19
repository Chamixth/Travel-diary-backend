package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"travel-diary-backend/internal/ai"
	"travel-diary-backend/internal/config"
	"travel-diary-backend/internal/dao"
	"travel-diary-backend/internal/models"

	"github.com/google/uuid"
)

type AlbumService struct {
	trips    dao.TripDAO
	images   dao.TripImageDAO
	analyses dao.TripImageAnalysisDAO
	plans    dao.AlbumPlanDAO
	config   config.Config
}

func NewAlbumService(trips dao.TripDAO, images dao.TripImageDAO, analyses dao.TripImageAnalysisDAO, plans dao.AlbumPlanDAO, cfg config.Config) *AlbumService {
	return &AlbumService{trips: trips, images: images, analyses: analyses, plans: plans, config: cfg}
}

func (s *AlbumService) Generate(ctx context.Context, userID, tripID string) (models.AlbumPlanRecord, error) {
	trip, err := s.trips.FindByIDAndUserID(ctx, tripID, userID)
	if err != nil {
		return models.AlbumPlanRecord{}, err
	}
	images, err := s.images.ListByTripID(ctx, userID, tripID)
	if err != nil {
		return models.AlbumPlanRecord{}, err
	}
	analyses, err := s.analyses.ListByTripID(ctx, userID, tripID)
	if err != nil {
		return models.AlbumPlanRecord{}, err
	}

	analysisByImage := make(map[string]models.TripImageAnalysis, len(analyses))
	for _, analysis := range analyses {
		analysisByImage[analysis.ImageID] = analysis
	}
	sort.SliceStable(images, func(i, j int) bool {
		left := analysisByImage[images[i].ID]
		right := analysisByImage[images[j].ID]
		return left.Story.Importance > right.Story.Importance
	})

	planningImages := make([]ai.AlbumPlanningImage, 0, len(images))
	for _, image := range images {
		analysis := analysisByImage[image.ID]
		planningImages = append(planningImages, ai.AlbumPlanningImage{
			ID: image.ID, FileName: image.FileName, Width: image.Width, Height: image.Height,
			EXIF: image.EXIF, Analysis: analysis,
		})
	}
	plan, planErr := ai.GenerateSmartAlbumPlan(ctx, s.config.OpenAIKey, s.config.OpenAIModelImage, trip.Title, trip.Destination, trip.CinematicMood, trip.Intention, trip.Departure, trip.Return, planningImages)
	if planErr == nil {
		if validationErr := validateAlbumPlan(plan, images); validationErr != nil {
			// Do not make a second long AI request here. The fallback below is
			// deterministic and keeps album generation within the API timeout.
			log.Printf("smart_album_validation_failed trip_id=%s error=%v; using fallback", tripID, validationErr)
			planErr = validationErr
		}
	}
	if planErr != nil {
		log.Printf("smart_album_generation_failed trip_id=%s error=%v", tripID, planErr)
		plan = buildAlbumPlan(trip.Title, trip.Destination, images, analysisByImage)
		// Do not make a second long AI request after a planning timeout. The
		// fallback must be returned immediately and remain usable offline.
		plan.Quotes = fallbackAlbumQuotes()
	}
	plan = sanitizeAlbumPlan(plan, images, trip.Title, trip.Destination)
	if validationErr := validateAlbumPlan(plan, images); validationErr != nil {
		log.Printf("album_plan_repair_required trip_id=%s error=%v", tripID, validationErr)
		plan = buildAlbumPlan(trip.Title, trip.Destination, images, analysisByImage)
		if len(plan.Quotes) == 0 {
			plan.Quotes = fallbackAlbumQuotes()
		}
		plan = sanitizeAlbumPlan(plan, images, trip.Title, trip.Destination)
	}
	if validationErr := validateAlbumPlan(plan, images); validationErr != nil {
		return models.AlbumPlanRecord{}, fmt.Errorf("could not produce a valid album plan: %w", validationErr)
	}
	return s.plans.Upsert(ctx, models.AlbumPlanRecord{
		ID:     uuid.NewString(),
		TripID: tripID,
		UserID: userID,
		Plan:   plan,
	})
}

func validateAlbumPlan(plan models.AlbumPlan, images []models.TripImage) error {
	validIDs := make(map[string]bool, len(images))
	for _, image := range images {
		validIDs[image.ID] = true
	}
	seen := make(map[string]bool, len(images))
	coverCount, closingCount := 0, 0
	blockCount := 0
	var lastBlock *models.AlbumBlock

	for _, chapter := range plan.Chapters {
		for _, block := range chapter.Blocks {
			blockCount++
			lastBlock = &block
			count := len(block.ImageIDs)
			for _, id := range block.ImageIDs {
				if !validIDs[id] {
					return errors.New("album plan contains an unknown image ID")
				}
				// A one-photo album must reuse its only image for the closing frame.
				if seen[id] && block.Type != "album_cover" && !(block.Type == "closing_frame" && len(validIDs) == 1) {
					return errors.New("album plan repeats an image")
				}
				seen[id] = true
			}
			switch block.Type {
			case "album_cover":
				coverCount++
				if count != 1 {
					return fmt.Errorf("album_cover must contain exactly 1 image, got %d", count)
				}
			case "full_bleed_image", "image_caption", "panorama", "closing_frame":
				if count != 1 {
					return fmt.Errorf("%s must contain exactly 1 image, got %d", block.Type, count)
				}
				if block.Type == "closing_frame" {
					closingCount++
				}
			case "full_bleed_quote":
				if count > 1 || strings.TrimSpace(block.Quote) == "" {
					return errors.New("full_bleed_quote must have a quote and at most one image")
				}
			case "portrait_pair", "landscape_pair":
				if count != 2 {
					return fmt.Errorf("%s must contain exactly 2 images, got %d", block.Type, count)
				}
			case "editorial_grid":
				if count < 3 || count > 6 {
					return fmt.Errorf("editorial_grid must contain 3-6 images, got %d", count)
				}
			case "film_strip":
				if count < 3 || count > 5 {
					return fmt.Errorf("film_strip must contain 3-5 images, got %d", count)
				}
			case "story_text":
				if strings.TrimSpace(block.Text) == "" {
					return errors.New("story_text must contain narrative text")
				}
			case "chapter_split":
			case "chapter_transition":
				if strings.TrimSpace(block.Text) == "" {
					return errors.New("chapter_transition must contain transition text")
				}
			default:
				return fmt.Errorf("unsupported album block type %q", block.Type)
			}
		}
	}
	if len(plan.Chapters) == 0 || blockCount == 0 || coverCount != 1 || closingCount != 1 {
		return errors.New("album plan must contain exactly one cover and one closing frame")
	}
	if len(plan.Chapters[0].Blocks) == 0 || plan.Chapters[0].Blocks[0].Type != "album_cover" {
		return errors.New("album_cover must be the first album block")
	}
	if lastBlock == nil || lastBlock.Type != "closing_frame" {
		return errors.New("closing_frame must be the final album block")
	}
	return nil
}

func sanitizeAlbumPlan(plan models.AlbumPlan, images []models.TripImage, title, destination string) models.AlbumPlan {
	validIDs := make(map[string]bool, len(images))
	for _, image := range images {
		validIDs[image.ID] = true
	}
	seen := make(map[string]bool, len(images))
	for chapterIndex := range plan.Chapters {
		plan.Chapters[chapterIndex].Order = chapterIndex + 1
		for blockIndex := range plan.Chapters[chapterIndex].Blocks {
			block := &plan.Chapters[chapterIndex].Blocks[blockIndex]
			filtered := block.ImageIDs[:0]
			for _, id := range block.ImageIDs {
				if validIDs[id] && (!seen[id] || (block.Type == "closing_frame" && len(validIDs) == 1)) {
					filtered = append(filtered, id)
					seen[id] = true
				}
			}
			block.ImageIDs = filtered
		}
	}
	if len(plan.Chapters) == 0 {
		plan = buildAlbumPlan(title, destination, images, nil)
	}
	if plan.Title == "" {
		plan.Title = title
		if plan.Title == "" {
			plan.Title = destination
		}
	}
	if plan.Subtitle == "" {
		plan.Subtitle = "A collection of moments, kept together."
	}
	if plan.Tone == "" {
		plan.Tone = "cinematic"
	}
	return plan
}

func fallbackAlbumQuotes() []models.AlbumQuote {
	return []models.AlbumQuote{
		{From: "COVER", To: "PROLOGUE", Text: "Every journey begins with a moment worth keeping.", Order: 1},
		{From: "PROLOGUE", To: "HEADER", Text: "The story gathers where the ordinary becomes unforgettable.", Order: 2},
		{From: "HEADER", To: "HIGHLIGHTS", Text: "Some moments stay long after the journey moves on.", Order: 3},
		{From: "HIGHLIGHTS", To: "EPILOGUE", Text: "What we carry home is often more than what we took with us.", Order: 4},
		{From: "EPILOGUE", To: "CREDITS", Text: "The road ends, but the memory keeps unfolding.", Order: 5},
	}
}

func (s *AlbumService) Get(ctx context.Context, userID, tripID string) (models.AlbumPlanRecord, error) {
	return s.plans.FindByTripID(ctx, userID, tripID)
}

func buildAlbumPlan(title, destination string, images []models.TripImage, analyses map[string]models.TripImageAnalysis) models.AlbumPlan {
	ids := make([]string, 0, len(images))
	for _, image := range images {
		if image.S3URL != "" {
			ids = append(ids, image.ID)
		}
	}
	if title == "" {
		title = destination
	}
	if title == "" {
		title = "Your journey"
	}

	blocks := make([]models.AlbumBlock, 0, len(ids)+3)
	if len(ids) > 0 {
		blocks = append(blocks, models.AlbumBlock{Type: "album_cover", ImageIDs: ids[:1], Title: title})
	}
	if destination != "" {
		blocks = append(blocks, models.AlbumBlock{Type: "story_text", Text: fmt.Sprintf("A collection of moments from %s.", destination)})
	}

	// Reserve the final image for the closing frame so the fallback still
	// follows the same contract as a successful AI-generated plan.
	middleIDs := ids
	if len(ids) > 1 {
		middleIDs = ids[1 : len(ids)-1]
	}
	for start := 0; start < len(middleIDs); {
		remaining := len(middleIDs) - start
		if remaining >= 3 {
			count := remaining
			if count > 6 {
				count = 6
			}
			blocks = append(blocks, models.AlbumBlock{Type: "editorial_grid", ImageIDs: middleIDs[start : start+count]})
			start += count
			continue
		}
		imageID := middleIDs[start]
		block := models.AlbumBlock{Type: "image_caption", ImageIDs: []string{imageID}}
		if analysis, ok := analyses[imageID]; ok {
			block.Caption = analysis.Caption
		}
		if strings.TrimSpace(block.Caption) == "" {
			block.Caption = "A moment worth keeping from the journey."
		}
		blocks = append(blocks, block)
		start++
	}
	if len(ids) > 0 {
		closingID := ids[len(ids)-1]
		blocks = append(blocks, models.AlbumBlock{
			Type: "closing_frame", ImageIDs: []string{closingID},
			Title: "Until the Next Journey", Caption: "The road ends, but the memory keeps unfolding.",
		})
	}

	return models.AlbumPlan{
		Title: title, Subtitle: "A collection of moments, kept together.", Tone: "cinematic",
		Chapters: []models.AlbumChapter{{ID: "journey", Order: 1, Eyebrow: "Your photographs", Title: "The journey so far", Blocks: blocks}},
	}
}
