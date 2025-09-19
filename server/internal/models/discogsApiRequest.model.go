package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type RequestStatus string

const (
	RequestStatusPending   RequestStatus = "pending"
	RequestStatusSent      RequestStatus = "sent"
	RequestStatusCompleted RequestStatus = "completed"
	RequestStatusFailed    RequestStatus = "failed"
	RequestStatusRetrying  RequestStatus = "retrying"
)

type DiscogsApiRequest struct {
	BaseUUIDModel
	UserID          uuid.UUID       `gorm:"type:uuid;not null;index:idx_discogs_api_requests_user" json:"userId" validate:"required"`
	SyncSessionID   uuid.UUID       `gorm:"type:uuid;not null;index:idx_discogs_api_requests_sync_session" json:"syncSessionId" validate:"required"`
	RequestID       string          `gorm:"type:text;not null;uniqueIndex:idx_discogs_api_requests_request_id" json:"requestId" validate:"required"`
	URL             string          `gorm:"type:text;not null" json:"url" validate:"required"`
	Method          string          `gorm:"type:text;not null;default:'GET'" json:"method"`
	Headers         datatypes.JSON  `gorm:"type:jsonb" json:"headers,omitempty"`
	Status          RequestStatus   `gorm:"type:text;default:'pending';index:idx_discogs_api_requests_status" json:"status"`
	RetryCount      int             `gorm:"type:int;default:0" json:"retryCount"`
	ResponseStatus  *int            `gorm:"type:int" json:"responseStatus,omitempty"`
	ResponseHeaders datatypes.JSON  `gorm:"type:jsonb" json:"responseHeaders,omitempty"`
	ResponseBody    datatypes.JSON  `gorm:"type:jsonb" json:"responseBody,omitempty"`
	ErrorMessage    *string         `gorm:"type:text" json:"errorMessage,omitempty"`
	SentAt          *time.Time      `gorm:"type:timestamp" json:"sentAt,omitempty"`
	CompletedAt     *time.Time      `gorm:"type:timestamp" json:"completedAt,omitempty"`

	// Relationships
	User        *User                  `gorm:"foreignKey:UserID" json:"user,omitempty"`
	SyncSession *DiscogsCollectionSync `gorm:"foreignKey:SyncSessionID" json:"syncSession,omitempty"`
}

func (dar *DiscogsApiRequest) BeforeCreate(tx *gorm.DB) (err error) {
	if dar.UserID == uuid.Nil {
		return gorm.ErrInvalidValue
	}
	if dar.SyncSessionID == uuid.Nil {
		return gorm.ErrInvalidValue
	}
	if dar.RequestID == "" {
		return gorm.ErrInvalidValue
	}
	if dar.URL == "" {
		return gorm.ErrInvalidValue
	}
	if dar.Method == "" {
		dar.Method = "GET"
	}
	if dar.Status == "" {
		dar.Status = RequestStatusPending
	}
	return nil
}

func (dar *DiscogsApiRequest) BeforeUpdate(tx *gorm.DB) (err error) {
	if dar.UserID == uuid.Nil {
		return gorm.ErrInvalidValue
	}
	if dar.SyncSessionID == uuid.Nil {
		return gorm.ErrInvalidValue
	}
	if dar.RequestID == "" {
		return gorm.ErrInvalidValue
	}
	return nil
}

func (dar *DiscogsApiRequest) MarkAsSent() {
	now := time.Now()
	dar.Status = RequestStatusSent
	dar.SentAt = &now
}

func (dar *DiscogsApiRequest) MarkAsCompleted(responseStatus int, responseHeaders, responseBody datatypes.JSON) {
	now := time.Now()
	dar.Status = RequestStatusCompleted
	dar.ResponseStatus = &responseStatus
	dar.ResponseHeaders = responseHeaders
	dar.ResponseBody = responseBody
	dar.CompletedAt = &now
}

func (dar *DiscogsApiRequest) MarkAsFailed(errorMessage string) {
	dar.Status = RequestStatusFailed
	dar.ErrorMessage = &errorMessage
}

func (dar *DiscogsApiRequest) MarkAsRetrying() {
	dar.Status = RequestStatusRetrying
	dar.RetryCount++
}

func (dar *DiscogsApiRequest) CanRetry(maxRetries int) bool {
	return dar.RetryCount < maxRetries && dar.Status != RequestStatusCompleted
}

func (dar *DiscogsApiRequest) IsCompleted() bool {
	return dar.Status == RequestStatusCompleted
}

func (dar *DiscogsApiRequest) IsFailed() bool {
	return dar.Status == RequestStatusFailed
}

func (dar *DiscogsApiRequest) IsPending() bool {
	return dar.Status == RequestStatusPending
}