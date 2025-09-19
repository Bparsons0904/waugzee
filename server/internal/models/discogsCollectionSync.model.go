package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SyncStatus string

const (
	SyncStatusInitiated  SyncStatus = "initiated"
	SyncStatusInProgress SyncStatus = "in_progress"
	SyncStatusPaused     SyncStatus = "paused"
	SyncStatusCompleted  SyncStatus = "completed"
	SyncStatusFailed     SyncStatus = "failed"
	SyncStatusCancelled  SyncStatus = "cancelled"
)

type SyncType string

const (
	SyncTypeCollection SyncType = "collection"
	SyncTypeWantlist   SyncType = "wantlist"
)

type DiscogsCollectionSync struct {
	BaseUUIDModel
	UserID            uuid.UUID `gorm:"type:uuid;not null;index:idx_discogs_collection_syncs_user" json:"userId" validate:"required"`
	SessionID         string    `gorm:"type:text;not null;uniqueIndex:idx_discogs_collection_syncs_session" json:"sessionId" validate:"required"`
	Status            SyncStatus `gorm:"type:text;default:'initiated';index:idx_discogs_collection_syncs_status" json:"status"`
	SyncType          SyncType  `gorm:"type:text;not null" json:"syncType" validate:"required"`
	TotalRequests     int       `gorm:"type:int;default:0" json:"totalRequests"`
	CompletedRequests int       `gorm:"type:int;default:0" json:"completedRequests"`
	FailedRequests    int       `gorm:"type:int;default:0" json:"failedRequests"`
	StartedAt         time.Time `gorm:"autoCreateTime" json:"startedAt"`
	CompletedAt       *time.Time `gorm:"type:timestamp" json:"completedAt,omitempty"`
	PausedAt          *time.Time `gorm:"type:timestamp" json:"pausedAt,omitempty"`
	ErrorMessage      *string   `gorm:"type:text" json:"errorMessage,omitempty"`
	LastPage          *int      `gorm:"type:int" json:"lastPage,omitempty"`
	TotalPages        *int      `gorm:"type:int" json:"totalPages,omitempty"`
	FullSync          bool      `gorm:"type:bool;default:false" json:"fullSync"`
	PageLimit         *int      `gorm:"type:int" json:"pageLimit,omitempty"`

	// Relationships
	User        *User                 `gorm:"foreignKey:UserID" json:"user,omitempty"`
	ApiRequests []DiscogsApiRequest   `gorm:"foreignKey:SyncSessionID;references:ID" json:"apiRequests,omitempty"`
}

func (dcs *DiscogsCollectionSync) BeforeCreate(tx *gorm.DB) (err error) {
	if dcs.UserID == uuid.Nil {
		return gorm.ErrInvalidValue
	}
	if dcs.SessionID == "" {
		return gorm.ErrInvalidValue
	}
	if dcs.SyncType == "" {
		return gorm.ErrInvalidValue
	}
	if dcs.Status == "" {
		dcs.Status = SyncStatusInitiated
	}
	return nil
}

func (dcs *DiscogsCollectionSync) BeforeUpdate(tx *gorm.DB) (err error) {
	if dcs.UserID == uuid.Nil {
		return gorm.ErrInvalidValue
	}
	if dcs.SessionID == "" {
		return gorm.ErrInvalidValue
	}
	return nil
}

func (dcs *DiscogsCollectionSync) MarkAsInProgress() {
	dcs.Status = SyncStatusInProgress
	dcs.PausedAt = nil
}

func (dcs *DiscogsCollectionSync) MarkAsCompleted() {
	now := time.Now()
	dcs.Status = SyncStatusCompleted
	dcs.CompletedAt = &now
	dcs.PausedAt = nil
}

func (dcs *DiscogsCollectionSync) MarkAsFailed(errorMessage string) {
	dcs.Status = SyncStatusFailed
	dcs.ErrorMessage = &errorMessage
	dcs.PausedAt = nil
}

func (dcs *DiscogsCollectionSync) MarkAsPaused() {
	now := time.Now()
	dcs.Status = SyncStatusPaused
	dcs.PausedAt = &now
}

func (dcs *DiscogsCollectionSync) MarkAsCancelled() {
	dcs.Status = SyncStatusCancelled
	dcs.PausedAt = nil
}

func (dcs *DiscogsCollectionSync) UpdateProgress(completed, failed int) {
	dcs.CompletedRequests = completed
	dcs.FailedRequests = failed
}

func (dcs *DiscogsCollectionSync) GetPercentComplete() float64 {
	if dcs.TotalRequests == 0 {
		return 0.0
	}
	return float64(dcs.CompletedRequests+dcs.FailedRequests) / float64(dcs.TotalRequests) * 100.0
}

func (dcs *DiscogsCollectionSync) IsActive() bool {
	return dcs.Status == SyncStatusInitiated || dcs.Status == SyncStatusInProgress
}

func (dcs *DiscogsCollectionSync) IsPaused() bool {
	return dcs.Status == SyncStatusPaused
}

func (dcs *DiscogsCollectionSync) IsCompleted() bool {
	return dcs.Status == SyncStatusCompleted
}

func (dcs *DiscogsCollectionSync) IsFailed() bool {
	return dcs.Status == SyncStatusFailed
}

func (dcs *DiscogsCollectionSync) IsCancelled() bool {
	return dcs.Status == SyncStatusCancelled
}

func (dcs *DiscogsCollectionSync) CanResume() bool {
	return dcs.Status == SyncStatusPaused || dcs.Status == SyncStatusInitiated
}

func (dcs *DiscogsCollectionSync) GetEstimatedTimeLeft() *time.Duration {
	if dcs.TotalRequests == 0 || dcs.CompletedRequests == 0 {
		return nil
	}

	elapsed := time.Since(dcs.StartedAt)
	avgTimePerRequest := elapsed / time.Duration(dcs.CompletedRequests)
	remainingRequests := dcs.TotalRequests - dcs.CompletedRequests - dcs.FailedRequests

	if remainingRequests <= 0 {
		return nil
	}

	estimatedTime := time.Duration(remainingRequests) * avgTimePerRequest
	return &estimatedTime
}