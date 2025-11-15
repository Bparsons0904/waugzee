package models

import (
	"time"

	"github.com/google/uuid"
)

type DailyRecommendation struct {
	BaseUUIDModel
	UserID        uuid.UUID   `gorm:"type:uuid;not null;index:idx_user_date,composite:0" json:"userId"`
	User          User        `gorm:"foreignKey:UserID"                                  json:"user"`
	UserReleaseID uuid.UUID   `gorm:"type:uuid;not null;index"                           json:"userReleaseId"`
	UserRelease   UserRelease `gorm:"foreignKey:UserReleaseID"                           json:"userRelease"`
	Date          time.Time   `gorm:"type:date;not null;index:idx_user_date,composite:1" json:"date"`
	ListenedAt    *time.Time  `gorm:"type:timestamp"                                     json:"listenedAt"`
	Algorithm     string      `gorm:"type:varchar(20);not null"                          json:"algorithm"`
}
