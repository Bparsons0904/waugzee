package models

import (
	"gorm.io/gorm"
)

type Master struct {
	BaseUUIDModel
	Title        string  `gorm:"type:text;not null;index:idx_masters_title" json:"title" validate:"required"`
	DiscogsID    *int64  `gorm:"type:bigint;uniqueIndex:idx_masters_discogs_id" json:"discogsId,omitempty"`
	MainRelease  *int64  `gorm:"type:bigint" json:"mainRelease,omitempty"`
	Year         *int    `gorm:"type:int;index:idx_masters_year" json:"year,omitempty"`
	Notes        *string `gorm:"type:text" json:"notes,omitempty"`
	DataQuality  *string `gorm:"type:text" json:"dataQuality,omitempty"`

	// Relationships
	Releases []Release `gorm:"foreignKey:MasterID" json:"releases,omitempty"`
	Genres   []Genre   `gorm:"many2many:master_genres;" json:"genres,omitempty"`
	Artists  []Artist  `gorm:"many2many:master_artists;" json:"artists,omitempty"`
}

func (m *Master) BeforeCreate(tx *gorm.DB) (err error) {
	if m.Title == "" {
		return gorm.ErrInvalidValue
	}
	return nil
}

func (m *Master) BeforeUpdate(tx *gorm.DB) (err error) {
	if m.Title == "" {
		return gorm.ErrInvalidValue
	}
	return nil
}