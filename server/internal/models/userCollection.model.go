package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RecordCondition string

const (
	ConditionMint     RecordCondition = "mint"
	ConditionNearMint RecordCondition = "near_mint"
	ConditionVeryGood RecordCondition = "very_good"
	ConditionGood     RecordCondition = "good"
	ConditionFair     RecordCondition = "fair"
	ConditionPoor     RecordCondition = "poor"
)

type UserCollection struct {
	BaseUUIDModel
	UserID              uuid.UUID        `gorm:"type:uuid;not null;index:idx_user_collections_user" json:"userId" validate:"required"`
	ReleaseID           uuid.UUID        `gorm:"type:uuid;not null;index:idx_user_collections_release" json:"releaseId" validate:"required"`
	Condition           *RecordCondition `gorm:"type:text" json:"condition,omitempty"`
	PurchaseDate        *time.Time       `gorm:"type:timestamp" json:"purchaseDate,omitempty"`
	PurchaseLocation    *string          `gorm:"type:text" json:"purchaseLocation,omitempty"`
	Notes               *string          `gorm:"type:text" json:"notes,omitempty"`
	StorageLocation     *string          `gorm:"type:text" json:"storageLocation,omitempty"`
	DiscogsInstanceID   *int64           `gorm:"type:bigint" json:"discogsInstanceId,omitempty"`

	// Relationships
	User     *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Release  *Release  `gorm:"foreignKey:ReleaseID" json:"release,omitempty"`
	PlaySessions []PlaySession `gorm:"foreignKey:UserCollectionID" json:"playSessions,omitempty"`
	MaintenanceRecords []MaintenanceRecord `gorm:"foreignKey:UserCollectionID" json:"maintenanceRecords,omitempty"`
}

func (uc *UserCollection) BeforeCreate(tx *gorm.DB) (err error) {
	if uc.UserID == uuid.Nil {
		return gorm.ErrInvalidValue
	}
	if uc.ReleaseID == uuid.Nil {
		return gorm.ErrInvalidValue
	}
	return nil
}

func (uc *UserCollection) BeforeUpdate(tx *gorm.DB) (err error) {
	if uc.UserID == uuid.Nil {
		return gorm.ErrInvalidValue
	}
	if uc.ReleaseID == uuid.Nil {
		return gorm.ErrInvalidValue
	}
	return nil
}





// Define unique constraint on UserID + ReleaseID
func (UserCollection) Constraints() []string {
	return []string{
		"UNIQUE(user_id, release_id)",
	}
}