package models

import (
	"github.com/google/uuid"
)

type UserConfiguration struct {
	BaseUUIDModel
	UserID           uuid.UUID `gorm:"type:uuid;not null;index" json:"userId"`
	DiscogsToken     *string   `gorm:"type:text"                json:"discogsToken"`
	DiscogsUsername  *string   `gorm:"type:text"                json:"discogsUsername"`
	SelectedFolderID *uuid.UUID `gorm:"type:uuid"               json:"selectedFolderId"`
}
