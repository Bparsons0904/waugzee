package models

import (
	"strings"
	"time"

	"gorm.io/gorm"
)

type User struct {
	BaseUUIDModel
	FirstName       string             `gorm:"type:text"                                 json:"firstName"`
	LastName        string             `gorm:"type:text"                                 json:"lastName"`
	FullName        string             `gorm:"type:text"                                 json:"fullName"`
	DisplayName     string             `gorm:"type:text"                                 json:"displayName"`
	Email           *string            `gorm:"type:text;uniqueIndex"                     json:"email"`
	IsAdmin         bool               `gorm:"type:bool;default:false"                   json:"isAdmin"`
	IsActive        bool               `gorm:"type:bool;default:true"                    json:"isActive"`
	LastLoginAt     *time.Time         `gorm:"type:timestamp"                            json:"lastLoginAt,omitempty"`
	ProfileVerified bool               `gorm:"type:bool;default:false"                   json:"profileVerified"`
	OIDCUserID      string             `gorm:"column:oidc_user_id;type:text;uniqueIndex" json:"-"`
	OIDCProvider    *string            `gorm:"column:oidc_provider;type:text"            json:"-"`
	OIDCProjectID   *string            `gorm:"column:oidc_project_id;type:text"          json:"-"`
	Configuration   *UserConfiguration `gorm:"foreignKey:UserID"                         json:"configuration,omitempty"`
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

type UserWithFoldersResponse struct {
	User    *User     `json:"user"`
	Folders []*Folder `json:"folders"`
}
