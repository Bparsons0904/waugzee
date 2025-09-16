package models

import (
	"gorm.io/gorm"
)

type Artist struct {
	BaseUUIDModel
	Name        string  `gorm:"type:text;not null;index:idx_artists_name" json:"name" validate:"required"`
	DiscogsID   *int64  `gorm:"type:bigint;uniqueIndex:idx_artists_discogs_id" json:"discogsId,omitempty"`
	Biography   *string `gorm:"type:text" json:"biography,omitempty"`
	IsActive    bool    `gorm:"type:bool;default:true;not null" json:"isActive"`

	// Relationships
	Releases     []Release     `gorm:"many2many:release_artists;" json:"releases,omitempty"`
	Tracks       []Track       `gorm:"foreignKey:ArtistID" json:"tracks,omitempty"`
	Images       []Image       `gorm:"polymorphic:Imageable;" json:"images,omitempty"`
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

// Helper methods for working with images
func (a *Artist) GetPrimaryImage() *Image {
	for _, img := range a.Images {
		if img.IsPrimary() {
			return &img
		}
	}
	return nil
}

func (a *Artist) GetThumbnailImage() *Image {
	for _, img := range a.Images {
		if img.IsThumbnail() {
			return &img
		}
	}
	// Fallback to primary image if no thumbnail exists
	return a.GetPrimaryImage()
}

func (a *Artist) GetImageByType(imageType string) *Image {
	for _, img := range a.Images {
		if img.ImageType == imageType {
			return &img
		}
	}
	return nil
}

