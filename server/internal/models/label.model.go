package models

import (
	"gorm.io/gorm"
)

type Label struct {
	BaseUUIDModel
	Name        string  `gorm:"type:text;not null;index:idx_labels_name" json:"name" validate:"required"`
	DiscogsID   *int64  `gorm:"type:bigint;uniqueIndex:idx_labels_discogs_id" json:"discogsId,omitempty"`
	Country     *string `gorm:"type:text" json:"country,omitempty"`
	FoundedYear *int    `gorm:"type:int" json:"foundedYear,omitempty"`
	Website     *string `gorm:"type:text" json:"website,omitempty"`
	ImageURL    *string `gorm:"type:text" json:"imageUrl,omitempty"`

	// Relationships
	Releases []Release `gorm:"foreignKey:LabelID" json:"releases,omitempty"`
}

func (l *Label) BeforeCreate(tx *gorm.DB) (err error) {
	if l.Name == "" {
		return gorm.ErrInvalidValue
	}
	return nil
}

func (l *Label) BeforeUpdate(tx *gorm.DB) (err error) {
	if l.Name == "" {
		return gorm.ErrInvalidValue
	}
	return nil
}

