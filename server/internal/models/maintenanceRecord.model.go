package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type MaintenanceType string

const (
	MaintenanceTypeEquipmentCleaning MaintenanceType = "equipment_cleaning"
	MaintenanceTypeRecordCleaning    MaintenanceType = "record_cleaning"
	MaintenanceTypeStylusReplacement MaintenanceType = "stylus_replacement"
	MaintenanceTypeCartridgeSetup    MaintenanceType = "cartridge_setup"
	MaintenanceTypeTurntableSetup    MaintenanceType = "turntable_setup"
	MaintenanceTypeCalibration       MaintenanceType = "calibration"
	MaintenanceTypeOther             MaintenanceType = "other"
)

type MaintenanceTargetType string

const (
	MaintenanceTargetTurntable  MaintenanceTargetType = "turntable"
	MaintenanceTargetCartridge  MaintenanceTargetType = "cartridge"
	MaintenanceTargetStylus     MaintenanceTargetType = "stylus"
	MaintenanceTargetRecord     MaintenanceTargetType = "record"
)

type MaintenanceRecord struct {
	BaseUUIDModel
	UserID           uuid.UUID              `gorm:"type:uuid;not null;index:idx_maintenance_records_user" json:"userId" validate:"required"`
	Type             MaintenanceType        `gorm:"type:text;not null;index:idx_maintenance_records_type" json:"type" validate:"required"`
	TargetType       MaintenanceTargetType  `gorm:"type:text;not null" json:"targetType" validate:"required"`
	TargetID         uuid.UUID              `gorm:"type:uuid;not null;index:idx_maintenance_records_target" json:"targetId" validate:"required"`
	UserCollectionID *uuid.UUID             `gorm:"type:uuid;index:idx_maintenance_records_collection" json:"userCollectionId,omitempty"`
	TurntableID      *uuid.UUID             `gorm:"type:uuid;index:idx_maintenance_records_turntable" json:"turntableId,omitempty"`
	CartridgeID      *uuid.UUID             `gorm:"type:uuid;index:idx_maintenance_records_cartridge" json:"cartridgeId,omitempty"`
	StylusID         *uuid.UUID             `gorm:"type:uuid;index:idx_maintenance_records_stylus" json:"stylusId,omitempty"`
	PerformedAt      time.Time              `gorm:"type:timestamp;not null;index:idx_maintenance_records_performed_at" json:"performedAt" validate:"required"`
	Notes            *string                `gorm:"type:text" json:"notes,omitempty"`
	Cost             *decimal.Decimal       `gorm:"type:decimal(10,2)" json:"cost,omitempty"`
	NextDueDate      *time.Time             `gorm:"type:timestamp" json:"nextDueDate,omitempty"`

	// Relationships
	User           *User           `gorm:"foreignKey:UserID" json:"user,omitempty"`
	UserCollection *UserCollection `gorm:"foreignKey:UserCollectionID" json:"userCollection,omitempty"`
	Turntable      *Turntable      `gorm:"foreignKey:TurntableID" json:"turntable,omitempty"`
	Cartridge      *Cartridge      `gorm:"foreignKey:CartridgeID" json:"cartridge,omitempty"`
	Stylus         *Stylus         `gorm:"foreignKey:StylusID" json:"stylus,omitempty"`
}

func (mr *MaintenanceRecord) BeforeCreate(tx *gorm.DB) (err error) {
	if mr.UserID == uuid.Nil {
		return gorm.ErrInvalidValue
	}
	if mr.Type == "" {
		return gorm.ErrInvalidValue
	}
	if mr.TargetType == "" {
		return gorm.ErrInvalidValue
	}
	if mr.TargetID == uuid.Nil {
		return gorm.ErrInvalidValue
	}
	if mr.PerformedAt.IsZero() {
		mr.PerformedAt = time.Now()
	}
	return nil
}

func (mr *MaintenanceRecord) BeforeUpdate(tx *gorm.DB) (err error) {
	if mr.UserID == uuid.Nil {
		return gorm.ErrInvalidValue
	}
	if mr.Type == "" {
		return gorm.ErrInvalidValue
	}
	if mr.TargetType == "" {
		return gorm.ErrInvalidValue
	}
	if mr.TargetID == uuid.Nil {
		return gorm.ErrInvalidValue
	}
	return nil
}



