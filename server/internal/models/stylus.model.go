package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type StylusType string

const (
	StylusTypeConical     StylusType = "Conical"
	StylusTypeElliptical  StylusType = "Elliptical"
	StylusTypeMicroLine   StylusType = "Microline"
	StylusTypeShibata     StylusType = "Shibata"
	StylusTypeLineContact StylusType = "Line Contact"
	StylusTypeOther       StylusType = "Other"
)

type Stylus struct {
	BaseUUIDModel
	Brand                   string     `gorm:"type:text;not null"                        json:"brand"`
	Model                   string     `gorm:"type:text;not null"                        json:"model"`
	Type                    StylusType `gorm:"type:text;default:'elliptical'"            json:"type"`
	RecommendedReplaceHours *int       `gorm:"type:integer;default:1000"                 json:"recommendedReplaceHours"`
	UserGeneratedID         *uuid.UUID `gorm:"type:uuid;index:idx_stylus_user_generated" json:"userGeneratedId,omitempty"`
	IsVerified              bool       `gorm:"type:bool;default:false;not null"          json:"isVerified"`
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
