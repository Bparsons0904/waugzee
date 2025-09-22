package repositories

import (
	"context"
	"time"
	"waugzee/internal/database"
	"waugzee/internal/logger"
	. "waugzee/internal/models"
)

const (
	USER_CACHE_EXPIRY = 7 * 24 * time.Hour // 7 days
	USER_CACHE_PREFIX = "user_oidc:"       // Single cache by OIDC ID
)

type UserRepository interface {
	GetByOIDCUserID(ctx context.Context, oidcUserID string) (*User, error)
	Update(ctx context.Context, user *User) error
	FindOrCreateOIDCUser(ctx context.Context, user *User) (*User, error)
	ClearUserCacheByOIDC(ctx context.Context, oidcUserID string) error
}

type userRepository struct {
	db  database.DB
	log logger.Logger
}

func NewUserRepository(db database.DB) UserRepository {
	return &userRepository{
		db:  db,
		log: logger.New("userRepository"),
	}
}

func (r *userRepository) Update(ctx context.Context, user *User) error {
	log := r.log.Function("Update")

	if err := r.db.SQLWithContext(ctx).Save(user).Error; err != nil {
		return log.Err("failed to update user", err, "user", user)
	}

	// Clear user cache after successful update
	if err := r.ClearUserCacheByOIDC(ctx, user.OIDCUserID); err != nil {
		log.Warn("failed to clear user cache after update", "userID", user.ID, "error", err)
	}

	return nil
}

func (r *userRepository) getCacheByOIDC(ctx context.Context, oidcUserID string, user *User) error {
	cacheKey := USER_CACHE_PREFIX + oidcUserID
	found, err := database.NewCacheBuilder(r.db.Cache.User, cacheKey).WithContext(ctx).Get(user)
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
	if err := database.NewCacheBuilder(r.db.Cache.User, cacheKey).
		WithStruct(user).
		WithTTL(USER_CACHE_EXPIRY).
		WithContext(ctx).
		Set(); err != nil {
		return r.log.Function("addUserToCache").
			Err("failed to add user to cache", err, "oidcUserID", user.OIDCUserID)
	}
	return nil
}

func (r *userRepository) getByEmail(ctx context.Context, email string) (*User, error) {
	log := r.log.Function("getByEmail")

	var user User
	if err := r.db.SQLWithContext(ctx).First(&user, "email = ?", email).Error; err != nil {
		return nil, log.Err("failed to get user by email", err, "email", email)
	}

	if err := r.addUserToCache(ctx, &user); err != nil {
		log.Warn("failed to add user to cache", "userID", user.ID, "error", err)
	}

	return &user, nil
}

func (r *userRepository) GetByOIDCUserID(ctx context.Context, oidcUserID string) (*User, error) {
	log := r.log.Function("GetByOIDCUserID")

	// Try to get user from cache first (single-layer caching by OIDC ID)
	var cachedUser User
	if err := r.getCacheByOIDC(ctx, oidcUserID, &cachedUser); err == nil {
		log.Info("user found in cache", "oidcUserID", oidcUserID)
		return &cachedUser, nil
	}

	// Cache miss, query database
	var user User
	if err := r.db.SQLWithContext(ctx).Preload("Configuration").First(&user, "oidc_user_id = ?", oidcUserID).Error; err != nil {
		return nil, log.Err("failed to get user by OIDC user ID", err, "oidcUserID", oidcUserID)
	}

	// Cache the user by OIDC ID
	if err := r.addUserToCache(ctx, &user); err != nil {
		log.Warn("failed to add user to cache", "oidcUserID", oidcUserID, "error", err)
	}

	return &user, nil
}

func (r *userRepository) createFromOIDC(
	ctx context.Context,
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

	if err := r.db.SQLWithContext(ctx).Create(user).Error; err != nil {
		return nil, log.Err("failed to create OIDC user", err, "userID", user.OIDCUserID)
	}

	if err := r.addUserToCache(ctx, user); err != nil {
		log.Warn("failed to add user to cache", "oidcUserID", user.OIDCUserID, "error", err)
	}

	return user, nil
}

func (r *userRepository) FindOrCreateOIDCUser(
	ctx context.Context,
	user *User,
) (*User, error) {
	log := r.log.Function("FindOrCreateOIDCUser")

	// First try to find by OIDC user ID
	existingUser, err := r.GetByOIDCUserID(ctx, user.OIDCUserID)
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

		if err := r.Update(ctx, existingUser); err != nil {
			log.Warn("failed to update existing OIDC user", "error", err, "userID", existingUser.ID)
		}
		return existingUser, nil
	}

	// If not found by OIDC ID, try by email (in case user exists but wasn't created via OIDC)
	if user.Email != nil && *user.Email != "" {
		existingUser, err := r.getByEmail(ctx, *user.Email)
		if err == nil && !existingUser.IsOIDCUser() {
			// Link existing user to OIDC
			existingUser.OIDCUserID = user.OIDCUserID
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
			if err := r.Update(ctx, existingUser); err != nil {
				return nil, log.Err(
					"failed to link existing user to OIDC",
					err,
					"userID",
					existingUser.ID,
				)
			}
			return existingUser, nil
		}
	}

	// Create new OIDC user
	return r.createFromOIDC(ctx, user)
}

// ClearUserCacheByOIDC clears user cache by OIDC user ID
func (r *userRepository) ClearUserCacheByOIDC(ctx context.Context, oidcUserID string) error {
	log := r.log.Function("ClearUserCacheByOIDC")

	// Clear user cache directly by OIDC ID (single-layer caching)
	userCacheKey := USER_CACHE_PREFIX + oidcUserID
	if err := database.NewCacheBuilder(r.db.Cache.User, userCacheKey).WithContext(ctx).Delete(); err != nil {
		log.Warn("failed to remove user from cache", "oidcUserID", oidcUserID, "error", err)
		return err
	}

	log.Info("cleared user cache", "oidcUserID", oidcUserID)
	return nil
}
