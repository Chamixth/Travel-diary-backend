package router

import (
	"travel-diary-backend/internal/config"
	"travel-diary-backend/internal/generation"
	"travel-diary-backend/internal/handler"
	"travel-diary-backend/internal/middleware"
	"travel-diary-backend/internal/service"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/mongo"
)

func Register(app *fiber.App, cfg config.Config, db *mongo.Database, analysis *service.ImageAnalysisQueue, generationQueue *generation.Queue) {
	api := app.Group("/api")

	v1 := api.Group("/v1")
	v1.Get("/health", handler.HealthCheck(cfg))
	pricing := v1.Group("/pricing")
	pricing.Get("/", handler.PricingListHandler(cfg, db))
	pricing.Get("/:slug", handler.PricingGetHandler(cfg, db))
	auth := v1.Group("/auth")
	auth.Post("/register", handler.RegisterHandler(cfg, db))
	auth.Post("/login", handler.LoginHandler(cfg, db))
	auth.Post("/refresh", handler.RefreshHandler(cfg, db))
	auth.Get("/google", handler.GoogleLoginHandler(cfg, db))
	auth.Get("/google/callback", handler.GoogleCallbackHandler(cfg, db))

	protected := v1.Group("", middleware.Auth0(cfg, db))
	protected.Get("/auth/plan", handler.CurrentPlanHandler(db))
	protected.Put("/auth/plan", handler.ChangePlanHandler(db))

	trips := protected.Group("/trips")
	trips.Post("/", handler.TripCreateHandler(db))
	trips.Get("/", handler.TripListHandler(db))
	trips.Get("/:id", handler.TripGetHandler(db))
	trips.Put("/:id", handler.TripUpdateHandler(db))
	trips.Delete("/:id", handler.TripDeleteHandler(cfg, db))
	trips.Post("/:id/images/presign", handler.TripImagePresignHandler(cfg, db, analysis))
	trips.Post("/:id/images/complete", handler.TripImageCompleteHandler(cfg, db, analysis))
	trips.Get("/:id/images", handler.TripImageListHandler(cfg, db, analysis))
	trips.Get("/:id/images/:imageId/analysis", handler.TripImageAnalysisHandler(db))
	trips.Post("/:id/images/retry-analysis", handler.TripImageRetryAnalysisHandler(cfg, db, analysis))
	trips.Post("/:id/media/backfill", handler.TripMediaBackfillHandler(cfg, db))
	trips.Delete("/:id/images/:imageId", handler.TripImageDeleteHandler(cfg, db, analysis))
	trips.Get("/:id/analysis-status", handler.TripAnalysisStatusHandler(db))
	trips.Post("/:id/album/generate", handler.GenerateAlbumHandler(cfg, db))
	trips.Get("/:id/album", handler.AlbumGetHandler(cfg, db))
	trips.Get("/:id/experience", handler.ExperienceGetHandler(db))
	trips.Get("/:id/experience/versions", handler.ExperienceVersionsListHandler(db))
	trips.Put("/:id/experience/versions/:version/activate", handler.ExperienceVersionActivateHandler(db))
	jobs := generationQueue.Service(db)
	trips.Post("/:id/generation-jobs", handler.GenerationJobStartHandler(jobs))
	trips.Get("/:id/generation-jobs/:jobId", handler.GenerationJobGetHandler(jobs))
}
