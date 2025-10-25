package models

import (
	"time"

	"github.com/google/uuid"
)

type PlayHistory struct {
	BaseUUIDModel
	UserID         uuid.UUID    `gorm:"type:uuid;not null;index"   json:"userId"`
	User           User         `gorm:"foreignKey:UserID"          json:"user"`
	UserReleaseID  uuid.UUID    `gorm:"type:uuid;not null;index"   json:"userReleaseId"`
	UserRelease    UserRelease  `gorm:"foreignKey:UserReleaseID"   json:"userRelease"`
	UserStylusID   *uuid.UUID   `gorm:"type:uuid"                  json:"userStylusId"`
	UserStylus     *UserStylus  `gorm:"foreignKey:UserStylusID"    json:"userStylus"`
	PlayedAt       time.Time    `gorm:"not null"                   json:"playedAt"`
	Notes          string       `gorm:"type:text"                  json:"notes"`
}
