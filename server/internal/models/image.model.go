package models

import (
	"gorm.io/gorm"
)

type Image struct {
	BaseUUIDModel
	URL           string  `gorm:"type:text;not null" json:"url" validate:"required,url"`
	AltText       *string `gorm:"type:text" json:"altText,omitempty"`
	Width         *int    `gorm:"type:int" json:"width,omitempty"`
	Height        *int    `gorm:"type:int" json:"height,omitempty"`
	FileSize      *int64  `gorm:"type:bigint" json:"fileSize,omitempty"`
	MimeType      *string `gorm:"type:varchar(100)" json:"mimeType,omitempty"`

	// Polymorphic fields
	ImageableID   string `gorm:"type:uuid;not null;index:idx_images_imageable" json:"imageableId"`
	ImageableType string `gorm:"type:varchar(50);not null;index:idx_images_imageable" json:"imageableType"`

	// Image categorization
	ImageType     string  `gorm:"type:varchar(50);not null;default:'primary'" json:"imageType"` // primary, thumbnail, gallery, etc.
	SortOrder     *int    `gorm:"type:int;default:0" json:"sortOrder,omitempty"`

	// Discogs specific fields
	DiscogsID     *int64  `gorm:"type:bigint" json:"discogsId,omitempty"`
	DiscogsType   *string `gorm:"type:varchar(50)" json:"discogsType,omitempty"` // primary, secondary, etc.
	DiscogsURI    *string `gorm:"type:text" json:"discogsUri,omitempty"`
	DiscogsURI150 *string `gorm:"type:text" json:"discogsUri150,omitempty"`
}

func (i *Image) BeforeCreate(tx *gorm.DB) (err error) {
	if i.URL == "" || i.ImageableID == "" || i.ImageableType == "" {
		return gorm.ErrInvalidValue
	}
	return nil
}

func (i *Image) BeforeUpdate(tx *gorm.DB) (err error) {
	if i.URL == "" || i.ImageableID == "" || i.ImageableType == "" {
		return gorm.ErrInvalidValue
	}
	return nil
}

// Helper methods for common image types
func (i *Image) IsPrimary() bool {
	return i.ImageType == "primary"
}

func (i *Image) IsThumbnail() bool {
	return i.ImageType == "thumbnail"
}

// ImageConstants for ImageableType values
const (
	ImageableTypeArtist  = "artist"
	ImageableTypeRelease = "release"
	ImageableTypeLabel   = "label"
	ImageableTypeMaster  = "master"
	ImageableTypeUser    = "user"
)

// ImageTypeConstants for ImageType values
const (
	ImageTypePrimary   = "primary"
	ImageTypeThumbnail = "thumbnail"
	ImageTypeGallery   = "gallery"
	ImageTypeSecondary = "secondary"
)