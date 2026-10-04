package app

import (
	"context"
	"errors"
	"fmt"
	"time"
	"travel-diary-backend/internal/config"
	"travel-diary-backend/internal/dao"
	"travel-diary-backend/internal/generation"
	"travel-diary-backend/internal/integrations"
	"travel-diary-backend/internal/middleware"
	"travel-diary-backend/internal/router"
	"travel-diary-backend/internal/service"

	"github.com/gofiber/fiber/v2"
	fibercors "github.com/gofiber/fiber/v2/middleware/cors"
)

func New(cfg config.Config) (*fiber.App, error) {
	// Fiber's default BodyLimit is 4 MB, which rejects even a single phone
	// photo on the trip-image multipart upload. Raise it app-wide.
	app := fiber.New(fiber.Config{
		BodyLimit: 64 * 1024 * 1024, // 64 MB
	})

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if err := config.ValidateDatabaseURL(cfg.DatabaseURL); err != nil {
		return nil, err
	}

	client, err := integrations.NewMongoClient(cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}

	db := client.Database(cfg.MongoDatabase)
	if err := dao.NewPricingDAO(db).SeedDefaults(context.Background(), service.DefaultPricingPlans()); err != nil {
		return nil, fmt.Errorf("seed pricing plans: %w", err)
	}
	s3Client, err := integrations.NewS3Client(context.Background(), cfg.AWSRegion, cfg.AWSAccessKeyID, cfg.AWSSecretAccessKey, cfg.AWSBucket)
	if err != nil {
		return nil, fmt.Errorf("create s3 client: %w", err)
	}
	analysisQueue := service.NewImageAnalysisQueue(cfg, db, s3Client)
	generationQueue := generation.NewQueue(cfg, db)

	corsCfg := fibercors.Config{
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET,POST,PUT,PATCH,DELETE,OPTIONS",
	}
	if cfg.CORSAllowedOrigins != "" {
		corsCfg.AllowOrigins = cfg.CORSAllowedOrigins
	}
	app.Use(fibercors.New(corsCfg))
	app.Use(middleware.RequestLogger())
	app.Use(middleware.Recover())

	router.Register(app, cfg, db, analysisQueue, generationQueue)
	generationQueue.Start()
	app.Hooks().OnShutdown(func() error {
		shutdownContext, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		queueErr := generationQueue.Shutdown(shutdownContext)
		databaseErr := client.Disconnect(shutdownContext)
		return errors.Join(queueErr, databaseErr)
	})

	return app, nil
}
