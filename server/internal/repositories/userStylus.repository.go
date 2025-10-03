package repositories

import (
	"context"
	"time"
	"waugzee/internal/database"
	"waugzee/internal/logger"
	. "waugzee/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	USER_STYLUSES_CACHE_PREFIX = "user_styluses"
	USER_STYLUSES_CACHE_EXPIRY = 7 * 24 * time.Hour
)

type UserStylusRepository interface {
	GetUserStyluses(ctx context.Context, tx *gorm.DB, userID uuid.UUID) ([]*UserStylus, error)
	Create(ctx context.Context, tx *gorm.DB, userStylus *UserStylus) error
	Update(
		ctx context.Context,
		tx *gorm.DB,
		userID uuid.UUID,
		stylusID uuid.UUID,
		updates map[string]any,
	) error
	Delete(ctx context.Context, tx *gorm.DB, userID uuid.UUID, stylusID uuid.UUID) error
	UnsetAllPrimary(ctx context.Context, tx *gorm.DB, userID uuid.UUID) error
}

type userStylusRepository struct {
	cache database.CacheClient
	log   logger.Logger
}

func NewUserStylusRepository(cache database.CacheClient) UserStylusRepository {
	return &userStylusRepository{
		cache: cache,
		log:   logger.New("userStylusRepository"),
	}
}

func (r *userStylusRepository) GetUserStyluses(
	ctx context.Context,
	tx *gorm.DB,
	userID uuid.UUID,
) ([]*UserStylus, error) {
	log := r.log.Function("GetUserStyluses")

	var cached []*UserStylus
	found, err := database.NewCacheBuilder(r.cache, userID.String()).
		WithContext(ctx).
		WithHash(USER_STYLUSES_CACHE_PREFIX).
		Get(&cached)
	if err != nil {
		log.Warn("failed to get user styluses from cache", "userID", userID, "error", err)
	}

	if found {
		log.Info("User styluses retrieved from cache", "userID", userID, "count", len(cached))
		return cached, nil
	}

	var styluses []*UserStylus
	if err = tx.WithContext(ctx).
		Preload("Stylus").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&styluses).Error; err != nil {
		return nil, log.Err("failed to get user styluses", err, "userID", userID)
	}

	err = database.NewCacheBuilder(r.cache, userID.String()).
		WithContext(ctx).
		WithHash(USER_STYLUSES_CACHE_PREFIX).
		WithStruct(styluses).
		WithTTL(USER_STYLUSES_CACHE_EXPIRY).
		Set()
	if err != nil {
		log.Warn("failed to set user styluses in cache", "userID", userID, "error", err)
	}

	log.Info(
		"User styluses retrieved from database and cached",
		"userID",
		userID,
		"count",
		len(styluses),
	)

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

	r.clearUserStylusCache(ctx, userStylus.UserID)

	return nil
}

func (r *userStylusRepository) Update(
	ctx context.Context,
	tx *gorm.DB,
	userID uuid.UUID,
	stylusID uuid.UUID,
	updates map[string]any,
) error {
	log := r.log.Function("Update")

	result := tx.WithContext(ctx).
		Model(&UserStylus{}).
		Where("id = ? AND user_id = ?", stylusID, userID).
		Updates(updates)

	if result.Error != nil {
		return log.Err(
			"failed to update user stylus",
			result.Error,
			"id",
			stylusID,
			"userID",
			userID,
		)
	}

	if result.RowsAffected == 0 {
		return log.Error(
			"user stylus not found or not owned by user",
			"id",
			stylusID,
			"userID",
			userID,
		)
	}

	r.clearUserStylusCache(ctx, userID)

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

	r.clearUserStylusCache(ctx, userID)

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

func (r *userStylusRepository) clearUserStylusCache(ctx context.Context, userID uuid.UUID) {
	err := database.NewCacheBuilder(r.cache, userID.String()).
		WithContext(ctx).
		WithHash(USER_STYLUSES_CACHE_PREFIX).
		Delete()
	if err != nil {
		r.log.Warn("failed to clear user stylus cache", "userID", userID, "error", err)
	}
}
