package repositories

import (
	"context"
	"waugzee/internal/logger"
	. "waugzee/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserConfigurationRepository interface {
	GetByUserID(ctx context.Context, tx *gorm.DB, userID uuid.UUID) (*UserConfiguration, error)
	Update(ctx context.Context, tx *gorm.DB, config *UserConfiguration, userRepo UserRepository) error
	CreateOrUpdate(ctx context.Context, tx *gorm.DB, config *UserConfiguration, userRepo UserRepository) error
}

type userConfigurationRepository struct {
	log logger.Logger
}

func NewUserConfigurationRepository() UserConfigurationRepository {
	return &userConfigurationRepository{
		log: logger.New("userConfigurationRepository"),
	}
}

func (r *userConfigurationRepository) GetByUserID(ctx context.Context, tx *gorm.DB, userID uuid.UUID) (*UserConfiguration, error) {
	log := r.log.Function("GetByUserID")

	var config UserConfiguration
	err := tx.WithContext(ctx).Where("user_id = ?", userID).First(&config).Error
	if err != nil {
		return nil, log.Err("failed to get user configuration", err)
	}

	return &config, nil
}



func (r *userConfigurationRepository) Update(ctx context.Context, tx *gorm.DB, config *UserConfiguration, userRepo UserRepository) error {
	log := r.log.Function("Update")

	if err := tx.WithContext(ctx).Save(config).Error; err != nil {
		return log.Err("failed to update user configuration", err)
	}

	// Clear user cache after successful update
	if err := userRepo.ClearUserCacheByUserID(ctx, tx, config.UserID.String()); err != nil {
		log.Warn("failed to clear user cache after configuration update", "userID", config.UserID, "error", err)
	}

	return nil
}

func (r *userConfigurationRepository) CreateOrUpdate(ctx context.Context, tx *gorm.DB, config *UserConfiguration, userRepo UserRepository) error {
	log := r.log.Function("CreateOrUpdate")

	if err := tx.WithContext(ctx).Save(config).Error; err != nil {
		return log.Err("failed to create or update user configuration", err)
	}

	// Clear user cache after successful update
	if err := userRepo.ClearUserCacheByUserID(ctx, tx, config.UserID.String()); err != nil {
		log.Warn("failed to clear user cache after configuration update", "userID", config.UserID, "error", err)
	}

	return nil
}