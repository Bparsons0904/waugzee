package models

import (
	"time"

	"github.com/google/uuid"
)

type Folder struct {
	ID          *int       `gorm:"type:bigint;primaryKey"       json:"id"`
	UserID      uuid.UUID  `gorm:"type:uuid;primaryKey"         json:"userId"`
	CreatedAt   time.Time  `gorm:"autoCreateTime"               json:"createdAt"`
	UpdatedAt   time.Time  `gorm:"autoUpdateTime"               json:"updatedAt"`
	User        User       `gorm:"foreignKey:UserID"            json:"user"`
	Name        string     `gorm:"not null"                     json:"name"`
	Count       int        `gorm:"not null"                     json:"count"`
	ResourceURL string     `gorm:"varchar(255)"                 json:"resourceUrl"`
	Releases    []*Release `gorm:"many2many:user_releases;"     json:"releases"`
}
