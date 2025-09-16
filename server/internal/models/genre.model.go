package models

import (
	"gorm.io/gorm"
	"waugzee/internal/utils"
)

type Genre struct {
	BaseModel
	Name          string `gorm:"type:text;not null;uniqueIndex:idx_genres_name" json:"name" validate:"required"`
	ParentGenreID *int   `gorm:"type:int;index:idx_genres_parent" json:"parentGenreId,omitempty"`
	ContentHash   string `gorm:"type:varchar(64);not null;index:idx_genres_content_hash" json:"contentHash"`

	// Relationships
	ParentGenre *Genre    `gorm:"foreignKey:ParentGenreID" json:"parentGenre,omitempty"`
	SubGenres   []Genre   `gorm:"foreignKey:ParentGenreID" json:"subGenres,omitempty"`
	Releases    []Release `gorm:"many2many:release_genres;" json:"releases,omitempty"`
}

func (g *Genre) BeforeCreate(tx *gorm.DB) (err error) {
	if g.Name == "" {
		return gorm.ErrInvalidValue
	}

	// Generate content hash
	hash, err := utils.GenerateEntityHash(g)
	if err != nil {
		return err
	}
	g.ContentHash = hash

	return nil
}

func (g *Genre) BeforeUpdate(tx *gorm.DB) (err error) {
	if g.Name == "" {
		return gorm.ErrInvalidValue
	}

	// Regenerate content hash
	hash, err := utils.GenerateEntityHash(g)
	if err != nil {
		return err
	}
	g.ContentHash = hash

	return nil
}

// Hashable interface implementation
func (g *Genre) GetHashableFields() map[string]interface{} {
	return map[string]interface{}{
		"Name":          g.Name,
		"ParentGenreID": g.ParentGenreID,
	}
}

func (g *Genre) SetContentHash(hash string) {
	g.ContentHash = hash
}

func (g *Genre) GetContentHash() string {
	return g.ContentHash
}
