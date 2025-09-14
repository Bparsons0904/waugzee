package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Currency string

const (
	CurrencyUSD Currency = "USD"
	CurrencyEUR Currency = "EUR"
	CurrencyGBP Currency = "GBP"
	CurrencyCAD Currency = "CAD"
	CurrencyAUD Currency = "AUD"
	CurrencyJPY Currency = "JPY"
)

type PreferredFormat string

const (
	PreferredFormatVinyl   PreferredFormat = "vinyl"
	PreferredFormatCD      PreferredFormat = "cd"
	PreferredFormatCassette PreferredFormat = "cassette"
	PreferredFormatDigital PreferredFormat = "digital"
	PreferredFormatAll     PreferredFormat = "all"
)

type NotificationSettings struct {
	EmailNotifications      bool `json:"emailNotifications"`
	SyncNotifications       bool `json:"syncNotifications"`
	MaintenanceReminders    bool `json:"maintenanceReminders"`
	StylusWearNotifications bool `json:"stylusWearNotifications"`
	WeeklyDigest            bool `json:"weeklyDigest"`
	MonthlyReport           bool `json:"monthlyReport"`
}

type DisplaySettings struct {
	Theme                string `json:"theme"`                // light, dark, auto
	DefaultView          string `json:"defaultView"`          // grid, list, table
	ItemsPerPage         int    `json:"itemsPerPage"`
	ShowAlbumArt         bool   `json:"showAlbumArt"`
	ShowRatings          bool   `json:"showRatings"`
	ShowPlayCount        bool   `json:"showPlayCount"`
	ShowLastPlayed       bool   `json:"showLastPlayed"`
	DefaultSortField     string `json:"defaultSortField"`
	DefaultSortDirection string `json:"defaultSortDirection"` // asc, desc
}

type PrivacySettings struct {
	PrivateCollection bool `json:"privateCollection"`
	ShowPrices        bool `json:"showPrices"`
	ShowPlayHistory   bool `json:"showPlayHistory"`
	AllowPublicStats  bool `json:"allowPublicStats"`
}

type UserPreferences struct {
	BaseUUIDModel
	UserID             uuid.UUID             `gorm:"type:uuid;not null;uniqueIndex:idx_user_preferences_user" json:"userId" validate:"required"`
	DefaultCurrency    *Currency             `gorm:"type:text;default:'USD'" json:"defaultCurrency,omitempty"`
	PreferredFormat    *PreferredFormat      `gorm:"type:text;default:'vinyl'" json:"preferredFormat,omitempty"`
	Notifications      *NotificationSettings `gorm:"type:jsonb" json:"notifications,omitempty"`
	Display            *DisplaySettings      `gorm:"type:jsonb" json:"display,omitempty"`
	Privacy            *PrivacySettings      `gorm:"type:jsonb" json:"privacy,omitempty"`
	Settings           map[string]interface{} `gorm:"type:jsonb" json:"settings,omitempty"`

	// Relationships
	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (up *UserPreferences) BeforeCreate(tx *gorm.DB) (err error) {
	if up.UserID == uuid.Nil {
		return gorm.ErrInvalidValue
	}

	// Set defaults if not provided
	if up.DefaultCurrency == nil {
		defaultCurrency := CurrencyUSD
		up.DefaultCurrency = &defaultCurrency
	}

	if up.PreferredFormat == nil {
		defaultFormat := PreferredFormatVinyl
		up.PreferredFormat = &defaultFormat
	}

	if up.Notifications == nil {
		up.Notifications = &NotificationSettings{
			EmailNotifications:      true,
			SyncNotifications:       true,
			MaintenanceReminders:    true,
			StylusWearNotifications: true,
			WeeklyDigest:            false,
			MonthlyReport:           false,
		}
	}

	if up.Display == nil {
		up.Display = &DisplaySettings{
			Theme:                "auto",
			DefaultView:          "grid",
			ItemsPerPage:         20,
			ShowAlbumArt:         true,
			ShowRatings:          true,
			ShowPlayCount:        true,
			ShowLastPlayed:       true,
			DefaultSortField:     "artist",
			DefaultSortDirection: "asc",
		}
	}

	if up.Privacy == nil {
		up.Privacy = &PrivacySettings{
			PrivateCollection: false,
			ShowPrices:        true,
			ShowPlayHistory:   true,
			AllowPublicStats:  false,
		}
	}

	return nil
}

func (up *UserPreferences) BeforeUpdate(tx *gorm.DB) (err error) {
	if up.UserID == uuid.Nil {
		return gorm.ErrInvalidValue
	}
	return nil
}



