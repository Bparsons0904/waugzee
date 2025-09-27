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
	GetExistingByUser(ctx context.Context, tx *gorm.DB, userID uuid.UUID) (map[int]*UserRelease, error)
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

	// Update each record individually to ensure proper handling
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

	result := tx.WithContext(ctx).
		Where("user_id = ? AND instance_id IN ?", userID, instanceIDs).
		Delete(&UserRelease{})

	if result.Error != nil {
		return log.Err(
			"failed to delete user releases",
			result.Error,
			"userID", userID,
			"instanceCount", len(instanceIDs),
		)
	}

	log.Info("Successfully deleted user releases",
		"userID", userID,
		"deletedCount", result.RowsAffected,
		"requestedCount", len(instanceIDs))

	return nil
}

func (r *userReleaseRepository) GetExistingByUser(
	ctx context.Context,
	tx *gorm.DB,
	userID uuid.UUID,
) (map[int]*UserRelease, error) {
	log := r.log.Function("GetExistingByUser")

	var userReleases []*UserRelease
	if err := tx.WithContext(ctx).
		Where("user_id = ? AND active = ?", userID, true).
		Find(&userReleases).Error; err != nil {
		return nil, log.Err(
			"failed to get existing user releases",
			err,
			"userID", userID,
		)
	}

	// Convert to map with InstanceID as key for efficient lookup
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

