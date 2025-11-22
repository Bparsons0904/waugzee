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

type StylusRepository interface {
	GetAllStyluses(ctx context.Context, tx *gorm.DB, userID *uuid.UUID) ([]*Stylus, error)
	GetUserStyluses(ctx context.Context, tx *gorm.DB, userID uuid.UUID) ([]*UserStylus, error)
	GetPrimaryUserStylus(ctx context.Context, tx *gorm.DB, userID uuid.UUID) (*UserStylus, error)
	GetStylusUsageHours(ctx context.Context, tx *gorm.DB, userID uuid.UUID) (map[uuid.UUID]float64, error)
	CreateCustomStylus(ctx context.Context, tx *gorm.DB, stylus *Stylus) error
	Create(ctx context.Context, tx *gorm.DB, userStylus *UserStylus) error
	Update(
		ctx context.Context,
		tx *gorm.DB,
		userID uuid.UUID,
		stylusID uuid.UUID,
		updatedStylus *UserStylus,
	) error
	Delete(ctx context.Context, tx *gorm.DB, userID uuid.UUID, stylusID uuid.UUID) error
	UnsetAllPrimary(ctx context.Context, tx *gorm.DB, userID uuid.UUID) error
	VerifyUserOwnership(
		ctx context.Context,
		tx *gorm.DB,
		stylusID uuid.UUID,
		userID uuid.UUID,
	) error

	ClearUserStylusCache(ctx context.Context, userID uuid.UUID) error
}

type stylusRepository struct {
	cache database.CacheClient
}

func NewStylusRepository(cache database.CacheClient) StylusRepository {
	return &stylusRepository{
		cache: cache,
	}
}

func (r *stylusRepository) GetAllStyluses(
	ctx context.Context,
	tx *gorm.DB,
	userID *uuid.UUID,
) ([]*Stylus, error) {
	log := logger.NewWithContext(ctx, "stylusRepository").Function("GetAllStyluses")

	var styluses []*Stylus
	query := tx.WithContext(ctx)

	if userID == nil {
		query = query.Where("user_generated_id IS NULL")
	} else {
		query = query.Where("user_generated_id IS NULL OR user_generated_id = ?", userID)
	}

	if err := query.
		Order("is_verified DESC, brand ASC, model ASC").
		Find(&styluses).Error; err != nil {
		return nil, log.Err("failed to get all styluses", err)
	}

	return styluses, nil
}

func (r *stylusRepository) GetUserStyluses(
	ctx context.Context,
	tx *gorm.DB,
	userID uuid.UUID,
) ([]*UserStylus, error) {
	log := logger.NewWithContext(ctx, "stylusRepository").Function("GetUserStyluses")

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

	styluses, err := gorm.G[*UserStylus](tx).
		Preload("Stylus", nil).
		Where(UserStylus{UserID: userID}).
		Order("created_at DESC").
		Find(ctx)
	if err != nil {
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

func (r *stylusRepository) GetPrimaryUserStylus(
	ctx context.Context,
	tx *gorm.DB,
	userID uuid.UUID,
) (*UserStylus, error) {
	log := logger.NewWithContext(ctx, "stylusRepository").Function("GetPrimaryUserStylus")

	userStylus, err := gorm.G[*UserStylus](tx).
		Preload("Stylus", nil).
		Where(UserStylus{UserID: userID, IsPrimary: true}).
		First(ctx)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, log.Err("failed to get primary user stylus", err, "userID", userID)
	}

	log.Info("Primary user stylus retrieved", "userID", userID, "stylusID", userStylus.StylusID)
	return userStylus, nil
}

func (r *stylusRepository) GetStylusUsageHours(
	ctx context.Context,
	tx *gorm.DB,
	userID uuid.UUID,
) (map[uuid.UUID]float64, error) {
	log := logger.NewWithContext(ctx, "stylusRepository").Function("GetStylusUsageHours")

	type usageResult struct {
		UserStylusID uuid.UUID `gorm:"column:user_stylus_id"`
		TotalSeconds int64     `gorm:"column:total_seconds"`
	}

	var results []usageResult
	err := tx.WithContext(ctx).
		Table("play_histories ph").
		Select("ph.user_stylus_id, COALESCE(SUM(r.total_duration), 0) as total_seconds").
		Joins("JOIN user_releases ur ON ph.user_release_id = ur.id").
		Joins("JOIN releases r ON ur.release_id = r.id").
		Where("ph.user_id = ? AND ph.user_stylus_id IS NOT NULL", userID).
		Group("ph.user_stylus_id").
		Scan(&results).Error
	if err != nil {
		return nil, log.Err("failed to get stylus usage hours", err, "userID", userID)
	}

	usageMap := make(map[uuid.UUID]float64)
	for _, r := range results {
		hours := float64(r.TotalSeconds) / 3600.0
		usageMap[r.UserStylusID] = hours
	}

	log.Info("Stylus usage hours calculated", "userID", userID, "stylusCount", len(usageMap))
	return usageMap, nil
}

func (r *stylusRepository) CreateCustomStylus(
	ctx context.Context,
	tx *gorm.DB,
	stylus *Stylus,
) error {
	log := logger.NewWithContext(ctx, "stylusRepository").Function("CreateCustomStylus")

	if err := tx.WithContext(ctx).Create(stylus).Error; err != nil {
		return log.Err(
			"failed to create custom stylus",
			err,
			"brand",
			stylus.Brand,
			"model",
			stylus.Model,
		)
	}

	log.Info(
		"Custom stylus created successfully",
		"id",
		stylus.ID,
		"userID",
		stylus.UserGeneratedID,
	)

	return nil
}

func (r *stylusRepository) Create(
	ctx context.Context,
	tx *gorm.DB,
	userStylus *UserStylus,
) error {
	log := logger.NewWithContext(ctx, "stylusRepository").Function("Create")

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

func (r *stylusRepository) Update(
	ctx context.Context,
	tx *gorm.DB,
	userID uuid.UUID,
	stylusID uuid.UUID,
	updatedStylus *UserStylus,
) error {
	log := logger.NewWithContext(ctx, "stylusRepository").Function("Update")

	rows, err := gorm.G[*UserStylus](tx).
		Where(BaseUUIDModel{ID: stylusID}).
		Updates(ctx, updatedStylus)
	if err != nil {
		return log.Err(
			"failed to update user stylus",
			err,
			"userID",
			userID,
			"stylusID",
			stylusID,
			"rows",
			rows,
		)
	}

	if rows == 0 {
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

func (r *stylusRepository) Delete(
	ctx context.Context,
	tx *gorm.DB,
	userID uuid.UUID,
	stylusID uuid.UUID,
) error {
	log := logger.NewWithContext(ctx, "stylusRepository").Function("Delete")

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

func (r *stylusRepository) UnsetAllPrimary(
	ctx context.Context,
	tx *gorm.DB,
	userID uuid.UUID,
) error {
	log := logger.NewWithContext(ctx, "stylusRepository").Function("UnsetAllPrimary")

	if _, err := gorm.G[UserStylus](tx).
		Where(UserStylus{UserID: userID}).
		Update(ctx, "is_primary", false); err != nil {
		return log.Err("failed to unset all primary styluses", err, "userID", userID)
	}

	r.clearUserStylusCache(ctx, userID)

	return nil
}

func (r *stylusRepository) VerifyUserOwnership(
	ctx context.Context,
	tx *gorm.DB,
	stylusID uuid.UUID,
	userID uuid.UUID,
) error {
	log := logger.NewWithContext(ctx, "stylusRepository").Function("VerifyUserOwnership")

	var userStylus UserStylus
	if err := tx.WithContext(ctx).
		Where("id = ? AND user_id = ?", stylusID, userID).
		First(&userStylus).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return gorm.ErrRecordNotFound
		}
		return log.Err(
			"failed to verify user stylus ownership",
			err,
			"stylusID",
			stylusID,
			"userID",
			userID,
		)
	}

	return nil
}

func (r *stylusRepository) clearUserStylusCache(ctx context.Context, userID uuid.UUID) {
	log := logger.NewWithContext(ctx, "stylusRepository").Function("clearUserStylusCache")

	err := database.NewCacheBuilder(r.cache, userID.String()).
		WithContext(ctx).
		WithHash(USER_STYLUSES_CACHE_PREFIX).
		Delete()
	if err != nil {
		log.Warn("failed to clear user stylus cache", "userID", userID, "error", err)
	}
}

func (r *stylusRepository) ClearUserStylusCache(ctx context.Context, userID uuid.UUID) error {
	log := logger.NewWithContext(ctx, "stylusRepository").Function("ClearUserStylusCache")

	r.clearUserStylusCache(ctx, userID)

	log.Info("cleared user stylus cache", "userID", userID)
	return nil
}
