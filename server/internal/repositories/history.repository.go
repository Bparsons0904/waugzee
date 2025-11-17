package repositories

import (
	"context"
	"fmt"
	"time"
	"waugzee/internal/constants"
	"waugzee/internal/database"
	"waugzee/internal/logger"
	. "waugzee/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	PLAY_HISTORY_CACHE_PREFIX     = "play_history"
	CLEANING_HISTORY_CACHE_PREFIX = "cleaning_history"
	HISTORY_CACHE_EXPIRY          = 24 * time.Hour
)

type HistoryRepository interface {
	CreatePlayHistory(ctx context.Context, tx *gorm.DB, playHistory *PlayHistory) error
	GetUserPlayHistory(
		ctx context.Context,
		tx *gorm.DB,
		userID uuid.UUID,
		limit int,
	) ([]*PlayHistory, error)
	UpdatePlayHistory(
		ctx context.Context,
		tx *gorm.DB,
		playHistoryID uuid.UUID,
		updates map[string]any,
	) (*PlayHistory, error)
	DeletePlayHistory(
		ctx context.Context,
		tx *gorm.DB,
		userID uuid.UUID,
		playHistoryID uuid.UUID,
	) error

	CreateCleaningHistory(ctx context.Context, tx *gorm.DB, cleaningHistory *CleaningHistory) error
	GetUserCleaningHistory(
		ctx context.Context,
		tx *gorm.DB,
		userID uuid.UUID,
		limit int,
	) ([]*CleaningHistory, error)
	UpdateCleaningHistory(
		ctx context.Context,
		tx *gorm.DB,
		cleaningHistoryID uuid.UUID,
		updates map[string]any,
	) (*CleaningHistory, error)
	DeleteCleaningHistory(
		ctx context.Context,
		tx *gorm.DB,
		userID uuid.UUID,
		cleaningHistoryID uuid.UUID,
	) error

	ClearUserHistoryCache(ctx context.Context, userID uuid.UUID) error
}

type historyRepository struct {
	cache database.CacheClient
	log   logger.Logger
}

func NewHistoryRepository(cache database.CacheClient) HistoryRepository {
	return &historyRepository{
		cache: cache,
		log:   logger.New("historyRepository"),
	}
}

func (r *historyRepository) CreatePlayHistory(
	ctx context.Context,
	tx *gorm.DB,
	playHistory *PlayHistory,
) error {
	log := r.log.Function("CreatePlayHistory")

	err := gorm.G[PlayHistory](tx).Create(ctx, playHistory)
	if err != nil {
		return log.Err(
			"failed to create user stylus",
			err,
			"userID",
			playHistory.UserID,
			"userReleaseID",
			playHistory.UserReleaseID,
		)
	}

	r.clearUserPlayHistoryCache(ctx, playHistory.UserID)
	r.clearAllUserReleasesCache(ctx, playHistory.UserID)
	r.clearUserStreakCache(ctx, playHistory.UserID)

	return nil
}

func (r *historyRepository) GetUserPlayHistory(
	ctx context.Context,
	tx *gorm.DB,
	userID uuid.UUID,
	limit int,
) ([]*PlayHistory, error) {
	log := r.log.Function("GetUserPlayHistory")

	var cached []*PlayHistory
	found, err := database.NewCacheBuilder(r.cache, userID.String()).
		WithContext(ctx).
		WithHash(PLAY_HISTORY_CACHE_PREFIX).
		Get(&cached)
	if err != nil {
		log.Warn("failed to get play history from cache", "userID", userID, "error", err)
	}

	if found {
		log.Info("Play history retrieved from cache", "userID", userID, "count", len(cached))
		return cached, nil
	}

	playHistory, err := gorm.G[*PlayHistory](tx).
		Preload("UserRelease.Release.Genres", nil).
		Preload("UserRelease.Release.Artists", nil).
		Preload("UserStylus.Stylus", nil).
		Where(PlayHistory{UserID: userID}).
		Order("played_at DESC").
		Limit(limit).
		Find(ctx)
	if err != nil {
		return nil, log.Err("failed to get user play history", err, "userID", userID)
	}

	err = database.NewCacheBuilder(r.cache, userID.String()).
		WithContext(ctx).
		WithHash(PLAY_HISTORY_CACHE_PREFIX).
		WithStruct(playHistory).
		WithTTL(HISTORY_CACHE_EXPIRY).
		Set()
	if err != nil {
		log.Warn("failed to set play history in cache", "userID", userID, "error", err)
	}

	log.Info(
		"Play history retrieved from database and cached",
		"userID",
		userID,
		"count",
		len(playHistory),
	)

	return playHistory, nil
}

func (r *historyRepository) UpdatePlayHistory(
	ctx context.Context,
	tx *gorm.DB,
	playHistoryID uuid.UUID,
	updates map[string]any,
) (*PlayHistory, error) {
	log := r.log.Function("UpdatePlayHistory")

	result := tx.Model(&PlayHistory{}).Where("id = ?", playHistoryID).Updates(updates)
	if result.Error != nil {
		return nil, log.Err(
			"failed to update play history",
			result.Error,
			"playHistoryID",
			playHistoryID,
		)
	}

	playHistory, err := gorm.G[*PlayHistory](tx).
		Preload("UserRelease.Release.Genres", nil).
		Preload("UserRelease.Release.Artists", nil).
		Preload("UserStylus.Stylus", nil).
		Where(PlayHistory{BaseUUIDModel: BaseUUIDModel{ID: playHistoryID}}).
		First(ctx)
	if err != nil {
		return nil, log.Err(
			"failed to retrieve updated play history",
			err,
			"playHistoryID",
			playHistoryID,
		)
	}

	r.clearUserPlayHistoryCache(ctx, playHistory.UserID)
	r.clearAllUserReleasesCache(ctx, playHistory.UserID)
	r.clearUserStreakCache(ctx, playHistory.UserID)

	return playHistory, nil
}

func (r *historyRepository) DeletePlayHistory(
	ctx context.Context,
	tx *gorm.DB,
	userID uuid.UUID,
	playHistoryID uuid.UUID,
) error {
	log := r.log.Function("DeletePlayHistory")

	rowsAffected, err := gorm.G[*PlayHistory](tx).
		Where("user_id = ? AND id = ?", userID, playHistoryID).
		Delete(ctx)
	if err != nil {
		return log.Err(
			"failed to delete play history",
			err,
			"userID",
			userID,
			"playHistoryID",
			playHistoryID,
		)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("play history not found or not owned by user")
	}

	r.clearUserPlayHistoryCache(ctx, userID)
	r.clearAllUserReleasesCache(ctx, userID)
	r.clearUserStreakCache(ctx, userID)

	return nil
}

func (r *historyRepository) CreateCleaningHistory(
	ctx context.Context,
	tx *gorm.DB,
	cleaningHistory *CleaningHistory,
) error {
	log := r.log.Function("CreateCleaningHistory")

	err := gorm.G[CleaningHistory](tx).Create(ctx, cleaningHistory)
	if err != nil {
		return log.Err(
			"failed to create cleaning history",
			err,
			"userID",
			cleaningHistory.UserID,
			"userReleaseID",
			cleaningHistory.UserReleaseID,
		)
	}

	r.clearUserCleaningHistoryCache(ctx, cleaningHistory.UserID)
	r.clearAllUserReleasesCache(ctx, cleaningHistory.UserID)

	return nil
}

func (r *historyRepository) GetUserCleaningHistory(
	ctx context.Context,
	tx *gorm.DB,
	userID uuid.UUID,
	limit int,
) ([]*CleaningHistory, error) {
	log := r.log.Function("GetUserCleaningHistory")

	var cached []*CleaningHistory
	found, err := database.NewCacheBuilder(r.cache, userID.String()).
		WithContext(ctx).
		WithHash(CLEANING_HISTORY_CACHE_PREFIX).
		Get(&cached)
	if err != nil {
		log.Warn("failed to get cleaning history from cache", "userID", userID, "error", err)
	}

	if found {
		log.Info("Cleaning history retrieved from cache", "userID", userID, "count", len(cached))
		return cached, nil
	}

	cleaningHistory, err := gorm.G[*CleaningHistory](tx).
		Preload("UserRelease.Release.Genres", nil).
		Preload("UserRelease.Release.Artists", nil).
		Where(CleaningHistory{UserID: userID}).
		Order("cleaned_at DESC").
		Limit(limit).
		Find(ctx)
	if err != nil {
		return nil, log.Err("failed to get user cleaning history", err, "userID", userID)
	}

	err = database.NewCacheBuilder(r.cache, userID.String()).
		WithContext(ctx).
		WithHash(CLEANING_HISTORY_CACHE_PREFIX).
		WithStruct(cleaningHistory).
		WithTTL(HISTORY_CACHE_EXPIRY).
		Set()
	if err != nil {
		log.Warn("failed to set cleaning history in cache", "userID", userID, "error", err)
	}

	log.Info(
		"Cleaning history retrieved from database and cached",
		"userID",
		userID,
		"count",
		len(cleaningHistory),
	)

	return cleaningHistory, nil
}

func (r *historyRepository) UpdateCleaningHistory(
	ctx context.Context,
	tx *gorm.DB,
	cleaningHistoryID uuid.UUID,
	updates map[string]any,
) (*CleaningHistory, error) {
	log := r.log.Function("UpdateCleaningHistory")

	result := tx.Model(&CleaningHistory{}).Where("id = ?", cleaningHistoryID).Updates(updates)
	if result.Error != nil {
		return nil, log.Err(
			"failed to update cleaning history",
			result.Error,
			"cleaningHistoryID",
			cleaningHistoryID,
		)
	}

	cleaningHistory, err := gorm.G[*CleaningHistory](tx).
		Preload("UserRelease.Release.Genres", nil).
		Preload("UserRelease.Release.Artists", nil).
		Where(CleaningHistory{BaseUUIDModel: BaseUUIDModel{ID: cleaningHistoryID}}).
		First(ctx)
	if err != nil {
		return nil, log.Err(
			"failed to retrieve updated cleaning history",
			err,
			"cleaningHistoryID",
			cleaningHistoryID,
		)
	}

	r.clearUserCleaningHistoryCache(ctx, cleaningHistory.UserID)
	r.clearAllUserReleasesCache(ctx, cleaningHistory.UserID)

	return cleaningHistory, nil
}

func (r *historyRepository) DeleteCleaningHistory(
	ctx context.Context,
	tx *gorm.DB,
	userID uuid.UUID,
	cleaningHistoryID uuid.UUID,
) error {
	log := r.log.Function("DeleteCleaningHistory")

	rowsAffected, err := gorm.G[*CleaningHistory](tx).
		Where("user_id = ? AND id = ?", userID, cleaningHistoryID).
		Delete(ctx)
	if err != nil {
		return log.Err(
			"failed to delete cleaning history",
			err,
			"userID",
			userID,
			"cleaningHistoryID",
			cleaningHistoryID,
		)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("cleaning history not found or not owned by user")
	}

	r.clearUserCleaningHistoryCache(ctx, userID)
	r.clearAllUserReleasesCache(ctx, userID)

	return nil
}

func (r *historyRepository) clearUserPlayHistoryCache(ctx context.Context, userID uuid.UUID) {
	err := database.NewCacheBuilder(r.cache, userID.String()).
		WithContext(ctx).
		WithHash(PLAY_HISTORY_CACHE_PREFIX).
		Delete()
	if err != nil {
		r.log.Warn("failed to clear user play history cache", "userID", userID, "error", err)
	}
}

func (r *historyRepository) clearUserCleaningHistoryCache(
	ctx context.Context,
	userID uuid.UUID,
) {
	err := database.NewCacheBuilder(r.cache, userID.String()).
		WithContext(ctx).
		WithHash(CLEANING_HISTORY_CACHE_PREFIX).
		Delete()
	if err != nil {
		r.log.Warn("failed to clear user cleaning history cache", "userID", userID, "error", err)
	}
}

func (r *historyRepository) ClearUserHistoryCache(ctx context.Context, userID uuid.UUID) error {
	log := r.log.Function("ClearUserHistoryCache")

	r.clearUserPlayHistoryCache(ctx, userID)
	r.clearUserCleaningHistoryCache(ctx, userID)

	log.Info("cleared user history cache", "userID", userID)
	return nil
}

func (r *historyRepository) clearAllUserReleasesCache(ctx context.Context, userID uuid.UUID) {
	cachePattern := fmt.Sprintf("%s:*", userID.String())
	r.log.Debug("clearing all user_releases caches", "userID", userID, "pattern", cachePattern)
}

func (r *historyRepository) clearUserStreakCache(ctx context.Context, userID uuid.UUID) {
	err := database.NewCacheBuilder(r.cache, userID.String()).
		WithContext(ctx).
		WithHash(constants.UserStreakCachePrefix).
		Delete()
	if err != nil {
		r.log.Warn("failed to clear user streak cache", "userID", userID, "error", err)
	}
}
