package jobs

import (
	"waugzee/config"
	logger "github.com/Bparsons0904/goLogger"
	"waugzee/internal/repositories"
	"waugzee/internal/services"
)

// Import schedule constants
const (
	Daily   = services.Daily
	Hourly  = services.Hourly
	Monthly = services.Monthly
)

func RegisterAllJobs(
	schedulerService *services.SchedulerService,
	config config.Config,
	services services.Service,
	repos repositories.Repository,
) error {
	log := logger.New("jobs").Function("RegisterAllJobs")
	log.Info("Registering jobs")

	discogsDownloadJob := NewDiscogsDownloadJob(
		services.Download,
		repos.DiscogsDataProcessing,
		Daily,
	)
	if err := schedulerService.AddJob(discogsDownloadJob); err != nil {
		return log.Err("failed to register Discogs download job", err)
	}
	log.Info("Registered Discogs download job", "schedule", "hourly")

	discogsXMLParserJob := NewDiscogsXMLParserJob(
		services.DiscogsXMLParser,
		Daily,
	)
	if err := schedulerService.AddJob(discogsXMLParserJob); err != nil {
		return log.Err("failed to register Discogs XML parser job", err)
	}
	log.Info("Registered Discogs XML parser job", "schedule", "hourly")

	fileCleanupJob := NewFileCleanupJob(
		services.FileCleanup,
		Monthly,
	)
	if err := schedulerService.AddJob(fileCleanupJob); err != nil {
		return log.Err("failed to register file cleanup job", err)
	}
	log.Info("Registered file cleanup job", "schedule", "monthly")

	return nil
}
