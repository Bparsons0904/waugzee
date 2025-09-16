package models

import (
	"gorm.io/gorm"
	"waugzee/internal/utils"
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
	BaseModel
	Title       string        `gorm:"type:text;not null;index:idx_releases_title" json:"title" validate:"required"`
	LabelID     *int          `gorm:"type:int;index:idx_releases_label" json:"labelId,omitempty"`
	MasterID    *int          `gorm:"type:int;index:idx_releases_master" json:"masterId,omitempty"`
	Year        *int          `gorm:"type:int;index:idx_releases_year" json:"year,omitempty"`
	Country     *string       `gorm:"type:text" json:"country,omitempty"`
	Format      ReleaseFormat `gorm:"type:text;default:'vinyl';index:idx_releases_format" json:"format"`
	ImageURL    *string       `gorm:"type:text" json:"imageUrl,omitempty"`
	TrackCount  *int          `gorm:"type:int" json:"trackCount,omitempty"`
	ContentHash string        `gorm:"type:varchar(64);not null;index:idx_releases_content_hash" json:"contentHash"`

	// Relationships
	Label           *Label             `gorm:"foreignKey:LabelID" json:"label,omitempty"`
	Master          *Master            `gorm:"foreignKey:MasterID" json:"master,omitempty"`
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
	if r.Format == "" {
		r.Format = FormatVinyl
	}

	// Generate content hash
	hash, err := utils.GenerateEntityHash(r)
	if err != nil {
		return err
	}
	r.ContentHash = hash

	return nil
}

func (r *Release) BeforeUpdate(tx *gorm.DB) (err error) {
	if r.Title == "" {
		return gorm.ErrInvalidValue
	}

	// Regenerate content hash
	hash, err := utils.GenerateEntityHash(r)
	if err != nil {
		return err
	}
	r.ContentHash = hash

	return nil
}

// Hashable interface implementation
func (r *Release) GetHashableFields() map[string]interface{} {
	return map[string]interface{}{
		"Title":      r.Title,
		"LabelID":    r.LabelID,
		"MasterID":   r.MasterID,
		"Year":       r.Year,
		"Country":    r.Country,
		"Format":     r.Format,
		"ImageURL":   r.ImageURL,
		"TrackCount": r.TrackCount,
	}
}

func (r *Release) SetContentHash(hash string) {
	r.ContentHash = hash
}

func (r *Release) GetContentHash() string {
	return r.ContentHash
}



