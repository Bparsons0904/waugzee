package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRelease struct {
	BaseUUIDModel
	UserID     uuid.UUID      `gorm:"type:uuid;not null;uniqueIndex:idx_user_instance" json:"userId"`
	User       User           `gorm:"foreignKey:UserID"                                 json:"user"`
	ReleaseID  int64          `gorm:"type:bigint;not null"                             json:"releaseId"`
	Release    Release        `gorm:"foreignKey:ReleaseID"                             json:"release"`
	InstanceID int            `gorm:"type:int;not null;uniqueIndex:idx_user_instance"  json:"instanceId"`
	FolderID   int            `gorm:"type:int;not null;index:idx_user_folder"          json:"folderId"`
	Rating     int            `gorm:"type:int"                                         json:"rating"`
	Notes      string         `gorm:"type:text"                                        json:"notes"`
	Active     bool           `gorm:"type:bool;default:true"                           json:"active"`
	DeletedAt  gorm.DeletedAt `gorm:"index"                                            json:"-"`
}
