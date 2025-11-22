package jobs

import (
	"context"
	logger "github.com/Bparsons0904/goLogger"
	"waugzee/internal/services"
)

type FileCleanupJob struct {
	fileCleanup *services.FileCleanupService
	log         logger.Logger
	schedule    services.Schedule
}

func NewFileCleanupJob(
	fileCleanup *services.FileCleanupService,
	schedule services.Schedule,
) *FileCleanupJob {
	log := logger.New("fileCleanupJob")
	log.Info("Creating new file cleanup job", "schedule", schedule)

	return &FileCleanupJob{
		fileCleanup: fileCleanup,
		log:         log,
		schedule:    schedule,
	}
}

func (j *FileCleanupJob) Name() string {
	return "MonthlyFileCleanup"
}

func (j *FileCleanupJob) Execute(ctx context.Context) error {
	log := j.log.Function("Execute")

	log.Info("Starting scheduled file cleanup check")

	if err := j.fileCleanup.ScheduledMonthlyCleanup(ctx); err != nil {
		return log.Err("scheduled cleanup failed", err)
	}

	log.Info("Scheduled file cleanup check completed")
	return nil
}

func (j *FileCleanupJob) Schedule() services.Schedule {
	return j.schedule
}
