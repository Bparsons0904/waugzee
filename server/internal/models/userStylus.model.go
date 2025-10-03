package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type UserStylus struct {
	BaseUUIDModel
	UserID                  uuid.UUID        `gorm:"type:uuid;not null;index:idx_user_styluses_user"    json:"userId"                            validate:"required"`
	StylusID                uuid.UUID        `gorm:"type:uuid;not null;index:idx_user_styluses_stylus"  json:"stylusId"                          validate:"required"`
	PurchaseDate            *time.Time       `gorm:"type:timestamp"                                      json:"purchaseDate,omitempty"`
	InstallDate             *time.Time       `gorm:"type:timestamp"                                      json:"installDate,omitempty"`
	HoursUsed               *decimal.Decimal `gorm:"type:decimal(8,2);default:0"                         json:"hoursUsed,omitempty"`
	Notes                   *string          `gorm:"type:text"                                           json:"notes,omitempty"`
	IsActive                bool             `gorm:"type:bool;default:true;not null"                     json:"isActive"`

	User    *User    `gorm:"foreignKey:UserID"    json:"user,omitempty"`
	Stylus  *Stylus  `gorm:"foreignKey:StylusID"  json:"stylus,omitempty"`
}

func (us *UserStylus) BeforeCreate(tx *gorm.DB) (err error) {
	if us.UserID == uuid.Nil {
		return gorm.ErrInvalidValue
	}
	if us.StylusID == uuid.Nil {
		return gorm.ErrInvalidValue
	}
	if us.HoursUsed == nil {
		zero := decimal.NewFromInt(0)
		us.HoursUsed = &zero
	}
	return nil
}

func (us *UserStylus) BeforeUpdate(tx *gorm.DB) (err error) {
	if us.UserID == uuid.Nil {
		return gorm.ErrInvalidValue
	}
	if us.StylusID == uuid.Nil {
		return gorm.ErrInvalidValue
	}
	return nil
}
