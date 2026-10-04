package handler

import (
	"context"
	"strconv"

	"travel-diary-backend/internal/config"
	"travel-diary-backend/internal/dao"
	"travel-diary-backend/internal/integrations"
	"travel-diary-backend/internal/middleware"
	"travel-diary-backend/internal/service"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/mongo"
)

func TripMediaBackfillHandler(cfg config.Config, db *mongo.Database) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, ok := middleware.Auth0UserID(c)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		limit := service.MaxMediaBackfillBatch
		if raw := c.Query("limit"); raw != "" {
			parsed, err := strconv.Atoi(raw)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "limit must be an integer"})
			}
			limit = parsed
		}
		s3, err := integrations.NewS3Client(context.Background(), cfg.AWSRegion, cfg.AWSAccessKeyID, cfg.AWSSecretAccessKey, cfg.AWSBucket)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not initialize image storage"})
		}
		svc := service.NewMediaBackfillService(dao.NewTripImageDAO(db), dao.NewMediaAssetDAO(db), s3)
		result, err := svc.BackfillTrip(c.Context(), userID, c.Params("id"), limit)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(result)
	}
}
