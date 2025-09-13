package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Track struct {
	BaseUUIDModel
	ReleaseID     uuid.UUID `gorm:"type:uuid;not null;index:idx_tracks_release" json:"releaseId" validate:"required"`
	Position      string    `gorm:"type:text;not null" json:"position" validate:"required"`
	Title         string    `gorm:"type:text;not null" json:"title" validate:"required"`
	Duration      *int      `gorm:"type:int" json:"duration,omitempty"` // Duration in seconds
	ArtistCredits *string   `gorm:"type:text" json:"artistCredits,omitempty"`
	ArtistID      *uuid.UUID `gorm:"type:uuid;index:idx_tracks_artist" json:"artistId,omitempty"`

	// Relationships
	Release *Release `gorm:"foreignKey:ReleaseID" json:"release,omitempty"`
	Artist  *Artist  `gorm:"foreignKey:ArtistID" json:"artist,omitempty"`
}

func (t *Track) BeforeCreate(tx *gorm.DB) (err error) {
	if t.ReleaseID == uuid.Nil {
		return gorm.ErrInvalidValue
	}
	if t.Position == "" {
		return gorm.ErrInvalidValue
	}
	if t.Title == "" {
		return gorm.ErrInvalidValue
	}
	return nil
}

func (t *Track) BeforeUpdate(tx *gorm.DB) (err error) {
	if t.ReleaseID == uuid.Nil {
		return gorm.ErrInvalidValue
	}
	if t.Position == "" {
		return gorm.ErrInvalidValue
	}
	if t.Title == "" {
		return gorm.ErrInvalidValue
	}
	return nil
}



