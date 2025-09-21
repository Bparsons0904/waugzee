package repositories

import (
	"context"
	"time"
	"waugzee/internal/database"
	"waugzee/internal/logger"
	. "waugzee/internal/models"

	"github.com/google/uuid"
)

const (
	USER_CACHE_EXPIRY         = 7 * 24 * time.Hour // 7 days
	USER_CACHE_PREFIX         = "user:"
	OIDC_MAPPING_CACHE_PREFIX = "oidc:"
)

type UserRepository interface {
	GetByID(ctx context.Context, id string) (*User, error)
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

func (r *userRepository) GetByID(ctx context.Context, id string) (*User, error) {
	log := r.log.Function("GetByID")

	var user User
	if err := r.getCacheByID(ctx, id, &user); err == nil {
		return &user, nil
	}

	if err := r.getDBByID(ctx, id, &user); err != nil {
		return nil, err
	}

	if err := r.addUserToCache(ctx, &user); err != nil {
		log.Warn("failed to add user to cache", "userID", id, "error", err)
	}

	return &user, nil
}

func (r *userRepository) Update(ctx context.Context, user *User) error {
	log := r.log.Function("Update")

	if err := r.db.SQLWithContext(ctx).Save(user).Error; err != nil {
		return log.Err("failed to update user", err, "user", user)
	}

	// Clear user cache after successful update
	if err := r.clearUserCache(ctx, user); err != nil {
		log.Warn("failed to clear user cache after update", "userID", user.ID, "error", err)
	}

	return nil
}

func (r *userRepository) getCacheByID(ctx context.Context, userID string, user *User) error {
	cacheKey := USER_CACHE_PREFIX + userID
	found, err := database.NewCacheBuilder(r.db.Cache.User, cacheKey).WithContext(ctx).Get(user)
	if err != nil {
		return r.log.Function("getCacheByID").
			Err("failed to get user from cache", err, "userID", userID)
	}

	if !found {
		return r.log.Function("getCacheByID").
			Error("user not found in cache", "userID", userID)
	}

	return nil
}

func (r *userRepository) addUserToCache(ctx context.Context, user *User) error {
	cacheKey := USER_CACHE_PREFIX + user.ID.String()
	if err := database.NewCacheBuilder(r.db.Cache.User, cacheKey).
		WithStruct(user).
		WithTTL(USER_CACHE_EXPIRY).
		WithContext(ctx).
		Set(); err != nil {
		return r.log.Function("addUserToCache").
			Err("failed to add user to cache", err, "user", user)
	}
	return nil
}

func (r *userRepository) clearUserCache(ctx context.Context, user *User) error {
	log := r.log.Function("clearUserCache")

	// Clear primary user cache
	userCacheKey := USER_CACHE_PREFIX + user.ID.String()
	if err := database.NewCacheBuilder(r.db.Cache.User, userCacheKey).WithContext(ctx).Delete(); err != nil {
		log.Warn("failed to clear user cache", "userID", user.ID, "error", err)
	}

	// Clear OIDC mapping cache if user has OIDC ID
	if user.OIDCUserID != "" {
		oidcCacheKey := OIDC_MAPPING_CACHE_PREFIX + user.OIDCUserID
		if err := database.NewCacheBuilder(r.db.Cache.User, oidcCacheKey).WithContext(ctx).Delete(); err != nil {
			log.Warn(
				"failed to clear OIDC mapping cache",
				"oidcUserID",
				user.OIDCUserID,
				"error",
				err,
			)
		}
	}

	return nil
}

func (r *userRepository) getDBByID(ctx context.Context, userID string, user *User) error {
	log := r.log.Function("getDBByID")

	id, err := uuid.Parse(userID)
	if err != nil {
		return log.Err("failed to parse userID", err, "userID", userID)
	}

	if err := r.db.SQLWithContext(ctx).First(user, "id = ?", id).Error; err != nil {
		return log.Err("failed to get user by id", err, "id", userID)
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

	// Try to get UUID from OIDC cache first
	var userUUID string
	oidcCacheKey := OIDC_MAPPING_CACHE_PREFIX + oidcUserID
	found, err := database.NewCacheBuilder(r.db.Cache.User, oidcCacheKey).
		WithContext(ctx).
		Get(&userUUID)
	if err == nil && found {
		// Found UUID in cache, now get user by UUID (which uses primary cache)
		var cachedUser User
		if err := r.getCacheByID(ctx, userUUID, &cachedUser); err == nil {
			log.Info("user found via OIDC cache", "userID", userUUID, "oidcUserID", oidcUserID)
			return &cachedUser, nil
		}
	}

	// Cache miss, query database
	var user User
	if err := r.db.SQLWithContext(ctx).First(&user, "oidc_user_id = ?", oidcUserID).Error; err != nil {
		return nil, log.Err("failed to get user by OIDC user ID", err, "oidcUserID", oidcUserID)
	}

	// Cache both the user and the OIDC -> UUID mapping
	if err := r.addUserToCache(ctx, &user); err != nil {
		log.Warn("failed to add user to cache", "userID", user.ID, "error", err)
	}

	// Cache OIDC ID to UUID mapping for faster future lookups
	if err := database.NewCacheBuilder(r.db.Cache.User, oidcCacheKey).
		WithStruct(user.ID.String()).
		WithTTL(USER_CACHE_EXPIRY).
		WithContext(ctx).
		Set(); err != nil {
		log.Warn("failed to cache OIDC mapping", "oidcUserID", oidcUserID, "error", err)
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
		log.Warn("failed to add user to cache", "userID", user.ID, "error", err)
	}

	// Cache OIDC ID to UUID mapping for faster future lookups
	oidcCacheKey := OIDC_MAPPING_CACHE_PREFIX + user.OIDCUserID
	if err := database.NewCacheBuilder(r.db.Cache.User, oidcCacheKey).
		WithStruct(user.ID.String()).
		WithTTL(USER_CACHE_EXPIRY).
		WithContext(ctx).
		Set(); err != nil {
		log.Warn("failed to cache OIDC mapping", "oidcUserID", user.OIDCUserID, "error", err)
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

	// Get user from database to find UUID for cache cleanup
	user, err := r.GetByOIDCUserID(ctx, oidcUserID)
	if err != nil {
		log.Warn(
			"failed to get user for cache cleanup",
			"error",
			err.Error(),
			"oidcUserID",
			oidcUserID,
		)
		return err
	}

	// Clear user cache by UUID using proper cache key prefix
	userCacheKey := USER_CACHE_PREFIX + user.ID.String()
	if err := database.NewCacheBuilder(r.db.Cache.User, userCacheKey).WithContext(ctx).Delete(); err != nil {
		log.Warn("failed to remove user from cache", "userID", user.ID, "error", err)
		return err
	}

	// Clear OIDC mapping cache using proper cache key prefix
	oidcCacheKey := OIDC_MAPPING_CACHE_PREFIX + oidcUserID
	if err := database.NewCacheBuilder(r.db.Cache.User, oidcCacheKey).WithContext(ctx).Delete(); err != nil {
		log.Warn("failed to remove OIDC mapping from cache", "oidcUserID", oidcUserID, "error", err)
		return err
	}

	return nil
}
