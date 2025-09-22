package models

import (
	"context"
	"waugzee/internal/database"
	"waugzee/internal/logger"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserConfiguration struct {
	BaseUUIDModel
	UserID           uuid.UUID  `gorm:"type:uuid;not null;index" json:"userId"`
	DiscogsToken     *string    `gorm:"type:text"                json:"discogsToken"`
	DiscogsUsername  *string    `gorm:"type:text"                json:"discogsUsername"`
	SelectedFolderID *uuid.UUID `gorm:"type:uuid"                json:"selectedFolderId,omitzero"`
}

const (
	USER_CACHE_PREFIX_CONFIG = "user_oidc:" // Single cache by OIDC ID
)

// AfterSave GORM hook to clear user cache when configuration is saved
func (uc *UserConfiguration) AfterSave(tx *gorm.DB) error {
	uc.clearUserCache(tx.Statement.Context, tx)
	return nil
}

// AfterCreate GORM hook to clear user cache when configuration is created
func (uc *UserConfiguration) AfterCreate(tx *gorm.DB) error {
	uc.clearUserCache(tx.Statement.Context, tx)
	return nil
}

func (uc *UserConfiguration) clearUserCache(ctx context.Context, tx *gorm.DB) {
	log := logger.New("UserConfiguration").Function("clearUserCache")

	// Get database instance from tx
	var db database.DB
	if dbInterface, exists := tx.Get("db_instance"); exists {
		if d, ok := dbInterface.(database.DB); ok {
			db = d
		} else {
			log.Warn("db_instance is not of type database.DB")
			return
		}
	} else {
		log.Warn("db_instance not found in GORM context")
		return
	}

	// Get user to find OIDC user ID for cache clearing
	var user User
	if err := tx.WithContext(ctx).First(&user, "id = ?", uc.UserID).Error; err != nil {
		log.Warn("failed to get user for cache clearing", "userID", uc.UserID, "error", err)
		return
	}

	// Clear single-layer cache by OIDC ID
	if user.OIDCUserID != "" {
		userCacheKey := USER_CACHE_PREFIX_CONFIG + user.OIDCUserID
		if err := database.NewCacheBuilder(db.Cache.User, userCacheKey).WithContext(ctx).Delete(); err != nil {
			log.Warn("failed to clear user cache", "oidcUserID", user.OIDCUserID, "error", err)
		} else {
			log.Info("cleared user cache after configuration change", "userID", uc.UserID, "oidcUserID", user.OIDCUserID)
		}
	} else {
		log.Warn("user has no OIDC ID, cannot clear cache", "userID", uc.UserID)
	}
}
