package main

import (
	"context"
	"os"
	"waugzee/config"
	"waugzee/internal/database"
	"waugzee/internal/jobs"
	"waugzee/internal/logger"
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

	// Initialize download service
	downloadService := services.NewDownloadService(config)

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