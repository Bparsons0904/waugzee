package models

import (
	"time"

	"gorm.io/datatypes"
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
	DiscogsID   int64         `gorm:"type:bigint;primaryKey;not null" json:"discogsId" validate:"required,gt=0"`
	CreatedAt   time.Time     `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt   time.Time     `gorm:"autoUpdateTime" json:"updatedAt"`
	Title       string        `gorm:"type:text;not null;index:idx_releases_title" json:"title" validate:"required"`
	LabelID     *int64        `gorm:"type:bigint;index:idx_releases_label" json:"labelId,omitempty"`
	MasterID    *int64        `gorm:"type:bigint;index:idx_releases_master;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"masterId,omitempty"`
	Year        *int          `gorm:"type:int;index:idx_releases_year" json:"year,omitempty"`
	Country     *string       `gorm:"type:text" json:"country,omitempty"`
	Format      ReleaseFormat `gorm:"type:text;default:'vinyl';index:idx_releases_format" json:"format"`
	ImageURL    *string       `gorm:"type:text" json:"imageUrl,omitempty"`
	TrackCount  *int          `gorm:"type:int" json:"trackCount,omitempty"`
	ContentHash string        `gorm:"type:varchar(64);not null;index:idx_releases_content_hash" json:"contentHash"`

	// JSONB columns for embedded data
	TracksJSON  datatypes.JSON `gorm:"type:jsonb" json:"tracks,omitempty"`
	ArtistsJSON datatypes.JSON `gorm:"type:jsonb" json:"artists,omitempty"`
	GenresJSON  datatypes.JSON `gorm:"type:jsonb" json:"genres,omitempty"`

	// Relationships
	Label           *Label             `gorm:"foreignKey:LabelID" json:"label,omitempty"`
	Master          *Master            `gorm:"foreignKey:MasterID" json:"master,omitempty"`
	// Note: Track/Artist/Genre data now stored as JSONB for simple display
	// Searchable relationships maintained at Master level
	UserCollections []UserCollection   `gorm:"foreignKey:ReleaseID" json:"userCollections,omitempty"`
	PlaySessions    []PlaySession      `gorm:"foreignKey:ReleaseID" json:"playSessions,omitempty"`
}

func (r *Release) BeforeCreate(tx *gorm.DB) (err error) {
	if r.DiscogsID <= 0 {
		return gorm.ErrInvalidValue
	}
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
		"Title":       r.Title,
		"LabelID":     r.LabelID,
		"MasterID":    r.MasterID,
		"Year":        r.Year,
		"Country":     r.Country,
		"Format":      r.Format,
		"ImageURL":    r.ImageURL,
		"TrackCount":  r.TrackCount,
		"TracksJSON":  r.TracksJSON,
		"ArtistsJSON": r.ArtistsJSON,
		"GenresJSON":  r.GenresJSON,
	}
}

func (r *Release) SetContentHash(hash string) {
	r.ContentHash = hash
}

func (r *Release) GetContentHash() string {
	return r.ContentHash
}

func (r *Release) GetDiscogsID() int64 {
	return r.DiscogsID
}



