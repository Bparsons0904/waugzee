package models

import (
	"time"

	"gorm.io/gorm"
	"waugzee/internal/utils"
)

type Artist struct {
	DiscogsID   int64     `gorm:"type:bigint;primaryKey;not null" json:"discogsId" validate:"required,gt=0"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
	Name        string    `gorm:"type:text;not null;index:idx_artists_name" json:"name" validate:"required"`
	IsActive    bool      `gorm:"type:bool;default:true;not null" json:"isActive"`
	ContentHash string    `gorm:"type:varchar(64);not null;index:idx_artists_content_hash" json:"contentHash"`

	// Relationships
	Releases []Release `gorm:"many2many:release_artists;" json:"releases,omitempty"`
	Images   []Image   `gorm:"polymorphic:Imageable;" json:"images,omitempty"`
}

func (a *Artist) BeforeCreate(tx *gorm.DB) (err error) {
	if a.DiscogsID <= 0 {
		return gorm.ErrInvalidValue
	}
	if a.Name == "" {
		return gorm.ErrInvalidValue
	}

	// Generate content hash
	hash, err := utils.GenerateEntityHash(a)
	if err != nil {
		return err
	}
	a.ContentHash = hash

	return nil
}

func (a *Artist) BeforeUpdate(tx *gorm.DB) (err error) {
	if a.Name == "" {
		return gorm.ErrInvalidValue
	}

	// Regenerate content hash
	hash, err := utils.GenerateEntityHash(a)
	if err != nil {
		return err
	}
	a.ContentHash = hash

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

// Hashable interface implementation
func (a *Artist) GetHashableFields() map[string]interface{} {
	return map[string]interface{}{
		"Name":     a.Name,
		"IsActive": a.IsActive,
	}
}

func (a *Artist) SetContentHash(hash string) {
	a.ContentHash = hash
}

func (a *Artist) GetContentHash() string {
	return a.ContentHash
}

