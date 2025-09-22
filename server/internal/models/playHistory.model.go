package models

import (
	"time"

	"github.com/google/uuid"
)

type PlayHistory struct {
	BaseUUIDModel
	UserID    uuid.UUID  `gorm:"type:uuid;not null;index" json:"userId"`
	User      User       `gorm:"foreignKey:UserID"        json:"user"`
	ReleaseID uuid.UUID  `gorm:"type:uuid;not null;index" json:"releaseId"`
	Release   Release    `gorm:"foreignKey:ReleaseID"     json:"release"`
	StylusID  *uuid.UUID `gorm:"type:uuid"                json:"stylusId"`
	Stylus    *Stylus    `gorm:"foreignKey:StylusID"      json:"stylus"`
	PlayedAt  time.Time  `gorm:"not null"                 json:"playedAt"`
	Notes     string     `gorm:"type:text"                json:"notes"`
}
