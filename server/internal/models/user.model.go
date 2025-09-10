package models

import (
	"gorm.io/gorm"
)

type User struct {
	BaseUUIDModel
	// Basic user info
	FirstName   string  `gorm:"type:text"               json:"firstName"`
	LastName    string  `gorm:"type:text"               json:"lastName"`
	FullName    string  `gorm:"type:text"               json:"fullName"`
	DisplayName string  `gorm:"type:text"               json:"displayName"`
	Email       *string `gorm:"type:text;uniqueIndex"   json:"email"`
	IsAdmin     bool    `gorm:"type:bool;default:false" json:"isAdmin"`
	IsActive    bool    `gorm:"type:bool;default:true"  json:"isActive"`

	// OIDC integration fields
	OIDCUserID   string  `gorm:"type:text;uniqueIndex" json:"-"`
	OIDCProvider *string `gorm:"type:text"             json:"oidcProvider,omitempty"`
	LastLoginAt  *int64  `gorm:"type:bigint"           json:"lastLoginAt,omitempty"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	fullName := u.FirstName + " " + u.LastName
	u.FullName = fullName
	if u.DisplayName == "" {
		u.DisplayName = fullName
	}
	return nil
}

// TODO: Does all this below here stay here?
type LoginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// OIDCUserCreateRequest represents data for creating a user from OIDC claims
type OIDCUserCreateRequest struct {
	OIDCUserID   string  `json:"oidcUserId"`
	Email        *string `json:"email,omitempty"`
	Name         *string `json:"name,omitempty"`
	FirstName    string  `json:"firstName"`
	LastName     string  `json:"lastName"`
	OIDCProvider string  `json:"oidcProvider"`
	TenantID     *string `json:"tenantId,omitempty"`
}

// UserProfile represents public user profile information
type UserProfile struct {
	ID          string  `json:"id"`
	FirstName   string  `json:"firstName"`
	LastName    string  `json:"lastName"`
	DisplayName string  `json:"displayName"`
	Email       *string `json:"email,omitempty"`
	IsActive    bool    `json:"isActive"`
	LastLoginAt *int64  `json:"lastLoginAt,omitempty"`
}

// ToProfile converts a User to a UserProfile (public information only)
func (u *User) ToProfile() UserProfile {
	return UserProfile{
		ID:          u.ID.String(),
		FirstName:   u.FirstName,
		LastName:    u.LastName,
		DisplayName: u.DisplayName,
		Email:       u.Email,
		IsActive:    u.IsActive,
		LastLoginAt: u.LastLoginAt,
	}
}

// IsOIDCUser returns true if the user was created via OIDC
func (u *User) IsOIDCUser() bool {
	return u.OIDCUserID != ""
}

// UpdateFromOIDC updates user information from OIDC claims
func (u *User) UpdateFromOIDC(oidcEmail, oidcName *string, provider string) {
	if oidcEmail != nil {
		u.Email = oidcEmail
	}

	if oidcName != nil {
		u.DisplayName = *oidcName
	}

	if provider != "" {
		providerPtr := &provider
		u.OIDCProvider = providerPtr
	}
}
