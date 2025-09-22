package repositories

import (
	"context"
	"waugzee/internal/database"
	"waugzee/internal/logger"
	. "waugzee/internal/models"

	"github.com/google/uuid"
)

type UserConfigurationRepository interface {
	GetByUserID(ctx context.Context, userID uuid.UUID) (*UserConfiguration, error)
	CreateOrUpdate(ctx context.Context, config *UserConfiguration) error
	Update(ctx context.Context, config *UserConfiguration) error
}

type userConfigurationRepository struct {
	db  database.DB
	log logger.Logger
}

func NewUserConfigurationRepository(db database.DB) UserConfigurationRepository {
	return &userConfigurationRepository{
		db:  db,
		log: logger.New("userConfigurationRepository"),
	}
}

func (r *userConfigurationRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*UserConfiguration, error) {
	log := r.log.Function("GetByUserID")

	var config UserConfiguration
	err := r.db.SQLWithContext(ctx).Where("user_id = ?", userID).First(&config).Error
	if err != nil {
		return nil, log.Err("failed to get user configuration", err)
	}

	return &config, nil
}

func (r *userConfigurationRepository) CreateOrUpdate(ctx context.Context, config *UserConfiguration) error {
	log := r.log.Function("CreateOrUpdate")

	// Try to find existing configuration
	var existing UserConfiguration
	err := r.db.SQLWithContext(ctx).Where("user_id = ?", config.UserID).First(&existing).Error

	if err != nil {
		// Configuration doesn't exist, create new one
		if err := r.db.SQLWithContext(ctx).Create(config).Error; err != nil {
			return log.Err("failed to create user configuration", err)
		}
		return nil
	}

	// Configuration exists, update it
	config.ID = existing.ID
	config.CreatedAt = existing.CreatedAt
	if err := r.db.SQLWithContext(ctx).Save(config).Error; err != nil {
		return log.Err("failed to update user configuration", err)
	}

	return nil
}

func (r *userConfigurationRepository) Update(ctx context.Context, config *UserConfiguration) error {
	log := r.log.Function("Update")

	if err := r.db.SQLWithContext(ctx).Save(config).Error; err != nil {
		return log.Err("failed to update user configuration", err)
	}

	return nil
}