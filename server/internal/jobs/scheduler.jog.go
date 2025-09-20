package jobs

import (
	"waugzee/config"
	"waugzee/internal/logger"
	"waugzee/internal/services"
)

// RegisterAllJobs registers all jobs with the scheduler service
// Currently no jobs are implemented - scheduler infrastructure is in place for future use
func RegisterAllJobs(
	schedulerService *services.SchedulerService,
	config config.Config,
) error {
	log := logger.New("jobs").Function("RegisterAllJobs")

	if !config.SchedulerEnabled {
		log.Info("Scheduler disabled, skipping job registration")
		return nil
	}

	// No jobs currently implemented
	// The scheduler service is available for future background job implementations
	log.Info("No jobs registered - scheduler infrastructure ready for future use")

	return nil
}
