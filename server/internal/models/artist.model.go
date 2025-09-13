package models

import (
	"gorm.io/gorm"
)

type Artist struct {
	BaseUUIDModel
	Name        string  `gorm:"type:text;not null;index:idx_artists_name" json:"name" validate:"required"`
	DiscogsID   *int64  `gorm:"type:bigint;uniqueIndex:idx_artists_discogs_id" json:"discogsId,omitempty"`
	Biography   *string `gorm:"type:text" json:"biography,omitempty"`
	ImageURL    *string `gorm:"type:text" json:"imageUrl,omitempty"`
	IsActive    bool    `gorm:"type:bool;default:true;not null" json:"isActive"`

	// Relationships
	Releases     []Release     `gorm:"many2many:release_artists;" json:"releases,omitempty"`
	Tracks       []Track       `gorm:"foreignKey:ArtistID" json:"tracks,omitempty"`
}

func (a *Artist) BeforeCreate(tx *gorm.DB) (err error) {
	if a.Name == "" {
		return gorm.ErrInvalidValue
	}
	return nil
}

func (a *Artist) BeforeUpdate(tx *gorm.DB) (err error) {
	if a.Name == "" {
		return gorm.ErrInvalidValue
	}
	return nil
}

