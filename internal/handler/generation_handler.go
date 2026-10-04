package handler

import (
	"errors"

	"travel-diary-backend/internal/generation"
	"travel-diary-backend/internal/middleware"
	"travel-diary-backend/internal/models"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/mongo"
)

type generationJobResponse struct {
	ID             string                        `json:"id"`
	TripID         string                        `json:"tripId"`
	Scope          string                        `json:"scope"`
	SourceRevision string                        `json:"sourceRevision"`
	CorrelationID  string                        `json:"correlationId"`
	Status         models.GenerationJobStatus    `json:"status"`
	Stage          models.GenerationStage        `json:"stage"`
	StageLabel     string                        `json:"stageLabel"`
	Attempt        int                           `json:"attempt"`
	MaxAttempts    int                           `json:"maxAttempts"`
	NextAttemptAt  string                        `json:"nextAttemptAt,omitempty"`
	History        []models.GenerationStageEvent `json:"history"`
	FailureCode    string                        `json:"failureCode,omitempty"`
	FailureMessage string                        `json:"failureMessage,omitempty"`
	CreatedAt      string                        `json:"createdAt"`
	UpdatedAt      string                        `json:"updatedAt"`
}

func GenerationJobStartHandler(svc *generation.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, ok := middleware.Auth0UserID(c)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		job, created, err := svc.Start(c.Context(), userID, c.Params("id"))
		if errors.Is(err, mongo.ErrNoDocuments) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "trip not found"})
		}
		if errors.Is(err, generation.ErrImagesNotReady) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
		}
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not start generation"})
		}
		status := fiber.StatusOK
		if created {
			status = fiber.StatusAccepted
		}
		return c.Status(status).JSON(toGenerationJobResponse(job))
	}
}

func GenerationJobGetHandler(svc *generation.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, ok := middleware.Auth0UserID(c)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		job, err := svc.Get(c.Context(), userID, c.Params("id"), c.Params("jobId"))
		if errors.Is(err, mongo.ErrNoDocuments) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "generation job not found"})
		}
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not load generation job"})
		}
		return c.JSON(toGenerationJobResponse(job))
	}
}

func toGenerationJobResponse(job models.GenerationJob) generationJobResponse {
	response := generationJobResponse{
		ID: job.ID, TripID: job.TripID, Scope: job.Scope, SourceRevision: job.SourceRevision,
		CorrelationID: job.CorrelationID, Status: job.Status, Stage: job.Stage,
		StageLabel: generation.ProductStageLabel(job.Stage), Attempt: job.Attempt, MaxAttempts: job.MaxAttempts,
		History: job.History, FailureCode: job.FailureCode, FailureMessage: publicGenerationFailure(job),
		CreatedAt: job.CreatedAt.Format("2006-01-02T15:04:05.999999999Z07:00"),
		UpdatedAt: job.UpdatedAt.Format("2006-01-02T15:04:05.999999999Z07:00"),
	}
	if job.NextAttemptAt != nil {
		response.NextAttemptAt = job.NextAttemptAt.Format("2006-01-02T15:04:05.999999999Z07:00")
	}
	return response
}

func publicGenerationFailure(job models.GenerationJob) string {
	if job.Status != models.GenerationJobFailed {
		return ""
	}
	switch job.FailureCode {
	case "journey_images_failed", "journey_analyses_failed":
		return "We couldn't finish reading the journey. Please try again."
	case "legacy_album_generation_failed":
		return "We couldn't finish designing this experience. Please try again."
	default:
		return "We couldn't finish this experience. Please try again."
	}
}
