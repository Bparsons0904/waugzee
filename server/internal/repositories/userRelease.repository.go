package repositories

import (
	"context"
	"waugzee/internal/logger"
	. "waugzee/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
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
	log logger.Logger
}

func NewUserReleaseRepository() UserReleaseRepository {
	return &userReleaseRepository{
		log: logger.New("userReleaseRepository"),
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
	if err := tx.WithContext(ctx).Create(&userReleases).Error; err != nil {
		return log.Err(
			"failed to create user releases",
			err,
			"count",
			len(userReleases),
		)
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
			Preload("Release").
			Preload("PlayHistory", "user_id = ?", userID).
			Preload("PlayHistory.UserStylus").
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

	log.Info(
		"Retrieved user releases by folder",
		"userID", userID,
		"folderID", folderID,
		"count", len(userReleases),
	)
	return userReleases, nil
}
