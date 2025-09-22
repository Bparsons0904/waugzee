package models

import (
	"github.com/google/uuid"
)

type Folder struct {
	BaseUUIDModel
	DiscogID    *int       `gorm:"type:bigint;index;uniqueIndex:idx_folder_discog_user"        json:"discogId"`
	UserID      uuid.UUID  `gorm:"type:uuid;not null;index;uniqueIndex:idx_folder_discog_user" json:"userId"`
	User        User       `gorm:"foreignKey:UserID"        json:"user"`
	Name        string     `gorm:"not null"                 json:"name"`
	Count       int        `gorm:"not null"                 json:"count"`
	ResourceURL string     `gorm:"varchar(255)"             json:"resourceUrl"`
	Releases    []*Release `gorm:"many2many:user_releases;" json:"releases"`
}
