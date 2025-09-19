package models

import (
	"time"
	"waugzee/internal/utils"

	"gorm.io/gorm"
)

type Master struct {
	DiscogsID   int64     `gorm:"type:bigint;primaryKey;not null"                          json:"discogsId"             validate:"required,gt=0"`
	CreatedAt   time.Time `gorm:"autoCreateTime"                                           json:"createdAt"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"                                           json:"updatedAt"`
	Title       string    `gorm:"type:text;not null;index:idx_masters_title"               json:"title"                 validate:"required"`
	MainRelease *int      `gorm:"type:int"                                                 json:"mainRelease,omitempty"`
	Year        *int      `gorm:"type:int;index:idx_masters_year"                          json:"year,omitempty"`
	ContentHash string    `gorm:"type:varchar(64);not null;index:idx_masters_content_hash" json:"contentHash"`

	// Relationships
	Releases []Release `gorm:"foreignKey:MasterID"       json:"releases,omitempty"`
	Genres   []Genre   `gorm:"many2many:master_genres;"  json:"genres,omitempty"`
	Artists  []Artist  `gorm:"many2many:master_artists;" json:"artists,omitempty"`
}

func (m *Master) BeforeCreate(tx *gorm.DB) (err error) {
	if m.DiscogsID <= 0 {
		return gorm.ErrInvalidValue
	}
	if m.Title == "" {
		return gorm.ErrInvalidValue
	}

	// Generate content hash
	hash, err := utils.GenerateEntityHash(m)
	if err != nil {
		return err
	}
	m.ContentHash = hash

	return nil
}

func (m *Master) BeforeUpdate(tx *gorm.DB) (err error) {
	if m.Title == "" {
		return gorm.ErrInvalidValue
	}

	// Regenerate content hash
	hash, err := utils.GenerateEntityHash(m)
	if err != nil {
		return err
	}
	m.ContentHash = hash

	return nil
}

// Hashable interface implementation
func (m *Master) GetHashableFields() map[string]any {
	return map[string]any{
		"Title":       m.Title,
		"MainRelease": m.MainRelease,
		"Year":        m.Year,
	}
}

func (m *Master) SetContentHash(hash string) {
	m.ContentHash = hash
}

func (m *Master) GetContentHash() string {
	return m.ContentHash
}

func (m *Master) GetDiscogsID() int64 {
	return m.DiscogsID
}
