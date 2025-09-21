package models

import (
	"time"

	"github.com/google/uuid"
)

type PlayHistory struct {
	ID        uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid7()" json:"id"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null;index"                         json:"userId"`
	User      User       `gorm:"foreignKey:UserID"                                json:"user"`
	ReleaseID uuid.UUID  `gorm:"type:uuid;not null;index"                         json:"releaseId"`
	Release   Release    `gorm:"foreignKey:ReleaseID"                             json:"release"`
	StylusID  *uuid.UUID `gorm:"type:uuid"                                        json:"stylusId"`
	Stylus    *Stylus    `gorm:"foreignKey:StylusID"                              json:"stylus"`
	PlayedAt  time.Time  `gorm:"not null"                                         json:"playedAt"`
	Notes     string     `gorm:"type:text"                                        json:"notes"`
	CreatedAt time.Time  `gorm:"autoCreateTime"                                   json:"createdAt"`
	UpdatedAt time.Time  `gorm:"autoUpdateTime"                                   json:"updatedAt"`
}
