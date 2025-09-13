package models

import (
	"strings"
	"time"
	"waugzee/internal/database"
	"waugzee/internal/logger"

	"gorm.io/gorm"
)

type User struct {
	BaseUUIDModel
	FirstName       string     `gorm:"type:text"                                 json:"firstName"`
	LastName        string     `gorm:"type:text"                                 json:"lastName"`
	FullName        string     `gorm:"type:text"                                 json:"fullName"`
	DisplayName     string     `gorm:"type:text"                                 json:"displayName"`
	Email           *string    `gorm:"type:text;uniqueIndex"                     json:"email"`
	IsAdmin         bool       `gorm:"type:bool;default:false"                   json:"isAdmin"`
	IsActive        bool       `gorm:"type:bool;default:true"                    json:"isActive"`
	LastLoginAt     *time.Time `gorm:"type:timestamp"                            json:"lastLoginAt,omitempty"`
	ProfileVerified bool       `gorm:"type:bool;default:false"                   json:"profileVerified"`
	DiscogsToken    *string    `gorm:"type:text"                                 json:"discogsToken,omitempty"`
	OIDCUserID      string     `gorm:"column:oidc_user_id;type:text;uniqueIndex" json:"-"`
	OIDCProvider    *string    `gorm:"column:oidc_provider;type:text"            json:"-"`
	OIDCProjectID   *string    `gorm:"column:oidc_project_id;type:text"          json:"-"`
}

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	if u.FirstName != "" || u.LastName != "" {
		if u.FullName == "" {
			u.FullName = strings.TrimSpace(u.FirstName + " " + u.LastName)
		}
		if u.DisplayName == "" {
			u.DisplayName = u.FullName
		}
	}

	return err
}

func (u *User) IsOIDCUser() bool {
	return u.OIDCUserID != ""
}

func (u *User) AfterUpdate(tx *gorm.DB) error {
	log := logger.New("User").Function("AfterUpdate")
	cacheInterface, exists := tx.Get("waugzee:cache")
	if !exists {
		return nil
	}

	cache, ok := cacheInterface.(database.CacheClient)
	if !ok {
		return nil
	}

	err := database.NewCacheBuilder(cache, u.ID).Delete()
	log.Warn("failed to remove user from cache", "userID", u.ID, "error", err)

	if u.OIDCUserID != "" {
		oidcCacheKey := "oidc:" + u.OIDCUserID
		err := database.NewCacheBuilder(cache, oidcCacheKey).Delete()
		log.Warn(
			"failed to remove OIDC mapping from cache",
			"oidcUserID",
			u.OIDCUserID,
			"error",
			err,
		)
	}

	return nil
}

func (u *User) UpdateFromOIDC(
	oidcUserID string,
	oidcEmail, oidcName *string,
	firstName, lastName, provider string,
	projectID *string,
	emailVerified bool,
) {
	now := time.Now()
	u.LastLoginAt = &now

	if oidcUserID != "" {
		u.OIDCUserID = oidcUserID
	}

	if oidcEmail != nil && *oidcEmail != "" {
		u.Email = oidcEmail
	}

	if firstName != "" {
		u.FirstName = firstName
	}
	if lastName != "" {
		u.LastName = lastName
	}

	if firstName != "" || lastName != "" {
		u.FullName = strings.TrimSpace(firstName + " " + lastName)
	}

	if oidcName != nil && *oidcName != "" {
		u.DisplayName = *oidcName
	} else if u.FullName != "" {
		u.DisplayName = u.FullName
	}

	if provider != "" {
		providerPtr := &provider
		u.OIDCProvider = providerPtr
	}

	if projectID != nil {
		u.OIDCProjectID = projectID
	}

	// Mark profile as verified based on email verification status
	if emailVerified && oidcEmail != nil && *oidcEmail != "" {
		u.ProfileVerified = true
	}
}
