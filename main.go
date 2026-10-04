package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"travel-diary-backend/internal/app"
	"travel-diary-backend/internal/config"
)

func main() {
	cfg := config.Load()

	fiberApp, err := app.New(cfg)
	if err != nil {
		log.Fatal(err)
	}

	serverError := make(chan error, 1)
	go func() {
		serverError <- fiberApp.Listen(cfg.ServerAddress())
	}()

	signalContext, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()
	select {
	case err := <-serverError:
		if err != nil {
			log.Fatal(err)
		}
	case <-signalContext.Done():
		log.Printf(`{"event":"server_shutdown_started"}`)
		shutdownContext, cancel := context.WithTimeout(context.Background(), 40*time.Second)
		defer cancel()
		if err := fiberApp.ShutdownWithContext(shutdownContext); err != nil {
			log.Printf(`{"event":"server_shutdown_failed","error":%q}`, err.Error())
			return
		}
		log.Printf(`{"event":"server_shutdown_completed"}`)
	}
}
