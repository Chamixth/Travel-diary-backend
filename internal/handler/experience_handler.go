package handler

import (
	"errors"
	"strconv"
	"travel-diary-backend/internal/dao"
	"travel-diary-backend/internal/middleware"
	"travel-diary-backend/internal/service"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/mongo"
)

func ExperienceGetHandler(db *mongo.Database) fiber.Handler {
	svc := newExperienceService(db)
	return func(c *fiber.Ctx) error {
		userID, ok := middleware.Auth0UserID(c)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		version, err := svc.GetActive(c.Context(), userID, c.Params("id"))
		if errors.Is(err, mongo.ErrNoDocuments) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "experience not generated"})
		}
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not load experience"})
		}
		return c.JSON(version)
	}
}

func ExperienceVersionsListHandler(db *mongo.Database) fiber.Handler {
	svc := newExperienceService(db)
	return func(c *fiber.Ctx) error {
		userID, ok := middleware.Auth0UserID(c)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		versions, err := svc.ListVersions(c.Context(), userID, c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "experience not found"})
		}
		return c.JSON(fiber.Map{"versions": versions})
	}
}

func ExperienceVersionActivateHandler(db *mongo.Database) fiber.Handler {
	svc := newExperienceService(db)
	return func(c *fiber.Ctx) error {
		userID, ok := middleware.Auth0UserID(c)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		versionNumber, err := strconv.ParseInt(c.Params("version"), 10, 64)
		if err != nil || versionNumber < 1 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid experience version"})
		}
		version, err := svc.ActivateVersion(c.Context(), userID, c.Params("id"), versionNumber)
		if errors.Is(err, mongo.ErrNoDocuments) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "experience version not found"})
		}
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not activate experience version"})
		}
		return c.JSON(version)
	}
}

func newExperienceService(db *mongo.Database) *service.ExperienceService {
	return service.NewExperienceService(
		dao.NewTripDAO(db), dao.NewTripImageDAO(db), dao.NewAlbumPlanDAO(db), dao.NewExperienceVersionDAO(db),
	)
}
