package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CartridgeType string

const (
	CartridgeTypeMovingMagnet CartridgeType = "moving_magnet"
	CartridgeTypeMovingCoil   CartridgeType = "moving_coil"
	CartridgeTypeCeramic      CartridgeType = "ceramic"
	CartridgeTypeOther        CartridgeType = "other"
)

type Cartridge struct {
	BaseUUIDModel
	UserID       uuid.UUID      `gorm:"type:uuid;not null;index:idx_cartridges_user" json:"userId" validate:"required"`
	TurntableID  *uuid.UUID     `gorm:"type:uuid;index:idx_cartridges_turntable" json:"turntableId,omitempty"`
	Brand        string         `gorm:"type:text;not null" json:"brand" validate:"required"`
	Model        string         `gorm:"type:text;not null" json:"model" validate:"required"`
	Type         CartridgeType  `gorm:"type:text;default:'moving_magnet'" json:"type"`
	PurchaseDate *time.Time     `gorm:"type:timestamp" json:"purchaseDate,omitempty"`
	InstallDate  *time.Time     `gorm:"type:timestamp" json:"installDate,omitempty"`
	Notes        *string        `gorm:"type:text" json:"notes,omitempty"`
	IsActive     bool           `gorm:"type:bool;default:true;not null" json:"isActive"`
	StylusID     *uuid.UUID     `gorm:"type:uuid;index:idx_cartridges_stylus" json:"stylusId,omitempty"`

	// Relationships
	User               *User               `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Turntable          *Turntable          `gorm:"foreignKey:TurntableID" json:"turntable,omitempty"`
	CurrentStylus      *Stylus             `gorm:"foreignKey:StylusID" json:"currentStylus,omitempty"`
	Styluses           []Stylus            `gorm:"foreignKey:CartridgeID" json:"styluses,omitempty"`
	PlaySessions       []PlaySession       `gorm:"foreignKey:CartridgeID" json:"playSessions,omitempty"`
	MaintenanceRecords []MaintenanceRecord `gorm:"foreignKey:CartridgeID" json:"maintenanceRecords,omitempty"`
}

func (c *Cartridge) BeforeCreate(tx *gorm.DB) (err error) {
	if c.UserID == uuid.Nil {
		return gorm.ErrInvalidValue
	}
	if c.Brand == "" {
		return gorm.ErrInvalidValue
	}
	if c.Model == "" {
		return gorm.ErrInvalidValue
	}
	if c.Type == "" {
		c.Type = CartridgeTypeMovingMagnet
	}
	return nil
}

func (c *Cartridge) BeforeUpdate(tx *gorm.DB) (err error) {
	if c.UserID == uuid.Nil {
		return gorm.ErrInvalidValue
	}
	if c.Brand == "" {
		return gorm.ErrInvalidValue
	}
	if c.Model == "" {
		return gorm.ErrInvalidValue
	}
	return nil
}



