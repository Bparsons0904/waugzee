package repositories

import (
	"context"
	"time"
	"waugzee/internal/database"
	"waugzee/internal/logger"
	. "waugzee/internal/models"
	"waugzee/internal/services"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	USER_CACHE_EXPIRY = 7 * 24 * time.Hour // 7 days
)

type UserRepository interface {
	GetByID(ctx context.Context, id string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByOIDCUserID(ctx context.Context, oidcUserID string) (*User, error)
	CreateFromOIDC(ctx context.Context, user *User) (*User, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id string) error
	FindOrCreateOIDCUser(ctx context.Context, user *User) (*User, error)
}

type userRepository struct {
	db  database.DB
	log logger.Logger
}

func New(db database.DB) UserRepository {
	return &userRepository{
		db:  db,
		log: logger.New("userRepository"),
	}
}

func (r *userRepository) getDB(ctx context.Context) *gorm.DB {
	if tx, ok := services.GetTransaction(ctx); ok {
		return tx
	}
	return r.db.SQLWithContext(ctx)
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

	db := r.getDB(ctx).Set("waugzee:cache", r.db.Cache.User)

	if err := db.Save(user).Error; err != nil {
		return log.Err("failed to update user", err, "user", user)
	}

	return nil
}

func (r *userRepository) Delete(ctx context.Context, id string) error {
	log := r.log.Function("Delete")

	// Get user first to clean up OIDC mapping cache
	var user User
	if err := r.getDB(ctx).First(&user, "id = ?", id).Error; err == nil {
		// Clean up OIDC mapping cache if user has OIDC ID
		if user.OIDCUserID != "" {
			oidcCacheKey := "oidc:" + user.OIDCUserID
			if err := database.NewCacheBuilder(r.db.Cache.User, oidcCacheKey).Delete(); err != nil {
				log.Warn(
					"failed to remove OIDC mapping from cache",
					"oidcUserID",
					user.OIDCUserID,
					"error",
					err,
				)
			}
		}
	}

	if err := r.getDB(ctx).Delete(&User{}, "id = ?", id).Error; err != nil {
		return log.Err("failed to delete user", err, "id", id)
	}

	if err := database.NewCacheBuilder(r.db.Cache.User, id).Delete(); err != nil {
		log.Warn("failed to remove user from cache", "userID", id, "error", err)
	}

	return nil
}

func (r *userRepository) getCacheByID(ctx context.Context, userID string, user *User) error {
	found, err := database.NewCacheBuilder(r.db.Cache.User, userID).Get(user)
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
	if err := database.NewCacheBuilder(r.db.Cache.User, user.ID).
		WithStruct(user).
		WithTTL(USER_CACHE_EXPIRY).
		WithContext(ctx).
		Set(); err != nil {
		return r.log.Function("addUserToCache").
			Err("failed to add user to cache", err, "user", user)
	}
	return nil
}

func (r *userRepository) getDBByID(ctx context.Context, userID string, user *User) error {
	log := r.log.Function("getDBByID")

	id, err := uuid.Parse(userID)
	if err != nil {
		return log.Err("failed to parse userID", err, "userID", userID)
	}

	if err := r.getDB(ctx).First(user, "id = ?", id).Error; err != nil {
		return log.Err("failed to get user by id", err, "id", userID)
	}

	return nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
	log := r.log.Function("GetByEmail")

	var user User
	if err := r.getDB(ctx).First(&user, "email = ?", email).Error; err != nil {
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
	oidcCacheKey := "oidc:" + oidcUserID
	found, err := database.NewCacheBuilder(r.db.Cache.User, oidcCacheKey).Get(&userUUID)
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
	if err := r.getDB(ctx).First(&user, "oidc_user_id = ?", oidcUserID).Error; err != nil {
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

func (r *userRepository) CreateFromOIDC(
	ctx context.Context,
	user *User,
) (*User, error) {
	log := r.log.Function("CreateFromOIDC")

	// Ensure defaults are set
	if !user.IsActive {
		user.IsActive = true
	}
	if user.LastLoginAt == nil {
		now := time.Now()
		user.LastLoginAt = &now
	}

	if err := r.getDB(ctx).Create(user).Error; err != nil {
		return nil, log.Err("failed to create OIDC user", err, "userID", user.OIDCUserID)
	}

	if err := r.addUserToCache(ctx, user); err != nil {
		log.Warn("failed to add user to cache", "userID", user.ID, "error", err)
	}

	// Cache OIDC ID to UUID mapping for faster future lookups
	oidcCacheKey := "oidc:" + user.OIDCUserID
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
		existingUser, err := r.GetByEmail(ctx, *user.Email)
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
	return r.CreateFromOIDC(ctx, user)
}
