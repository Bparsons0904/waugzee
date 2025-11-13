package jobs

import (
	"context"
	"waugzee/internal/logger"
	"waugzee/internal/services"
)

type DailyRecommendationJob struct {
	recommendationService *services.RecommendationService
	log                   logger.Logger
	schedule              services.Schedule
}

func NewDailyRecommendationJob(
	recommendationService *services.RecommendationService,
	schedule services.Schedule,
) *DailyRecommendationJob {
	log := logger.New("dailyRecommendationJob")
	log.Info("Creating new daily recommendation job", "schedule", schedule)

	return &DailyRecommendationJob{
		recommendationService: recommendationService,
		log:                   log,
		schedule:              schedule,
	}
}

func (j *DailyRecommendationJob) Name() string {
	return "DailyRecommendationGeneration"
}

func (j *DailyRecommendationJob) Execute(ctx context.Context) error {
	log := j.log.Function("Execute")

	log.Info("Starting daily recommendation generation")

	if err := j.recommendationService.GenerateDailyRecommendationsForAllUsers(ctx); err != nil {
		return log.Err("daily recommendation generation failed", err)
	}

	log.Info("Daily recommendation generation completed successfully")
	return nil
}

func (j *DailyRecommendationJob) Schedule() services.Schedule {
	return j.schedule
}
