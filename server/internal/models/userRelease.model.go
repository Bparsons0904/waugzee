package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type UserRelease struct {
	BaseUUIDModel
	UserID     uuid.UUID      `gorm:"type:uuid;not null;uniqueIndex:idx_user_instance" json:"userId"`
	User       User           `gorm:"foreignKey:UserID"                                json:"user"`
	ReleaseID  int64          `gorm:"type:bigint;not null"                             json:"releaseId"`
	Release    Release        `gorm:"foreignKey:ReleaseID"                             json:"release"`
	InstanceID int            `gorm:"type:int;not null;uniqueIndex:idx_user_instance"  json:"instanceId"`
	FolderID   int            `gorm:"type:int;not null;index:idx_user_folder"          json:"folderId"`
	Rating     int            `gorm:"type:int"                                         json:"rating"`
	Notes      datatypes.JSON `gorm:"type:jsonb"                                       json:"notes"`
	DateAdded  time.Time      `gorm:"type:timestamptz;not null"                        json:"dateAdded"`
	Active     bool           `gorm:"type:bool;default:true"                           json:"active"`
	DeletedAt  gorm.DeletedAt `gorm:"index"                                            json:"-"`

	PlayHistory     []PlayHistory     `gorm:"foreignKey:UserReleaseID" json:"playHistory"`
	CleaningHistory []CleaningHistory `gorm:"foreignKey:UserReleaseID" json:"cleaningHistory"`
}
