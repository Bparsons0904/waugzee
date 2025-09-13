package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Genre struct {
	BaseUUIDModel
	Name          string     `gorm:"type:text;not null;uniqueIndex:idx_genres_name" json:"name" validate:"required"`
	Description   *string    `gorm:"type:text" json:"description,omitempty"`
	ParentGenreID *uuid.UUID `gorm:"type:uuid;index:idx_genres_parent" json:"parentGenreId,omitempty"`
	Color         *string    `gorm:"type:text" json:"color,omitempty"`

	// Relationships
	ParentGenre *Genre    `gorm:"foreignKey:ParentGenreID" json:"parentGenre,omitempty"`
	SubGenres   []Genre   `gorm:"foreignKey:ParentGenreID" json:"subGenres,omitempty"`
	Releases    []Release `gorm:"many2many:release_genres;" json:"releases,omitempty"`
}

func (g *Genre) BeforeCreate(tx *gorm.DB) (err error) {
	if g.Name == "" {
		return gorm.ErrInvalidValue
	}
	return nil
}

func (g *Genre) BeforeUpdate(tx *gorm.DB) (err error) {
	if g.Name == "" {
		return gorm.ErrInvalidValue
	}
	return nil
}



