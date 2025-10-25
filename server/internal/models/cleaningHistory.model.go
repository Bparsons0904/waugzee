package models

import (
	"time"

	"github.com/google/uuid"
)

type CleaningHistory struct {
	BaseUUIDModel
	UserID        uuid.UUID   `gorm:"type:uuid;not null;index"  json:"userId"`
	User          User        `gorm:"foreignKey:UserID"         json:"user"`
	UserReleaseID uuid.UUID   `gorm:"type:uuid;not null;index"  json:"userReleaseId"`
	UserRelease   UserRelease `gorm:"foreignKey:UserReleaseID"  json:"userRelease"`
	CleanedAt     time.Time   `gorm:"not null"                  json:"cleanedAt"`
	Notes         string      `gorm:"type:text"                 json:"notes"`
	IsDeepClean   bool        `gorm:"not null;default:false"    json:"isDeepClean"`
}
