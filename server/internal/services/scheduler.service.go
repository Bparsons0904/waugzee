package services

import (
	"context"
	"fmt"
	"sync"
	"time"
	"waugzee/internal/logger"

	"github.com/go-co-op/gocron"
)

type Schedule int

const (
	Hourly          Schedule = iota
	Daily                    // Start at 02:00 UTC every day
	DailyProcessing          // Start at 03:00 UTC every day (1 hour after download)
)

// Job represents a scheduled task that can be executed by the scheduler
type Job interface {
	// Name returns a human-readable name for the job
	Name() string

	// Execute runs the job with the given context
	// Context can be used for cancellation and timeout handling
	Execute(ctx context.Context) error
	Schedule() Schedule
}

type SchedulerService struct {
	scheduler *gocron.Scheduler
	jobs      []Job
	log       logger.Logger
	started   bool
	mu        sync.Mutex
	ctx       context.Context
	cancel    context.CancelFunc
}

func NewSchedulerService() *SchedulerService {
	// Create scheduler in UTC timezone
	scheduler := gocron.NewScheduler(time.UTC)

	// Create cancellable context for job execution
	ctx, cancel := context.WithCancel(context.Background())

	return &SchedulerService{
		scheduler: scheduler,
		jobs:      make([]Job, 0),
		log:       logger.New("scheduler"),
		started:   false,
		ctx:       ctx,
		cancel:    cancel,
	}
}

func (s *SchedulerService) executeJob(job Job, log logger.Logger) {
	log.Info("Executing scheduled job", "job", job.Name())
	if err := job.Execute(s.ctx); err != nil {
		_ = log.Err("Job execution failed", err, "job", job.Name())
	} else {
		log.Info("Job execution completed successfully", "job", job.Name())
	}
}

// AddJob registers a job with the scheduler
func (s *SchedulerService) AddJob(job Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	log := s.log.Function("AddJob")

	var err error
	switch job.Schedule() {
	case Daily:
		_, err = s.scheduler.Every(1).Day().At("02:00").Do(func() {
			s.executeJob(job, log)
		})
	case DailyProcessing:
		_, err = s.scheduler.Every(1).Day().At("03:00").Do(func() {
			s.executeJob(job, log)
		})
	case Hourly:
		_, err = s.scheduler.Every(1).Hour().Do(func() {
			s.executeJob(job, log)
		})
	}

	if err != nil {
		return log.Err("failed to register job with scheduler", err, "job", job.Name())
	}

	// Store job reference for management
	s.jobs = append(s.jobs, job)
	log.Info("Job registered successfully", "job", job.Name())

	return nil
}

// Start begins the scheduler
func (s *SchedulerService) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	log := s.log.Function("Start")

	if s.started {
		log.Info("Scheduler already started")
		return nil
	}

	if len(s.jobs) == 0 {
		log.Info("No jobs registered, scheduler will not start")
		return nil
	}

	log.Info("Starting scheduler", "jobCount", len(s.jobs))
	s.scheduler.StartAsync()
	s.started = true

	// Log next run times for all jobs
	for _, job := range s.scheduler.Jobs() {
		log.Info("Job scheduled", "nextRun", job.NextRun())
	}

	log.Info("Scheduler started successfully")
	return nil
}

// Stop gracefully shuts down the scheduler
func (s *SchedulerService) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	log := s.log.Function("Stop")

	if !s.started {
		log.Info("Scheduler not started, nothing to stop")
		return nil
	}

	log.Info("Stopping scheduler")

	// Cancel the context to signal running jobs to stop
	if s.cancel != nil {
		s.cancel()
	}

	s.scheduler.Stop()
	s.started = false

	log.Info("Scheduler stopped successfully")
	return nil
}

// IsRunning returns whether the scheduler is currently running
func (s *SchedulerService) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.started
}

// GetJobCount returns the number of registered jobs
func (s *SchedulerService) GetJobCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.jobs)
}

// GetNextRunTime returns the next scheduled run time if scheduler is running
func (s *SchedulerService) GetNextRunTime() *time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started || len(s.scheduler.Jobs()) == 0 {
		return nil
	}

	nextRun := s.scheduler.Jobs()[0].NextRun()
	return &nextRun
}

// TriggerJobByName manually executes a registered job by name
// Claude is this really needed? The jobs should be pretty stupid and just kicking off other services, we should be able to directly kick off the event we want to work with
func (s *SchedulerService) TriggerJobByName(ctx context.Context, jobName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	log := s.log.Function("TriggerJobByName")

	var targetJob Job
	for _, job := range s.jobs {
		if job.Name() == jobName {
			targetJob = job
			break
		}
	}

	if targetJob == nil {
		// Claude again with using fmt for errors, use the damn logger package.
		err := fmt.Errorf("job not found: %s", jobName)
		return log.Err("job not found", err, "job", jobName)
	}

	go func() {
		log.Info("Manually triggering job", "job", jobName)
		if err := targetJob.Execute(ctx); err != nil {
			_ = log.Err("Manual job execution failed", err, "job", jobName)
		} else {
			log.Info("Manual job execution completed", "job", jobName)
		}
	}()

	return nil
}
