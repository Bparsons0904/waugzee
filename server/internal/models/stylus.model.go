package models

import (
	"github.com/google/uuid"
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
	Brand                   string     `gorm:"type:text;not null"                        json:"brand"`
	Model                   string     `gorm:"type:text;not null"                        json:"model"`
	Type                    StylusType `gorm:"type:text;default:'elliptical'"            json:"type"`
	RecommendedReplaceHours *int       `gorm:"type:integer"                              json:"recommendedReplaceHours,omitempty"`
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
