package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Turntable struct {
	BaseUUIDModel
	UserID        uuid.UUID  `gorm:"type:uuid;not null;index:idx_turntables_user" json:"userId" validate:"required"`
	Brand         string     `gorm:"type:text;not null" json:"brand" validate:"required"`
	Model         string     `gorm:"type:text;not null" json:"model" validate:"required"`
	SerialNumber  *string    `gorm:"type:text" json:"serialNumber,omitempty"`
	PurchaseDate  *time.Time `gorm:"type:timestamp" json:"purchaseDate,omitempty"`
	Notes         *string    `gorm:"type:text" json:"notes,omitempty"`
	IsActive      bool       `gorm:"type:bool;default:true;not null" json:"isActive"`
	CartridgeID   *uuid.UUID `gorm:"type:uuid;index:idx_turntables_cartridge" json:"cartridgeId,omitempty"`

	// Relationships
	User               *User               `gorm:"foreignKey:UserID" json:"user,omitempty"`
	CurrentCartridge   *Cartridge          `gorm:"foreignKey:CartridgeID" json:"currentCartridge,omitempty"`
	Cartridges         []Cartridge         `gorm:"foreignKey:TurntableID" json:"cartridges,omitempty"`
	PlaySessions       []PlaySession       `gorm:"foreignKey:TurntableID" json:"playSessions,omitempty"`
	MaintenanceRecords []MaintenanceRecord `gorm:"foreignKey:TurntableID" json:"maintenanceRecords,omitempty"`
}

func (t *Turntable) BeforeCreate(tx *gorm.DB) (err error) {
	if t.UserID == uuid.Nil {
		return gorm.ErrInvalidValue
	}
	if t.Brand == "" {
		return gorm.ErrInvalidValue
	}
	if t.Model == "" {
		return gorm.ErrInvalidValue
	}
	return nil
}

func (t *Turntable) BeforeUpdate(tx *gorm.DB) (err error) {
	if t.UserID == uuid.Nil {
		return gorm.ErrInvalidValue
	}
	if t.Brand == "" {
		return gorm.ErrInvalidValue
	}
	if t.Model == "" {
		return gorm.ErrInvalidValue
	}
	return nil
}



