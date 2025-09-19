package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SyncStatus is defined in discogsCollectionSync.model.go to avoid duplication

type DiscogsSettings struct {
	AutoSync            bool `json:"autoSync"`
	SyncCollection      bool `json:"syncCollection"`
	SyncWantlist        bool `json:"syncWantlist"`
	SyncInterval        int  `json:"syncInterval"` // Hours between syncs
	ImportPrices        bool `json:"importPrices"`
	ImportCondition     bool `json:"importCondition"`
	ImportNotes         bool `json:"importNotes"`
	OverwriteExisting   bool `json:"overwriteExisting"`
}

type DiscogsSync struct {
	BaseUUIDModel
	UserID          uuid.UUID        `gorm:"type:uuid;not null;uniqueIndex:idx_discogs_sync_user" json:"userId" validate:"required"`
	LastSyncAt      *time.Time       `gorm:"type:timestamp" json:"lastSyncAt,omitempty"`
	SyncStatus      SyncStatus       `gorm:"type:text;default:'never_synced'" json:"syncStatus"`
	ErrorMessage    *string          `gorm:"type:text" json:"errorMessage,omitempty"`
	CollectionCount *int             `gorm:"type:int" json:"collectionCount,omitempty"`
	WantlistCount   *int             `gorm:"type:int" json:"wantlistCount,omitempty"`
	Settings        *DiscogsSettings `gorm:"type:jsonb" json:"settings,omitempty"`

	// Relationships
	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (ds *DiscogsSync) BeforeCreate(tx *gorm.DB) (err error) {
	if ds.UserID == uuid.Nil {
		return gorm.ErrInvalidValue
	}
	if ds.SyncStatus == "" {
		ds.SyncStatus = SyncStatusNeverSynced
	}
	if ds.Settings == nil {
		ds.Settings = &DiscogsSettings{
			AutoSync:          false,
			SyncCollection:    true,
			SyncWantlist:      false,
			SyncInterval:      24, // 24 hours
			ImportPrices:      true,
			ImportCondition:   true,
			ImportNotes:       true,
			OverwriteExisting: false,
		}
	}
	return nil
}

func (ds *DiscogsSync) BeforeUpdate(tx *gorm.DB) (err error) {
	if ds.UserID == uuid.Nil {
		return gorm.ErrInvalidValue
	}
	return nil
}



