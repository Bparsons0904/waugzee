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
	USER_CACHE_PREFIX = "user_oidc"
	USER_CACHE_EXPIRY = 7 * 24 * time.Hour
)

type UserRepository interface {
	GetByID(ctx context.Context, tx *gorm.DB, userID uuid.UUID) (*User, error)
	GetByOIDCUserID(ctx context.Context, tx *gorm.DB, oidcUserID string) (*User, error)
	GetAllUsers(ctx context.Context, tx *gorm.DB) ([]*User, error)
	Update(ctx context.Context, tx *gorm.DB, user *User) error
	FindOrCreateOIDCUser(ctx context.Context, tx *gorm.DB, user *User) (*User, error)
	ClearUserCacheByOIDC(ctx context.Context, oidcUserID string) error
	ClearUserCacheByUserID(ctx context.Context, tx *gorm.DB, userID string) error
}

type userRepository struct {
	cache database.DB
}

func NewUserRepository(cache database.DB) UserRepository {
	return &userRepository{
		cache: cache,
	}
}

func (r *userRepository) GetByID(
	ctx context.Context,
	tx *gorm.DB,
	userID uuid.UUID,
) (*User, error) {
	log := logger.NewWithContext(ctx, "userRepository").Function("GetByID")

	var user User
	if err := tx.WithContext(ctx).Preload("Configuration").First(&user, "id = ?", userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, log.Error("user not found", "userID", userID)
		}
		return nil, log.Err("failed to get user by ID", err, "userID", userID)
	}

	return &user, nil
}

func (r *userRepository) Update(ctx context.Context, tx *gorm.DB, user *User) error {
	log := logger.NewWithContext(ctx, "userRepository").Function("Update")

	if err := tx.WithContext(ctx).Save(user).Error; err != nil {
		return log.Err("failed to update user", err, "user", user)
	}

	if err := r.ClearUserCacheByOIDC(ctx, user.OIDCUserID); err != nil {
		log.Warn("failed to clear user cache after update", "userID", user.ID, "error", err)
	}

	return nil
}

func (r *userRepository) getCacheByOIDC(ctx context.Context, oidcUserID string, user *User) error {
	log := logger.NewWithContext(ctx, "userRepository").Function("getCacheByOIDC")

	found, err := database.NewCacheBuilder(r.cache.Cache.User, oidcUserID).
		WithContext(ctx).
		WithHash(USER_CACHE_PREFIX).
		Get(user)
	if err != nil {
		return log.Err("failed to get user from cache", err, "oidcUserID", oidcUserID)
	}

	if !found {
		return log.Error("user not found in cache", "oidcUserID", oidcUserID)
	}

	return nil
}

func (r *userRepository) addUserToCache(ctx context.Context, user *User) error {
	log := logger.NewWithContext(ctx, "userRepository").Function("addUserToCache")

	if err := database.NewCacheBuilder(r.cache.Cache.User, user.OIDCUserID).
		WithContext(ctx).
		WithHash(USER_CACHE_PREFIX).
		WithStruct(user).
		WithTTL(USER_CACHE_EXPIRY).
		Set(); err != nil {
		return log.Err("failed to add user to cache", err, "oidcUserID", user.OIDCUserID)
	}
	return nil
}

func (r *userRepository) GetByOIDCUserID(
	ctx context.Context,
	tx *gorm.DB,
	oidcUserID string,
) (*User, error) {
	log := logger.NewWithContext(ctx, "userRepository").Function("GetByOIDCUserID")

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
	log := logger.NewWithContext(ctx, "userRepository").Function("createFromOIDC")

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
	log := logger.NewWithContext(ctx, "userRepository").Function("FindOrCreateOIDCUser")

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
	log := logger.NewWithContext(ctx, "userRepository").Function("ClearUserCacheByOIDC")

	if err := database.NewCacheBuilder(r.cache.Cache.User, oidcUserID).
		WithContext(ctx).
		WithHash(USER_CACHE_PREFIX).
		Delete(); err != nil {
		log.Warn("failed to remove user from cache", "oidcUserID", oidcUserID, "error", err)
		return err
	}

	log.Info("cleared user cache", "oidcUserID", oidcUserID)
	return nil
}

func (r *userRepository) ClearUserCacheByUserID(
	ctx context.Context,
	tx *gorm.DB,
	userID string,
) error {
	log := logger.NewWithContext(ctx, "userRepository").Function("ClearUserCacheByUserID")

	var user User
	if err := tx.WithContext(ctx).First(&user, "id = ?", userID).Error; err != nil {
		return log.Err("failed to get user for cache clearing", err, "userID", userID)
	}

	if user.OIDCUserID == "" {
		return log.Error("user has no OIDC ID, cannot clear cache", "userID", userID)
	}

	return r.ClearUserCacheByOIDC(ctx, user.OIDCUserID)
}

func (r *userRepository) GetAllUsers(ctx context.Context, tx *gorm.DB) ([]*User, error) {
	log := logger.NewWithContext(ctx, "userRepository").Function("GetAllUsers")

	users, err := gorm.G[*User](tx).
		Where(User{IsActive: true}).
		Find(ctx)
	if err != nil {
		return nil, log.Err("failed to get all active users", err)
	}

	log.Info("retrieved all active users", "count", len(users))
	return users, nil
}
