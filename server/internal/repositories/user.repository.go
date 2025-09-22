package repositories

import (
	"context"
	"time"
	"waugzee/internal/database"
	"waugzee/internal/logger"
	. "waugzee/internal/models"

	"gorm.io/gorm"
)

const (
	USER_CACHE_EXPIRY = 7 * 24 * time.Hour // 7 days
	USER_CACHE_PREFIX = "user_oidc:"       // Single cache by OIDC ID
)

type UserRepository interface {
	GetByOIDCUserID(ctx context.Context, tx *gorm.DB, oidcUserID string) (*User, error)
	Update(ctx context.Context, tx *gorm.DB, user *User) error
	FindOrCreateOIDCUser(ctx context.Context, tx *gorm.DB, user *User) (*User, error)
	ClearUserCacheByOIDC(ctx context.Context, oidcUserID string) error
}

type userRepository struct {
	cache database.DB // Keep cache for cache operations
	log   logger.Logger
}

func NewUserRepository(cache database.DB) UserRepository {
	return &userRepository{
		cache: cache,
		log:   logger.New("userRepository"),
	}
}

func (r *userRepository) Update(ctx context.Context, tx *gorm.DB, user *User) error {
	log := r.log.Function("Update")

	if err := tx.WithContext(ctx).Save(user).Error; err != nil {
		return log.Err("failed to update user", err, "user", user)
	}

	if err := r.ClearUserCacheByOIDC(ctx, user.OIDCUserID); err != nil {
		log.Warn("failed to clear user cache after update", "userID", user.ID, "error", err)
	}

	return nil
}

func (r *userRepository) getCacheByOIDC(ctx context.Context, oidcUserID string, user *User) error {
	cacheKey := USER_CACHE_PREFIX + oidcUserID
	found, err := database.NewCacheBuilder(r.cache.Cache.User, cacheKey).WithContext(ctx).Get(user)
	if err != nil {
		return r.log.Function("getCacheByOIDC").
			Err("failed to get user from cache", err, "oidcUserID", oidcUserID)
	}

	if !found {
		return r.log.Function("getCacheByOIDC").
			Error("user not found in cache", "oidcUserID", oidcUserID)
	}

	return nil
}

func (r *userRepository) addUserToCache(ctx context.Context, user *User) error {
	cacheKey := USER_CACHE_PREFIX + user.OIDCUserID
	if err := database.NewCacheBuilder(r.cache.Cache.User, cacheKey).
		WithStruct(user).
		WithTTL(USER_CACHE_EXPIRY).
		WithContext(ctx).
		Set(); err != nil {
		return r.log.Function("addUserToCache").
			Err("failed to add user to cache", err, "oidcUserID", user.OIDCUserID)
	}
	return nil
}

func (r *userRepository) GetByOIDCUserID(ctx context.Context, tx *gorm.DB, oidcUserID string) (*User, error) {
	log := r.log.Function("GetByOIDCUserID")

	var cachedUser User
	if err := r.getCacheByOIDC(ctx, oidcUserID, &cachedUser); err == nil {
		log.Info("user found in cache", "oidcUserID", oidcUserID)
		return &cachedUser, nil
	}

	var user User
	if err := tx.WithContext(ctx).Preload("Configuration").First(&user, "oidc_user_id = ?", oidcUserID).Error; err != nil {
		return nil, log.Err("failed to get user by OIDC user ID", err, "oidcUserID", oidcUserID)
	}

	if err := r.addUserToCache(ctx, &user); err != nil {
		log.Warn("failed to add user to cache", "oidcUserID", oidcUserID, "error", err)
	}

	return &user, nil
}

func (r *userRepository) createFromOIDC(
	ctx context.Context,
	tx *gorm.DB,
	user *User,
) (*User, error) {
	log := r.log.Function("createFromOIDC")

	// Ensure defaults are set
	if !user.IsActive {
		user.IsActive = true
	}
	if user.LastLoginAt == nil {
		now := time.Now()
		user.LastLoginAt = &now
	}

	if err := tx.WithContext(ctx).Create(user).Error; err != nil {
		return nil, log.Err("failed to create OIDC user", err, "userID", user.OIDCUserID)
	}

	if err := r.addUserToCache(ctx, user); err != nil {
		log.Warn("failed to add user to cache", "oidcUserID", user.OIDCUserID, "error", err)
	}

	return user, nil
}

func (r *userRepository) FindOrCreateOIDCUser(
	ctx context.Context,
	tx *gorm.DB,
	user *User,
) (*User, error) {
	log := r.log.Function("FindOrCreateOIDCUser")

	// First try to find by OIDC user ID
	existingUser, err := r.GetByOIDCUserID(ctx, tx, user.OIDCUserID)
	if err == nil {
		// Update existing user with latest OIDC info using detailed method
		oidcProvider := "zitadel"
		if user.OIDCProvider != nil {
			oidcProvider = *user.OIDCProvider
		}
		existingUser.UpdateFromOIDC(
			user.OIDCUserID,
			user.Email,
			&user.DisplayName,
			user.FirstName,
			user.LastName,
			oidcProvider,
			user.OIDCProjectID,
			user.ProfileVerified,
		)

		if err := r.Update(ctx, tx, existingUser); err != nil {
			log.Warn("failed to update existing OIDC user", "error", err, "userID", existingUser.ID)
		}
		return existingUser, nil
	}

	return r.createFromOIDC(ctx, tx, user)
}

func (r *userRepository) ClearUserCacheByOIDC(ctx context.Context, oidcUserID string) error {
	log := r.log.Function("ClearUserCacheByOIDC")

	userCacheKey := USER_CACHE_PREFIX + oidcUserID
	if err := database.NewCacheBuilder(r.cache.Cache.User, userCacheKey).WithContext(ctx).Delete(); err != nil {
		log.Warn("failed to remove user from cache", "oidcUserID", oidcUserID, "error", err)
		return err
	}

	log.Info("cleared user cache", "oidcUserID", oidcUserID)
	return nil
}
