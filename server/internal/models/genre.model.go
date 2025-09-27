package models

import (
	"strings"
	"unicode/utf8"

	"gorm.io/gorm"
)

type Genre struct {
	BaseDiscogModel
	Name      string    `gorm:"type:text;not null;uniqueIndex:idx_genres_name_type,priority:1" json:"name"`
	Type      string    `gorm:"type:text;not null;uniqueIndex:idx_genres_name_type,priority:2" json:"type"`
	NameLower string    `gorm:"type:text;not null;index:idx_genres_name_lower"                 json:"nameLower"`
	Releases  []Release `gorm:"many2many:release_genres;"                                      json:"releases,omitempty"`
	Masters   []Master  `gorm:"many2many:master_genres;"                                       json:"masters,omitempty"`
}

func (g *Genre) BeforeSave(tx *gorm.DB) error {
	if !utf8.ValidString(g.Name) || strings.Contains(g.Name, "\x00") {
		g.Name = strings.ToValidUTF8(strings.ReplaceAll(g.Name, "\x00", ""), "")
	}
	g.NameLower = strings.ToLower(g.Name)
	return nil
}
