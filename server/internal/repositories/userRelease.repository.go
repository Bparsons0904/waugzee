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
	USER_RELEASES_CACHE_PREFIX = "user_releases"
	USER_RELEASES_CACHE_EXPIRY = 24 * time.Hour
)

type UserReleaseRepository interface {
	CreateBatch(ctx context.Context, tx *gorm.DB, userReleases []*UserRelease) error
	UpdateBatch(ctx context.Context, tx *gorm.DB, userReleases []*UserRelease) error
	DeleteBatch(ctx context.Context, tx *gorm.DB, userID uuid.UUID, instanceIDs []int) error
	GetExistingByUser(
		ctx context.Context,
		tx *gorm.DB,
		userID uuid.UUID,
	) (map[int]*UserRelease, error)
	GetUserReleasesByFolderID(
		ctx context.Context,
		tx *gorm.DB,
		userID uuid.UUID,
		folderID int,
	) ([]*UserRelease, error)
}

type userReleaseRepository struct {
	cache database.CacheClient
	log   logger.Logger
}

func NewUserReleaseRepository(cache database.CacheClient) UserReleaseRepository {
	return &userReleaseRepository{
		cache: cache,
		log:   logger.New("userReleaseRepository"),
	}
}

func (r *userReleaseRepository) CreateBatch(
	ctx context.Context,
	tx *gorm.DB,
	userReleases []*UserRelease,
) error {
	log := r.log.Function("CreateBatch")

	if len(userReleases) == 0 {
		log.Info("No user releases to create")
		return nil
	}

	err := gorm.G[[]*UserRelease](tx).Create(ctx, &userReleases)
	if err != nil {
		return log.Err(
			"failed to create user releases",
			err,
			"count",
			len(userReleases),
		)
	}

	if len(userReleases) > 0 {
		r.clearUserReleasesCache(ctx, userReleases[0].UserID)
	}

	log.Info("Successfully created user releases", "count", len(userReleases))
	return nil
}

func (r *userReleaseRepository) UpdateBatch(
	ctx context.Context,
	tx *gorm.DB,
	userReleases []*UserRelease,
) error {
	log := r.log.Function("UpdateBatch")

	if len(userReleases) == 0 {
		log.Info("No user releases to update")
		return nil
	}

	for _, userRelease := range userReleases {
		if err := tx.WithContext(ctx).Save(userRelease).Error; err != nil {
			return log.Err(
				"failed to update user release",
				err,
				"instanceID", userRelease.InstanceID,
				"userID", userRelease.UserID,
			)
		}
	}

	if len(userReleases) > 0 {
		r.clearUserReleasesCache(ctx, userReleases[0].UserID)
	}

	log.Info("Successfully updated user releases", "count", len(userReleases))
	return nil
}

func (r *userReleaseRepository) DeleteBatch(
	ctx context.Context,
	tx *gorm.DB,
	userID uuid.UUID,
	instanceIDs []int,
) error {
	log := r.log.Function("DeleteBatch")

	if len(instanceIDs) == 0 {
		log.Info("No user releases to delete")
		return nil
	}

	rowsAffected, err := gorm.G[*UserRelease](tx).
		Where("user_id = ? AND instance_id IN ?", userID, instanceIDs).
		Delete(ctx)
	if err != nil {
		return log.Err(
			"failed to delete user releases",
			err,
			"userID", userID,
			"instanceCount", len(instanceIDs),
		)
	}

	r.clearAllUserReleasesCache(ctx, userID)

	log.Info("Successfully deleted user releases",
		"userID", userID,
		"deletedCount", rowsAffected,
		"requestedCount", len(instanceIDs))

	return nil
}

func (r *userReleaseRepository) GetExistingByUser(
	ctx context.Context,
	tx *gorm.DB,
	userID uuid.UUID,
) (map[int]*UserRelease, error) {
	log := r.log.Function("GetExistingByUser")

	userReleases, err := gorm.G[*UserRelease](tx).
		Where("user_id = ? AND active = ?", userID, true).
		Find(ctx)
	if err != nil {
		return nil, log.Err(
			"failed to get existing user releases",
			err,
			"userID", userID,
		)
	}

	result := make(map[int]*UserRelease, len(userReleases))
	for _, userRelease := range userReleases {
		result[userRelease.InstanceID] = userRelease
	}

	log.Info(
		"Retrieved existing user releases",
		"userID", userID,
		"count", len(result),
	)
	return result, nil
}

func userReleasesWithPreloads(userID uuid.UUID) func(db *gorm.Statement) {
	return func(db *gorm.Statement) {
		db.Preload("Release.Artists").
			Preload("Release.Genres").
			Preload("Release.Labels").
			Preload("PlayHistory", "user_id = ?", userID).
			Preload("PlayHistory.UserStylus.Stylus").
			Preload("CleaningHistory", "user_id = ?", userID)
	}
}

func (r *userReleaseRepository) GetUserReleasesByFolderID(
	ctx context.Context,
	tx *gorm.DB,
	userID uuid.UUID,
	folderID int,
) ([]*UserRelease, error) {
	log := r.log.Function("GetUserReleasesByFolderID")

	var cachedReleases []*UserRelease
	found, err := database.NewCacheBuilder(r.cache, userID.String()).
		WithContext(ctx).
		WithHash(USER_RELEASES_CACHE_PREFIX).
		Get(&cachedReleases)
	if err != nil {
		log.Warn("failed to get user releases from cache", "userID", userID, "folderID", folderID, "error", err)
	}

	if found {
		log.Info("User releases retrieved from cache", "userID", userID, "folderID", folderID, "count", len(cachedReleases))
		return cachedReleases, nil
	}

	query := gorm.G[*UserRelease](tx).
		Scopes(userReleasesWithPreloads(userID)).
		Where("user_id = ? AND active = ?", userID, true).
		Order("date_added DESC")

	if folderID != 0 {
		query = query.Where("folder_id = ?", folderID)
	}

	userReleases, err := query.
		Find(ctx)
	if err != nil {
		return nil, log.Err(
			"failed to get user releases by folder",
			err,
			"userID", userID,
			"folderID", folderID,
		)
	}

	err = database.NewCacheBuilder(r.cache, userID.String()).
		WithContext(ctx).
		WithHash(USER_RELEASES_CACHE_PREFIX).
		WithStruct(userReleases).
		WithTTL(USER_RELEASES_CACHE_EXPIRY).
		Set()
	if err != nil {
		log.Warn("failed to set user releases in cache", "userID", userID, "folderID", folderID, "error", err)
	}

	log.Info(
		"Retrieved user releases by folder from database and cached",
		"userID", userID,
		"folderID", folderID,
		"count", len(userReleases),
	)
	return userReleases, nil
}

func (r *userReleaseRepository) clearUserReleasesCache(
	ctx context.Context,
	userID uuid.UUID,
) {
	err := database.NewCacheBuilder(r.cache, userID.String()).
		WithContext(ctx).
		WithHash(USER_RELEASES_CACHE_PREFIX).
		Delete()
	if err != nil {
		r.log.Warn("failed to clear user releases cache", "userID", userID, "error", err)
	}
}

func (r *userReleaseRepository) clearAllUserReleasesCache(
	ctx context.Context,
	userID uuid.UUID,
) {
	r.clearUserReleasesCache(ctx, userID)
}
