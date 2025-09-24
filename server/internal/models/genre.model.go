package models

type Genre struct {
	BaseDiscogModel
	Name          string `gorm:"type:text;not null;uniqueIndex:idx_genres_name" json:"name"                    validate:"required"`
	ParentGenreID *int   `gorm:"type:int;index:idx_genres_parent"               json:"parentGenreId,omitempty"`

	// Relationships
	ParentGenre *Genre    `gorm:"foreignKey:ParentGenreID"  json:"parentGenre,omitempty"`
	SubGenres   []Genre   `gorm:"foreignKey:ParentGenreID"  json:"subGenres,omitempty"`
	Releases    []Release `gorm:"many2many:release_genres;" json:"releases,omitempty"`
}
