package repositories

import (
	"context"
	"fmt"
	"time"
	"waugzee/internal/database"
	"waugzee/internal/logger"
	. "waugzee/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	DAILY_RECOMMENDATIONS_CACHE_PREFIX = "daily_recommendations"
	DAILY_RECOMMENDATIONS_CACHE_EXPIRY = 24 * time.Hour
)

type DailyRecommendationRepository interface {
	GetTodayRecommendation(
		ctx context.Context,
		tx *gorm.DB,
		userID uuid.UUID,
	) (*DailyRecommendation, error)
	CreateRecommendation(
		ctx context.Context,
		tx *gorm.DB,
		recommendation *DailyRecommendation,
	) error
	MarkAsListened(
		ctx context.Context,
		tx *gorm.DB,
		recommendationID uuid.UUID,
		userID uuid.UUID,
	) error
	ClearUserRecommendationCache(ctx context.Context, userID uuid.UUID) error
}

type dailyRecommendationRepository struct {
	cache database.CacheClient
	log   logger.Logger
}

func NewDailyRecommendationRepository(cache database.CacheClient) DailyRecommendationRepository {
	return &dailyRecommendationRepository{
		cache: cache,
		log:   logger.New("dailyRecommendationRepository"),
	}
}

func (r *dailyRecommendationRepository) GetTodayRecommendation(
	ctx context.Context,
	tx *gorm.DB,
	userID uuid.UUID,
) (*DailyRecommendation, error) {
	log := r.log.Function("GetTodayRecommendation")

	var cached *DailyRecommendation
	found, err := database.NewCacheBuilder(r.cache, userID.String()).
		WithContext(ctx).
		WithHash(DAILY_RECOMMENDATIONS_CACHE_PREFIX).
		Get(&cached)
	if err != nil {
		log.Warn("failed to get daily recommendation from cache", "userID", userID, "error", err)
	}

	if found {
		log.Info("Daily recommendation retrieved from cache", "userID", userID)
		return cached, nil
	}

	today := time.Now().Truncate(24 * time.Hour)

	recommendation, err := gorm.G[*DailyRecommendation](tx).
		Preload("UserRelease.Release.Genres", nil).
		Preload("UserRelease.Release.Artists", nil).
		Where(DailyRecommendation{UserID: userID}).
		Where("date = ?", today).
		First(ctx)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, err
		}
		return nil, log.Err("failed to get today's recommendation", err, "userID", userID)
	}

	err = database.NewCacheBuilder(r.cache, userID.String()).
		WithContext(ctx).
		WithHash(DAILY_RECOMMENDATIONS_CACHE_PREFIX).
		WithStruct(recommendation).
		WithTTL(DAILY_RECOMMENDATIONS_CACHE_EXPIRY).
		Set()
	if err != nil {
		log.Warn("failed to set daily recommendation in cache", "userID", userID, "error", err)
	}

	log.Info("Daily recommendation retrieved from database and cached", "userID", userID)
	return recommendation, nil
}

func (r *dailyRecommendationRepository) CreateRecommendation(
	ctx context.Context,
	tx *gorm.DB,
	recommendation *DailyRecommendation,
) error {
	log := r.log.Function("CreateRecommendation")

	err := gorm.G[DailyRecommendation](tx).Create(ctx, recommendation)
	if err != nil {
		return log.Err(
			"failed to create daily recommendation",
			err,
			"userID",
			recommendation.UserID,
			"userReleaseID",
			recommendation.UserReleaseID,
		)
	}

	r.clearUserRecommendationCache(ctx, recommendation.UserID)

	return nil
}

func (r *dailyRecommendationRepository) MarkAsListened(
	ctx context.Context,
	tx *gorm.DB,
	recommendationID uuid.UUID,
	userID uuid.UUID,
) error {
	log := r.log.Function("MarkAsListened")

	now := time.Now()
	rows, err := gorm.G[DailyRecommendation](tx).
		Where("id = ? AND user_id = ?", recommendationID, userID).
		Update(ctx, "listened_at", now)
	if err != nil {
		return log.Err(
			"failed to mark recommendation as listened",
			err,
			"recommendationID",
			recommendationID,
			"userID",
			userID,
		)
	}

	if rows == 0 {
		return fmt.Errorf("recommendation not found or not owned by user")
	}

	err = r.clearUserRecommendationCacheWithError(ctx, userID)
	if err != nil {
		log.Warn("failed to clear recommendation cache", "userID", userID, "error", err)
	}

	return nil
}

func (r *dailyRecommendationRepository) clearUserRecommendationCache(
	ctx context.Context,
	userID uuid.UUID,
) {
	err := database.NewCacheBuilder(r.cache, userID.String()).
		WithContext(ctx).
		WithHash(DAILY_RECOMMENDATIONS_CACHE_PREFIX).
		Delete()
	if err != nil {
		r.log.Warn("failed to clear user recommendation cache", "userID", userID, "error", err)
	}
}

func (r *dailyRecommendationRepository) clearUserRecommendationCacheWithError(
	ctx context.Context,
	userID uuid.UUID,
) error {
	err := database.NewCacheBuilder(r.cache, userID.String()).
		WithContext(ctx).
		WithHash(DAILY_RECOMMENDATIONS_CACHE_PREFIX).
		Delete()
	if err != nil {
		return err
	}
	return nil
}

func (r *dailyRecommendationRepository) ClearUserRecommendationCache(
	ctx context.Context,
	userID uuid.UUID,
) error {
	log := r.log.Function("ClearUserRecommendationCache")

	r.clearUserRecommendationCache(ctx, userID)

	log.Info("cleared user recommendation cache", "userID", userID)
	return nil
}
