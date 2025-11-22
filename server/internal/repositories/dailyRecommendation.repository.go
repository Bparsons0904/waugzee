package repositories

import (
	"context"
	"fmt"
	"time"
	"waugzee/internal/constants"
	"waugzee/internal/database"
	logger "github.com/Bparsons0904/goLogger"
	. "waugzee/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	DAILY_RECOMMENDATIONS_CACHE_PREFIX = "daily_recommendations"
	DAILY_RECOMMENDATIONS_CACHE_EXPIRY = 24 * time.Hour
	RECENT_RECOMMENDATION_CACHE_PREFIX = "recent_recommendation"
	RECENT_RECOMMENDATION_CACHE_EXPIRY = 24 * time.Hour
)

type StreakData struct {
	CurrentStreak int
	LongestStreak int
}

type DailyRecommendationRepository interface {
	GetTodayRecommendation(
		ctx context.Context,
		tx *gorm.DB,
		userID uuid.UUID,
	) (*DailyRecommendation, error)
	GetMostRecentRecommendation(
		ctx context.Context,
		tx *gorm.DB,
		userID uuid.UUID,
	) (*DailyRecommendation, error)
	GetByID(
		ctx context.Context,
		tx *gorm.DB,
		recommendationID uuid.UUID,
		userID uuid.UUID,
	) (*DailyRecommendation, error)
	GetAllUserRecommendations(
		ctx context.Context,
		tx *gorm.DB,
		userID uuid.UUID,
	) ([]*DailyRecommendation, error)
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
	ClearUserStreakCache(ctx context.Context, userID uuid.UUID) error
	CalculateUserStreaks(ctx context.Context, tx *gorm.DB, userID uuid.UUID) (*StreakData, error)
	GetUserStreakFromCache(ctx context.Context, userID uuid.UUID) (*StreakData, bool, error)
	SetUserStreakCache(ctx context.Context, userID uuid.UUID, streakData *StreakData) error
}

type dailyRecommendationRepository struct {
	cache database.CacheClient
}

func NewDailyRecommendationRepository(cache database.CacheClient) DailyRecommendationRepository {
	return &dailyRecommendationRepository{
		cache: cache,
	}
}

func (r *dailyRecommendationRepository) GetTodayRecommendation(
	ctx context.Context,
	tx *gorm.DB,
	userID uuid.UUID,
) (*DailyRecommendation, error) {
	log := logger.New("dailyRecommendationRepository").TraceFromContext(ctx).Function("GetTodayRecommendation")

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

func (r *dailyRecommendationRepository) GetMostRecentRecommendation(
	ctx context.Context,
	tx *gorm.DB,
	userID uuid.UUID,
) (*DailyRecommendation, error) {
	log := logger.New("dailyRecommendationRepository").TraceFromContext(ctx).Function("GetMostRecentRecommendation")

	var cached *DailyRecommendation
	found, err := database.NewCacheBuilder(r.cache, userID.String()).
		WithContext(ctx).
		WithHash(RECENT_RECOMMENDATION_CACHE_PREFIX).
		Get(&cached)
	if err != nil {
		log.Warn("failed to get recent recommendation from cache", "userID", userID, "error", err)
	}

	if found {
		log.Info("Recent recommendation retrieved from cache", "userID", userID)
		return cached, nil
	}

	recommendation, err := gorm.G[*DailyRecommendation](tx).
		Where(DailyRecommendation{UserID: userID}).
		Preload("UserRelease.Release.Genres", nil).
		Preload("UserRelease.Release.Artists", nil).
		Order("created_at DESC").
		First(ctx)
	if err != nil {
		return nil, log.Err("failed to get most recent recommendation", err, "userID", userID)
	}

	err = database.NewCacheBuilder(r.cache, userID.String()).
		WithContext(ctx).
		WithHash(RECENT_RECOMMENDATION_CACHE_PREFIX).
		WithStruct(recommendation).
		WithTTL(RECENT_RECOMMENDATION_CACHE_EXPIRY).
		Set()
	if err != nil {
		log.Warn("failed to set recent recommendation in cache", "userID", userID, "error", err)
	}

	log.Info("Recent recommendation retrieved from database and cached", "userID", userID)
	return recommendation, nil
}

func (r *dailyRecommendationRepository) GetByID(
	ctx context.Context,
	tx *gorm.DB,
	recommendationID uuid.UUID,
	userID uuid.UUID,
) (*DailyRecommendation, error) {
	log := logger.New("dailyRecommendationRepository").TraceFromContext(ctx).Function("GetByID")

	recommendation, err := gorm.G[*DailyRecommendation](tx).
		Where(DailyRecommendation{BaseUUIDModel: BaseUUIDModel{ID: recommendationID}, UserID: userID}).
		First(ctx)
	if err != nil {
		return nil, log.Err(
			"failed to get recommendation by ID",
			err,
			"recommendationID",
			recommendationID,
			"userID",
			userID,
		)
	}

	return recommendation, nil
}

func (r *dailyRecommendationRepository) CreateRecommendation(
	ctx context.Context,
	tx *gorm.DB,
	recommendation *DailyRecommendation,
) error {
	log := logger.New("dailyRecommendationRepository").TraceFromContext(ctx).Function("CreateRecommendation")

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
	r.clearRecentRecommendationCache(ctx, recommendation.UserID)

	return nil
}

func (r *dailyRecommendationRepository) MarkAsListened(
	ctx context.Context,
	tx *gorm.DB,
	recommendationID uuid.UUID,
	userID uuid.UUID,
) error {
	log := logger.New("dailyRecommendationRepository").TraceFromContext(ctx).Function("MarkAsListened")

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
	r.clearRecentRecommendationCache(ctx, userID)

	if err := r.ClearUserStreakCache(ctx, userID); err != nil {
		log.Warn("failed to clear streak cache", "userID", userID, "error", err)
	}

	return nil
}

func (r *dailyRecommendationRepository) clearUserRecommendationCache(
	ctx context.Context,
	userID uuid.UUID,
) {
	log := logger.New("dailyRecommendationRepository").TraceFromContext(ctx).Function("clearUserRecommendationCache")

	err := database.NewCacheBuilder(r.cache, userID.String()).
		WithContext(ctx).
		WithHash(DAILY_RECOMMENDATIONS_CACHE_PREFIX).
		Delete()
	if err != nil {
		log.Warn("failed to clear user recommendation cache", "userID", userID, "error", err)
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

func (r *dailyRecommendationRepository) GetAllUserRecommendations(
	ctx context.Context,
	tx *gorm.DB,
	userID uuid.UUID,
) ([]*DailyRecommendation, error) {
	log := logger.New("dailyRecommendationRepository").TraceFromContext(ctx).Function("GetAllUserRecommendations")

	recommendations, err := gorm.G[*DailyRecommendation](tx).
		Where(DailyRecommendation{UserID: userID}).
		Order("date DESC").
		Find(ctx)
	if err != nil {
		return nil, log.Err("failed to get all user recommendations", err, "userID", userID)
	}

	log.Info("retrieved all user recommendations", "userID", userID, "count", len(recommendations))
	return recommendations, nil
}

func (r *dailyRecommendationRepository) ClearUserRecommendationCache(
	ctx context.Context,
	userID uuid.UUID,
) error {
	log := logger.New("dailyRecommendationRepository").TraceFromContext(ctx).Function("ClearUserRecommendationCache")

	r.clearUserRecommendationCache(ctx, userID)

	log.Info("cleared user recommendation cache", "userID", userID)
	return nil
}

func (r *dailyRecommendationRepository) ClearUserStreakCache(
	ctx context.Context,
	userID uuid.UUID,
) error {
	log := logger.New("dailyRecommendationRepository").TraceFromContext(ctx).Function("ClearUserStreakCache")

	err := database.NewCacheBuilder(r.cache, userID.String()).
		WithContext(ctx).
		WithHash(constants.UserStreakCachePrefix).
		Delete()
	if err != nil {
		return log.Err("failed to clear user streak cache", err, "userID", userID)
	}

	log.Info("cleared user streak cache", "userID", userID)
	return nil
}

func (r *dailyRecommendationRepository) CalculateUserStreaks(
	ctx context.Context,
	tx *gorm.DB,
	userID uuid.UUID,
) (*StreakData, error) {
	log := logger.New("dailyRecommendationRepository").TraceFromContext(ctx).Function("CalculateUserStreaks")

	// Calculate streaks using CTEs:
	// 1. ordered_recs: Order recommendations by date DESC with row numbers
	// 2. current_streak_calc: Count consecutive listened from most recent date
	//    - Handles edge case: if all recommendations are listened, count all (not 0)
	// 3. consecutive_groups: Use gap-and-islands technique for grouping consecutive listened periods
	// 4. longest_streak_calc: Find maximum consecutive listened group across all time
	query := `
		WITH ordered_recs AS (
			SELECT
				date,
				listened_at,
				ROW_NUMBER() OVER (ORDER BY date DESC) as rn
			FROM daily_recommendations
			WHERE user_id = ?
			ORDER BY date DESC
		),
		current_streak_calc AS (
			SELECT
				CASE
					-- If most recent (rn=1) is not listened, streak is broken
					WHEN EXISTS(SELECT 1 FROM ordered_recs WHERE listened_at IS NULL AND rn = 1)
					THEN 0
					-- If all are listened (no unlistened records exist), count all
					WHEN NOT EXISTS(SELECT 1 FROM ordered_recs WHERE listened_at IS NULL)
					THEN (SELECT COUNT(*) FROM ordered_recs WHERE listened_at IS NOT NULL)
					-- Otherwise, count consecutive listened from start until first unlistened
					ELSE (
						SELECT COUNT(*)
						FROM ordered_recs
						WHERE rn < (SELECT MIN(rn) FROM ordered_recs WHERE listened_at IS NULL)
						AND listened_at IS NOT NULL
					)
				END as current_streak
		),
		consecutive_groups AS (
			SELECT
				date,
				listened_at,
				ROW_NUMBER() OVER (ORDER BY date) -
				ROW_NUMBER() OVER (PARTITION BY CASE WHEN listened_at IS NOT NULL THEN 1 ELSE 0 END ORDER BY date) as grp
			FROM daily_recommendations
			WHERE user_id = ?
		),
		longest_streak_calc AS (
			SELECT
				COALESCE(MAX(streak_length), 0) as longest_streak
			FROM (
				SELECT
					COUNT(*) as streak_length
				FROM consecutive_groups
				WHERE listened_at IS NOT NULL
				GROUP BY grp
			) streaks
		)
		SELECT
			(SELECT current_streak FROM current_streak_calc) as current_streak,
			(SELECT longest_streak FROM longest_streak_calc) as longest_streak
	`

	var result StreakData
	err := tx.WithContext(ctx).Raw(query, userID, userID).Scan(&result).Error
	if err != nil {
		return nil, log.Err("failed to calculate user streaks", err, "userID", userID)
	}

	log.Info(
		"calculated user streaks",
		"userID",
		userID,
		"currentStreak",
		result.CurrentStreak,
		"longestStreak",
		result.LongestStreak,
	)

	return &result, nil
}

func (r *dailyRecommendationRepository) GetUserStreakFromCache(
	ctx context.Context,
	userID uuid.UUID,
) (*StreakData, bool, error) {
	log := logger.New("dailyRecommendationRepository").TraceFromContext(ctx).Function("GetUserStreakFromCache")

	var cachedStreak *StreakData
	found, err := database.NewCacheBuilder(r.cache, userID.String()).
		WithContext(ctx).
		WithHash(constants.UserStreakCachePrefix).
		Get(&cachedStreak)
	if err != nil {
		log.Warn("failed to get streak from cache", "userID", userID, "error", err)
		return nil, false, err
	}

	if found {
		log.Info("streak retrieved from cache", "userID", userID)
	}

	return cachedStreak, found, nil
}

func (r *dailyRecommendationRepository) SetUserStreakCache(
	ctx context.Context,
	userID uuid.UUID,
	streakData *StreakData,
) error {
	log := logger.New("dailyRecommendationRepository").TraceFromContext(ctx).Function("SetUserStreakCache")

	err := database.NewCacheBuilder(r.cache, userID.String()).
		WithContext(ctx).
		WithHash(constants.UserStreakCachePrefix).
		WithStruct(streakData).
		WithTTL(constants.UserStreakCacheTTL).
		Set()
	if err != nil {
		log.Warn("failed to cache streak data", "userID", userID, "error", err)
		return err
	}

	return nil
}

func (r *dailyRecommendationRepository) clearRecentRecommendationCache(
	ctx context.Context,
	userID uuid.UUID,
) {
	log := logger.New("dailyRecommendationRepository").TraceFromContext(ctx).Function("clearRecentRecommendationCache")

	err := database.NewCacheBuilder(r.cache, userID.String()).
		WithContext(ctx).
		WithHash(RECENT_RECOMMENDATION_CACHE_PREFIX).
		Delete()
	if err != nil {
		log.Warn("failed to clear recent recommendation cache", "userID", userID, "error", err)
	}
}
