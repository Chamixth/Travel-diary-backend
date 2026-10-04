package handler

import (
	"travel-diary-backend/internal/config"
	"travel-diary-backend/internal/dao"
	"travel-diary-backend/internal/dto"
	"travel-diary-backend/internal/middleware"
	"travel-diary-backend/internal/service"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/mongo"
)

func TripCreateHandler(db *mongo.Database) fiber.Handler {
	svc := service.NewTripService(dao.NewTripDAO(db))
	return func(c *fiber.Ctx) error {
		userID, ok := middleware.Auth0UserID(c)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		var req dto.TripCreateRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
		}
		resp, err := svc.Create(c.Context(), userID, req)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusCreated).JSON(resp)
	}
}

func TripListHandler(db *mongo.Database) fiber.Handler {
	svc := service.NewTripService(dao.NewTripDAO(db))
	return func(c *fiber.Ctx) error {
		userID, ok := middleware.Auth0UserID(c)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		resp, err := svc.List(c.Context(), userID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(resp)
	}
}

func TripGetHandler(db *mongo.Database) fiber.Handler {
	svc := service.NewTripService(dao.NewTripDAO(db))
	return func(c *fiber.Ctx) error {
		userID, ok := middleware.Auth0UserID(c)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		resp, err := svc.Get(c.Context(), userID, c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(resp)
	}
}

func TripUpdateHandler(db *mongo.Database) fiber.Handler {
	svc := service.NewTripService(dao.NewTripDAO(db))
	return func(c *fiber.Ctx) error {
		userID, ok := middleware.Auth0UserID(c)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		var req dto.TripUpdateRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
		}
		resp, err := svc.Update(c.Context(), userID, c.Params("id"), req)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(resp)
	}
}

func TripDeleteHandler(cfg config.Config, db *mongo.Database) fiber.Handler {
	svc := service.NewTripService(dao.NewTripDAO(db))
	return func(c *fiber.Ctx) error {
		userID, ok := middleware.Auth0UserID(c)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		imageSvc, err := newTripImageService(cfg, db, nil)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not initialize image storage"})
		}
		if err := imageSvc.DeleteByTripID(c.Context(), userID, c.Params("id")); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not delete trip images"})
		}
		if err := svc.Delete(c.Context(), userID, c.Params("id")); err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		_ = dao.NewAlbumPlanDAO(db).DeleteByTripID(c.Context(), userID, c.Params("id"))
		_ = dao.NewExperienceVersionDAO(db).DeleteByTripID(c.Context(), userID, c.Params("id"))
		_ = dao.NewJourneyDAO(db).DeleteByTripID(c.Context(), userID, c.Params("id"))
		_ = dao.NewStoryDAO(db).DeleteByTripID(c.Context(), userID, c.Params("id"))
		_ = dao.NewCreativeDAO(db).DeleteByTripID(c.Context(), userID, c.Params("id"))
		_ = dao.NewGenerationJobDAO(db).DeleteByTripID(c.Context(), userID, c.Params("id"))
		_ = dao.NewUploadReservationDAO(db).DeleteByTripID(c.Context(), userID, c.Params("id"))
		return c.SendStatus(fiber.StatusNoContent)
	}
}
