package models

import (
	"strings"
	"time"

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
	OIDCUserID      string     `gorm:"column:oidc_user_id;type:text;uniqueIndex" json:"-"`
	OIDCProvider    *string    `gorm:"column:oidc_provider;type:text"            json:"-"`
	OIDCProjectID   *string    `gorm:"column:oidc_project_id;type:text"          json:"-"`
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
	OIDCUserID      string  `json:"oidcUserId"`
	Email           *string `json:"email,omitempty"`
	Name            *string `json:"name,omitempty"`
	FirstName       string  `json:"firstName"`
	LastName        string  `json:"lastName"`
	OIDCProvider    string  `json:"oidcProvider"`
	OIDCProjectID   *string `json:"oidcProjectId,omitempty"`
	ProfileVerified bool    `json:"profileVerified"`
}

// IsOIDCUser returns true if the user was created via OIDC
func (u *User) IsOIDCUser() bool {
	return u.OIDCUserID != ""
}

// UpdateFromOIDC updates user information from OIDC claims
func (u *User) UpdateFromOIDC(oidcEmail, oidcName *string, provider string, projectID *string) {
	now := time.Now()
	u.LastLoginAt = &now

	if oidcEmail != nil && *oidcEmail != "" {
		u.Email = oidcEmail
	}

	if oidcName != nil && *oidcName != "" {
		u.DisplayName = *oidcName
	}

	if provider != "" {
		providerPtr := &provider
		u.OIDCProvider = providerPtr
	}

	if projectID != nil {
		u.OIDCProjectID = projectID
	}

	// Mark profile as verified if we have email from OIDC provider
	if oidcEmail != nil && *oidcEmail != "" {
		u.ProfileVerified = true
	}
}

// UpdateFromOIDCDetailed updates user information from detailed OIDC claims including name components
func (u *User) UpdateFromOIDCDetailed(
	oidcUserID string,
	oidcEmail, oidcName *string,
	firstName, lastName, provider string,
	projectID *string,
	emailVerified bool,
) {
	now := time.Now()
	u.LastLoginAt = &now

	// Preserve OIDC User ID - this is critical for linking sessions
	if oidcUserID != "" {
		u.OIDCUserID = oidcUserID
	}

	if oidcEmail != nil && *oidcEmail != "" {
		u.Email = oidcEmail
	}

	// Update first/last names if provided
	if firstName != "" {
		u.FirstName = firstName
	}
	if lastName != "" {
		u.LastName = lastName
	}

	// Rebuild FullName from components
	if firstName != "" || lastName != "" {
		u.FullName = strings.TrimSpace(firstName + " " + lastName)
	}

	// Update display name with preference: oidcName > FullName > existing
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
