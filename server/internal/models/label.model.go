package models

import (
	"time"

	"gorm.io/gorm"
	"waugzee/internal/utils"
)

type Label struct {
	DiscogsID   int64     `gorm:"type:bigint;primaryKey;not null" json:"discogsId" validate:"required,gt=0"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
	Name        string    `gorm:"type:text;not null;index:idx_labels_name" json:"name" validate:"required"`
	ContentHash string    `gorm:"type:varchar(64);not null;index:idx_labels_content_hash" json:"contentHash"`

	// Relationships
	Releases []Release `gorm:"foreignKey:LabelID" json:"releases,omitempty"`
}

func (l *Label) BeforeCreate(tx *gorm.DB) (err error) {
	if l.DiscogsID <= 0 {
		return gorm.ErrInvalidValue
	}
	if l.Name == "" {
		return gorm.ErrInvalidValue
	}

	// Generate content hash
	hash, err := utils.GenerateEntityHash(l)
	if err != nil {
		return err
	}
	l.ContentHash = hash

	return nil
}

func (l *Label) BeforeUpdate(tx *gorm.DB) (err error) {
	if l.Name == "" {
		return gorm.ErrInvalidValue
	}

	// Regenerate content hash
	hash, err := utils.GenerateEntityHash(l)
	if err != nil {
		return err
	}
	l.ContentHash = hash

	return nil
}

// Hashable interface implementation
func (l *Label) GetHashableFields() map[string]interface{} {
	return map[string]interface{}{
		"Name": l.Name,
	}
}

func (l *Label) SetContentHash(hash string) {
	l.ContentHash = hash
}

func (l *Label) GetContentHash() string {
	return l.ContentHash
}

func (l *Label) GetDiscogsID() int64 {
	return l.DiscogsID
}

