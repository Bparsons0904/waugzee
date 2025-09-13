package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ReleaseFormat string

const (
	FormatVinyl   ReleaseFormat = "vinyl"
	FormatCD      ReleaseFormat = "cd"
	FormatCassette ReleaseFormat = "cassette"
	FormatDigital ReleaseFormat = "digital"
	FormatOther   ReleaseFormat = "other"
)

type Release struct {
	BaseUUIDModel
	Title         string        `gorm:"type:text;not null;index:idx_releases_title" json:"title" validate:"required"`
	DiscogsID     int64         `gorm:"type:bigint;not null;uniqueIndex:idx_releases_discogs_id" json:"discogsId" validate:"required"`
	LabelID       *uuid.UUID    `gorm:"type:uuid;index:idx_releases_label" json:"labelId,omitempty"`
	Year          *int          `gorm:"type:int;index:idx_releases_year" json:"year,omitempty"`
	Country       *string       `gorm:"type:text" json:"country,omitempty"`
	CatalogNumber *string       `gorm:"type:text" json:"catalogNumber,omitempty"`
	Format        ReleaseFormat `gorm:"type:text;default:'vinyl';index:idx_releases_format" json:"format"`
	ImageURL      *string       `gorm:"type:text" json:"imageUrl,omitempty"`
	TrackCount    *int          `gorm:"type:int" json:"trackCount,omitempty"`

	// Relationships
	Label           *Label             `gorm:"foreignKey:LabelID" json:"label,omitempty"`
	Artists         []Artist           `gorm:"many2many:release_artists;" json:"artists,omitempty"`
	Genres          []Genre            `gorm:"many2many:release_genres;" json:"genres,omitempty"`
	Tracks          []Track            `gorm:"foreignKey:ReleaseID" json:"tracks,omitempty"`
	UserCollections []UserCollection   `gorm:"foreignKey:ReleaseID" json:"userCollections,omitempty"`
	PlaySessions    []PlaySession      `gorm:"foreignKey:ReleaseID" json:"playSessions,omitempty"`
}

func (r *Release) BeforeCreate(tx *gorm.DB) (err error) {
	if r.Title == "" {
		return gorm.ErrInvalidValue
	}
	if r.DiscogsID == 0 {
		return gorm.ErrInvalidValue
	}
	if r.Format == "" {
		r.Format = FormatVinyl
	}
	return nil
}

func (r *Release) BeforeUpdate(tx *gorm.DB) (err error) {
	if r.Title == "" {
		return gorm.ErrInvalidValue
	}
	if r.DiscogsID == 0 {
		return gorm.ErrInvalidValue
	}
	return nil
}



