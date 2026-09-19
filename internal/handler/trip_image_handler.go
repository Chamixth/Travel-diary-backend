package handler

import (
	"context"

	"travel-diary-backend/internal/config"
	"travel-diary-backend/internal/dao"
	"travel-diary-backend/internal/dto"
	"travel-diary-backend/internal/integrations"
	"travel-diary-backend/internal/middleware"
	"travel-diary-backend/internal/service"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/mongo"
)

// TripImagePresignHandler hands back one presigned S3 PUT URL per requested
// file. No file bytes touch this backend — the browser uploads directly to
// S3 with the returned URL and confirms completion via TripImageCompleteHandler.
func TripImagePresignHandler(cfg config.Config, db *mongo.Database, analysis *service.ImageAnalysisQueue) fiber.Handler {
	return func(c *fiber.Ctx) error {
		svc, err := newTripImageService(cfg, db, analysis)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		userID, ok := middleware.Auth0UserID(c)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}

		var req dto.PresignImagesRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
		}

		resp, err := svc.PresignUploads(c.Context(), userID, c.Params("id"), req.Files)
		if err != nil {
			if err == fiber.ErrNotFound {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "trip not found"})
			}
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(resp)
	}
}

// TripImageCompleteHandler is called once the browser has PUT each file
// straight to S3 using the URLs from TripImagePresignHandler. It verifies
// every object against S3 directly, records it, and queues analysis.
func TripImageCompleteHandler(cfg config.Config, db *mongo.Database, analysis *service.ImageAnalysisQueue) fiber.Handler {
	return func(c *fiber.Ctx) error {
		svc, err := newTripImageService(cfg, db, analysis)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		userID, ok := middleware.Auth0UserID(c)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}

		var req dto.CompleteImagesRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
		}

		resp, err := svc.CompleteUploads(c.Context(), userID, c.Params("id"), req.Images)
		if err != nil {
			if err == fiber.ErrNotFound {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "trip not found"})
			}
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusCreated).JSON(resp)
	}
}

func TripImageListHandler(cfg config.Config, db *mongo.Database, analysis *service.ImageAnalysisQueue) fiber.Handler {
	return func(c *fiber.Ctx) error {
		svc, err := newTripImageService(cfg, db, analysis)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		userID, ok := middleware.Auth0UserID(c)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		resp, err := svc.List(c.Context(), userID, c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(dto.TripImageUploadResponse{Uploaded: resp})
	}
}

func TripImageRetryAnalysisHandler(cfg config.Config, db *mongo.Database, analysis *service.ImageAnalysisQueue) fiber.Handler {
	return func(c *fiber.Ctx) error {
		svc, err := newTripImageService(cfg, db, analysis)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		userID, ok := middleware.Auth0UserID(c)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		retried, err := svc.RetryFailedAnalysis(c.Context(), userID, c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"retried": retried})
	}
}

func TripImageDeleteHandler(cfg config.Config, db *mongo.Database, analysis *service.ImageAnalysisQueue) fiber.Handler {
	return func(c *fiber.Ctx) error {
		svc, err := newTripImageService(cfg, db, analysis)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		userID, ok := middleware.Auth0UserID(c)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		if err := svc.Delete(c.Context(), userID, c.Params("id"), c.Params("imageId")); err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.SendStatus(fiber.StatusNoContent)
	}
}

func newTripImageService(cfg config.Config, db *mongo.Database, analysis *service.ImageAnalysisQueue) (*service.TripImageService, error) {
	s3Client, err := integrations.NewS3Client(context.Background(), cfg.AWSRegion, cfg.AWSAccessKeyID, cfg.AWSSecretAccessKey, cfg.AWSBucket)
	if err != nil {
		return nil, err
	}
	return service.NewTripImageService(dao.NewTripDAO(db), dao.NewTripImageDAO(db), dao.NewTripImageAnalysisDAO(db), s3Client, analysis), nil
}
