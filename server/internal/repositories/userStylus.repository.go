package repositories

import (
	"context"
	"waugzee/internal/logger"
	. "waugzee/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserStylusRepository interface {
	GetUserStyluses(ctx context.Context, tx *gorm.DB, userID uuid.UUID) ([]*UserStylus, error)
	Create(ctx context.Context, tx *gorm.DB, userStylus *UserStylus) error
	Update(ctx context.Context, tx *gorm.DB, userStylus *UserStylus) error
	Delete(ctx context.Context, tx *gorm.DB, userID uuid.UUID, stylusID uuid.UUID) error
	UnsetAllPrimary(ctx context.Context, tx *gorm.DB, userID uuid.UUID) error
	GetByID(ctx context.Context, tx *gorm.DB, userID uuid.UUID, id uuid.UUID) (*UserStylus, error)
}

type userStylusRepository struct {
	log logger.Logger
}

func NewUserStylusRepository() UserStylusRepository {
	return &userStylusRepository{
		log: logger.New("userStylusRepository"),
	}
}

func (r *userStylusRepository) GetUserStyluses(
	ctx context.Context,
	tx *gorm.DB,
	userID uuid.UUID,
) ([]*UserStylus, error) {
	log := r.log.Function("GetUserStyluses")

	var styluses []*UserStylus
	if err := tx.WithContext(ctx).
		Preload("Stylus").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&styluses).Error; err != nil {
		return nil, log.Err("failed to get user styluses", err, "userID", userID)
	}

	return styluses, nil
}

func (r *userStylusRepository) Create(
	ctx context.Context,
	tx *gorm.DB,
	userStylus *UserStylus,
) error {
	log := r.log.Function("Create")

	if err := tx.WithContext(ctx).Create(userStylus).Error; err != nil {
		return log.Err(
			"failed to create user stylus",
			err,
			"userID",
			userStylus.UserID,
			"stylusID",
			userStylus.StylusID,
		)
	}

	return nil
}

func (r *userStylusRepository) Update(
	ctx context.Context,
	tx *gorm.DB,
	userStylus *UserStylus,
) error {
	log := r.log.Function("Update")

	if err := tx.WithContext(ctx).Save(userStylus).Error; err != nil {
		return log.Err(
			"failed to update user stylus",
			err,
			"id",
			userStylus.ID,
			"userID",
			userStylus.UserID,
		)
	}

	return nil
}

func (r *userStylusRepository) Delete(
	ctx context.Context,
	tx *gorm.DB,
	userID uuid.UUID,
	stylusID uuid.UUID,
) error {
	log := r.log.Function("Delete")

	result := tx.WithContext(ctx).
		Where("user_id = ? AND id = ?", userID, stylusID).
		Delete(&UserStylus{})

	if result.Error != nil {
		return log.Err(
			"failed to delete user stylus",
			result.Error,
			"userID",
			userID,
			"stylusID",
			stylusID,
		)
	}

	if result.RowsAffected == 0 {
		return log.Error(
			"user stylus not found or not owned by user",
			"userID",
			userID,
			"stylusID",
			stylusID,
		)
	}

	return nil
}

func (r *userStylusRepository) UnsetAllPrimary(
	ctx context.Context,
	tx *gorm.DB,
	userID uuid.UUID,
) error {
	log := r.log.Function("UnsetAllPrimary")

	if err := tx.WithContext(ctx).
		Model(&UserStylus{}).
		Where("user_id = ? AND is_active = ?", userID, true).
		Update("is_active", false).Error; err != nil {
		return log.Err("failed to unset all primary styluses", err, "userID", userID)
	}

	return nil
}

func (r *userStylusRepository) GetByID(
	ctx context.Context,
	tx *gorm.DB,
	userID uuid.UUID,
	id uuid.UUID,
) (*UserStylus, error) {
	log := r.log.Function("GetByID")

	var userStylus UserStylus
	if err := tx.WithContext(ctx).
		Preload("Stylus").
		Where("id = ? AND user_id = ?", id, userID).
		First(&userStylus).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, log.Error("user stylus not found", "id", id, "userID", userID)
		}
		return nil, log.Err("failed to get user stylus by ID", err, "id", id, "userID", userID)
	}

	return &userStylus, nil
}
