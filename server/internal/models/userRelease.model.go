package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRelease struct {
	UserID     uuid.UUID      `gorm:"type:uuid;primaryKey,idx_user_folder" json:"userId"`
	User       User           `gorm:"foreignKey:UserID"                    json:"user"`
	ReleaseID  uuid.UUID      `gorm:"type:uuid;primaryKey"                 json:"releaseId"`
	Release    Release        `gorm:"foreignKey:ReleaseID"                 json:"release"`
	InstanceID int            `gorm:"type:int"                             json:"instanceId"`
	FolderID   int            `gorm:"type:int;idx_user_folder"             json:"folderId"`
	Rating     int            `gorm:"type:int"                             json:"rating"`
	Notes      string         `gorm:"type:text"                            json:"notes"`
	Active     bool           `gorm:"type:bool;default:true"               json:"active"`
	AddedAt    time.Time      `gorm:"autoCreateTime"                       json:"addedAt"`
	UpdatedAt  time.Time      `gorm:"autoUpdateTime"                       json:"updatedAt"`
	DeletedAt  gorm.DeletedAt `gorm:"index"                                json:"deletedAt"`
}

