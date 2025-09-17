package models

import (
	"time"

	"gorm.io/gorm"
	"waugzee/internal/utils"
)

type Track struct {
	ID          int       `gorm:"type:int;primaryKey;autoIncrement" json:"id"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
	ReleaseID   int64     `gorm:"type:bigint;not null;index:idx_tracks_release" json:"releaseId" validate:"required"`
	Position    string    `gorm:"type:text;not null" json:"position" validate:"required"`
	Title       string    `gorm:"type:text;not null" json:"title" validate:"required"`
	Duration    *int      `gorm:"type:int" json:"duration,omitempty"` // Duration in seconds
	ContentHash string    `gorm:"type:varchar(64);not null;index:idx_tracks_content_hash" json:"contentHash"`

	// Relationships
	Release *Release `gorm:"foreignKey:ReleaseID" json:"release,omitempty"`
}

func (t *Track) BeforeCreate(tx *gorm.DB) (err error) {
	if t.ReleaseID <= 0 {
		return gorm.ErrInvalidValue
	}
	if t.Position == "" {
		return gorm.ErrInvalidValue
	}
	if t.Title == "" {
		return gorm.ErrInvalidValue
	}

	// Generate content hash
	hash, err := utils.GenerateEntityHash(t)
	if err != nil {
		return err
	}
	t.ContentHash = hash

	return nil
}

func (t *Track) BeforeUpdate(tx *gorm.DB) (err error) {
	if t.ReleaseID <= 0 {
		return gorm.ErrInvalidValue
	}
	if t.Position == "" {
		return gorm.ErrInvalidValue
	}
	if t.Title == "" {
		return gorm.ErrInvalidValue
	}

	// Regenerate content hash
	hash, err := utils.GenerateEntityHash(t)
	if err != nil {
		return err
	}
	t.ContentHash = hash

	return nil
}

// Hashable interface implementation
func (t *Track) GetHashableFields() map[string]interface{} {
	return map[string]interface{}{
		"ReleaseID": t.ReleaseID,
		"Position":  t.Position,
		"Title":     t.Title,
		"Duration":  t.Duration,
	}
}

func (t *Track) SetContentHash(hash string) {
	t.ContentHash = hash
}

func (t *Track) GetContentHash() string {
	return t.ContentHash
}



