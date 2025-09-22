package models

import "github.com/google/uuid"

type UserConfiguration struct {
	BaseUUIDModel
	UserID           uuid.UUID  `gorm:"type:uuid;not null;index" json:"userId"`
	DiscogsToken     *string    `gorm:"type:text"                json:"-"`
	DiscogsUsername  *string    `gorm:"type:text"                json:"-"`
	SelectedFolderID *uuid.UUID `gorm:"type:uuid"                json:"selectedFolderId,omitzero"`
}
