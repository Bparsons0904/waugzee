package models

import (
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
	Brand                   string           `gorm:"type:text;not null"         json:"brand"                             validate:"required"`
	Model                   string           `gorm:"type:text;not null"         json:"model"                             validate:"required"`
	Type                    StylusType       `gorm:"type:text;default:'elliptical'" json:"type"`
	RecommendedReplaceHours *decimal.Decimal `gorm:"type:decimal(8,2)"          json:"recommendedReplaceHours,omitempty"`
}

func (s *Stylus) BeforeCreate(tx *gorm.DB) (err error) {
	if s.Brand == "" {
		return gorm.ErrInvalidValue
	}
	if s.Model == "" {
		return gorm.ErrInvalidValue
	}
	if s.Type == "" {
		s.Type = StylusTypeElliptical
	}
	return nil
}

func (s *Stylus) BeforeUpdate(tx *gorm.DB) (err error) {
	if s.Brand == "" {
		return gorm.ErrInvalidValue
	}
	if s.Model == "" {
		return gorm.ErrInvalidValue
	}
	return nil
}
