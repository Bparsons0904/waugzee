package models

import (
	"time"

	"github.com/google/uuid"
)

type PlayHistory struct {
	BaseUUIDModel
	UserID       uuid.UUID   `gorm:"type:uuid;not null;index"      json:"userId"`
	User         User        `gorm:"foreignKey:UserID"             json:"user"`
	ReleaseID    int64       `gorm:"type:bigint;not null;index"    json:"releaseId"`
	Release      Release     `gorm:"foreignKey:ReleaseID"          json:"release"`
	UserStylusID *uuid.UUID  `gorm:"type:uuid"                     json:"userStylusId"`
	UserStylus   *UserStylus `gorm:"foreignKey:UserStylusID"       json:"userStylus"`
	PlayedAt     time.Time   `gorm:"not null"                      json:"playedAt"`
	Notes        string      `gorm:"type:text"                     json:"notes"`
}
