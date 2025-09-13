package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PlaySession struct {
	BaseUUIDModel
	UserID           uuid.UUID  `gorm:"type:uuid;not null;index:idx_play_sessions_user" json:"userId" validate:"required"`
	ReleaseID        uuid.UUID  `gorm:"type:uuid;not null;index:idx_play_sessions_release" json:"releaseId" validate:"required"`
	UserCollectionID *uuid.UUID `gorm:"type:uuid;index:idx_play_sessions_collection" json:"userCollectionId,omitempty"`
	TurntableID      *uuid.UUID `gorm:"type:uuid;index:idx_play_sessions_turntable" json:"turntableId,omitempty"`
	CartridgeID      *uuid.UUID `gorm:"type:uuid;index:idx_play_sessions_cartridge" json:"cartridgeId,omitempty"`
	StylusID         *uuid.UUID `gorm:"type:uuid;index:idx_play_sessions_stylus" json:"stylusId,omitempty"`
	PlayedAt         time.Time  `gorm:"type:timestamp;not null;index:idx_play_sessions_played_at" json:"playedAt" validate:"required"`
	Duration         *int       `gorm:"type:int" json:"duration,omitempty"` // Duration in seconds
	Rating           *int       `gorm:"type:int;check:rating >= 1 AND rating <= 10" json:"rating,omitempty"`
	Notes            *string    `gorm:"type:text" json:"notes,omitempty"`
	Location         *string    `gorm:"type:text" json:"location,omitempty"`

	// Relationships
	User           *User           `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Release        *Release        `gorm:"foreignKey:ReleaseID" json:"release,omitempty"`
	UserCollection *UserCollection `gorm:"foreignKey:UserCollectionID" json:"userCollection,omitempty"`
	Turntable      *Turntable      `gorm:"foreignKey:TurntableID" json:"turntable,omitempty"`
	Cartridge      *Cartridge      `gorm:"foreignKey:CartridgeID" json:"cartridge,omitempty"`
	Stylus         *Stylus         `gorm:"foreignKey:StylusID" json:"stylus,omitempty"`
}

func (ps *PlaySession) BeforeCreate(tx *gorm.DB) (err error) {
	if ps.UserID == uuid.Nil {
		return gorm.ErrInvalidValue
	}
	if ps.ReleaseID == uuid.Nil {
		return gorm.ErrInvalidValue
	}
	if ps.PlayedAt.IsZero() {
		ps.PlayedAt = time.Now()
	}
	if ps.Rating != nil && (*ps.Rating < 1 || *ps.Rating > 10) {
		return gorm.ErrInvalidValue
	}
	return nil
}

func (ps *PlaySession) BeforeUpdate(tx *gorm.DB) (err error) {
	if ps.UserID == uuid.Nil {
		return gorm.ErrInvalidValue
	}
	if ps.ReleaseID == uuid.Nil {
		return gorm.ErrInvalidValue
	}
	if ps.Rating != nil && (*ps.Rating < 1 || *ps.Rating > 10) {
		return gorm.ErrInvalidValue
	}
	return nil
}



