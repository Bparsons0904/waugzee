package main

import (
	"context"
	"os"
	"waugzee/config"
	"waugzee/internal/database"
	"waugzee/internal/events"
	"waugzee/internal/jobs"
	logger "github.com/Bparsons0904/goLogger"
	"waugzee/internal/repositories"
	"waugzee/internal/services"
)

func main() {
	log := logger.New("test-download")

	// Initialize config
	config, err := config.New()
	if err != nil {
		log.Er("failed to initialize config", err)
		os.Exit(1)
	}

	// Initialize database
	db, err := database.New(config)
	if err != nil {
		log.Er("failed to create database", err)
		os.Exit(1)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Er("failed to close database", err)
		}
	}()

	// Initialize repositories
	repos := repositories.New(db)

	// Initialize event bus
	eventBus := events.New(db.Cache.Events, config)
	defer func() {
		if err := eventBus.Close(); err != nil {
			log.Er("failed to close event bus", err)
		}
	}()

	// Initialize download service
	downloadService := services.NewDownloadService(config, eventBus)

	// Create download job
	downloadJob := jobs.NewDiscogsDownloadJob(
		downloadService,
		repos.DiscogsDataProcessing,
		services.Daily,
	)

	log.Info("Starting manual download job execution")

	// Execute the job
	ctx := context.Background()
	if err := downloadJob.Execute(ctx); err != nil {
		log.Er("download job failed", err)
		os.Exit(1)
	}

	log.Info("Download job completed successfully")
}