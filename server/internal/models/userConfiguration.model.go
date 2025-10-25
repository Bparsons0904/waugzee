package models

import (
	"github.com/google/uuid"
)

type UserConfiguration struct {
	BaseUUIDModel
	UserID           uuid.UUID `gorm:"type:uuid;not null;uniqueIndex;constraint:OnDelete:CASCADE;" json:"userId"`
	User             *User     `gorm:"foreignKey:UserID;references:ID"                             json:"-"`
	DiscogsToken     *string   `gorm:"type:text"                                                   json:"discogsToken"`
	DiscogsUsername  *string   `gorm:"type:text"                                                   json:"discogsUsername"`
	SelectedFolderID *int      `gorm:"type:bigint;default:0"                                       json:"selectedFolderId"`
}
