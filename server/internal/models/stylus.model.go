package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type StylusType string

const (
	StylusTypeConical     StylusType = "conical"
	StylusTypeElliptical  StylusType = "elliptical"
	StylusTypeMicroLine   StylusType = "microline"
	StylusTypeShibata     StylusType = "shibata"
	StylusTypeLineContact StylusType = "line_contact"
	StylusTypeOther       StylusType = "other"
)

type Stylus struct {
	BaseUUIDModel
	UserID                  uuid.UUID        `gorm:"type:uuid;not null;index:idx_styluses_user" json:"userId"                            validate:"required"`
	CartridgeID             *uuid.UUID       `gorm:"type:uuid;index:idx_styluses_cartridge"     json:"cartridgeId,omitempty"`
	Brand                   string           `gorm:"type:text;not null"                         json:"brand"                             validate:"required"`
	Model                   string           `gorm:"type:text;not null"                         json:"model"                             validate:"required"`
	Type                    StylusType       `gorm:"type:text;default:'elliptical'"             json:"type"`
	PurchaseDate            *time.Time       `gorm:"type:timestamp"                             json:"purchaseDate,omitempty"`
	InstallDate             *time.Time       `gorm:"type:timestamp"                             json:"installDate,omitempty"`
	HoursUsed               *decimal.Decimal `gorm:"type:decimal(8,2);default:0"                json:"hoursUsed,omitempty"`
	RecommendedReplaceHours *decimal.Decimal `gorm:"type:decimal(8,2)"                          json:"recommendedReplaceHours,omitempty"`
	Notes                   *string          `gorm:"type:text"                                  json:"notes,omitempty"`
	IsActive                bool             `gorm:"type:bool;default:true;not null"            json:"isActive"`

	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (s *Stylus) BeforeCreate(tx *gorm.DB) (err error) {
	if s.UserID == uuid.Nil {
		return gorm.ErrInvalidValue
	}
	if s.Brand == "" {
		return gorm.ErrInvalidValue
	}
	if s.Model == "" {
		return gorm.ErrInvalidValue
	}
	if s.Type == "" {
		s.Type = StylusTypeElliptical
	}
	if s.HoursUsed == nil {
		zero := decimal.NewFromInt(0)
		s.HoursUsed = &zero
	}
	return nil
}

func (s *Stylus) BeforeUpdate(tx *gorm.DB) (err error) {
	if s.UserID == uuid.Nil {
		return gorm.ErrInvalidValue
	}
	if s.Brand == "" {
		return gorm.ErrInvalidValue
	}
	if s.Model == "" {
		return gorm.ErrInvalidValue
	}
	return nil
}
