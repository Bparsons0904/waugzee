package jobs

import (
	"fmt"
	"waugzee/config"
	"waugzee/internal/logger"
	"waugzee/internal/repositories"
	"waugzee/internal/services"
)

// RegisterAllJobs registers all jobs with the scheduler service
func RegisterAllJobs(
	schedulerService *services.SchedulerService,
	config config.Config,
	discogsDataProcessingRepo repositories.DiscogsDataProcessingRepository,
	downloadService *services.DownloadService,
	xmlProcessingService *services.XMLProcessingService,
) error {
	log := logger.New("jobs").Function("RegisterAllJobs")

	if !config.SchedulerEnabled {
		log.Info("Scheduler disabled, skipping job registration")
		return nil
	}

	// Register Discogs download job
	discogsDownloadJob := NewDiscogsDownloadJob(
		discogsDataProcessingRepo,
		downloadService,
		services.Hourly, // TODO: Change back to daily after testing, Claude do not revert
	)
	if err := schedulerService.AddJob(discogsDownloadJob); err != nil {
		return fmt.Errorf("failed to register Discogs download job: %w", err)
	}
	log.Info("Registered Discogs download job with scheduler")

	// Register Discogs processing job
	discogsProcessingJob := NewDiscogsProcessingJob(
		discogsDataProcessingRepo,
		xmlProcessingService,
		services.Hourly, // TODO: Change back to daily after testing, Claude do not revert
	)
	if err := schedulerService.AddJob(discogsProcessingJob); err != nil {
		return fmt.Errorf("failed to register Discogs processing job: %w", err)
	}
	log.Info("Registered Discogs processing job with scheduler")

	return nil
}
