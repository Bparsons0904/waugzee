package types

import (
	"time"

	"github.com/google/uuid"
)

// SyncStatus represents the current status of a sync operation
type SyncStatus string

const (
	SyncStatusStarted    SyncStatus = "started"
	SyncStatusInProgress SyncStatus = "in_progress"
	SyncStatusCompleted  SyncStatus = "completed"
	SyncStatusError      SyncStatus = "error"
	SyncStatusCancelled  SyncStatus = "cancelled"
)

// SyncType represents the type of sync operation
type SyncType string

const (
	SyncTypeCollection SyncType = "collection"
	SyncTypeWantlist   SyncType = "wantlist"
	SyncTypeFolders    SyncType = "folders"
	SyncTypeFull       SyncType = "full"
)

// SyncProgressData represents the data structure for sync progress events
type SyncProgressData struct {
	SyncID   string     `json:"syncId"`
	Status   SyncStatus `json:"status"`
	Progress int        `json:"progress"`
	Total    int        `json:"total"`
	Message  string     `json:"message"`
}

// SyncStartedData represents the data structure for sync started events
type SyncStartedData struct {
	SyncID   string   `json:"syncId"`
	SyncType SyncType `json:"syncType"`
	Status   SyncStatus `json:"status"`
}

// SyncCompletedData represents the data structure for sync completed events
type SyncCompletedData struct {
	SyncID   string         `json:"syncId"`
	SyncType SyncType       `json:"syncType"`
	Status   SyncStatus     `json:"status"`
	Summary  SyncSummary    `json:"summary"`
}

// SyncErrorData represents the data structure for sync error events
type SyncErrorData struct {
	SyncID  string     `json:"syncId"`
	Status  SyncStatus `json:"status"`
	Error   string     `json:"error"`
	Details map[string]any `json:"details,omitempty"`
}

// SyncCancelledData represents the data structure for sync cancelled events
type SyncCancelledData struct {
	SyncID string     `json:"syncId"`
	Status SyncStatus `json:"status"`
	Reason string     `json:"reason"`
}

// SyncSummary represents the summary data after a sync completion
type SyncSummary struct {
	ProcessedItems int           `json:"processedItems"`
	NewItems       int           `json:"newItems"`
	UpdatedItems   int           `json:"updatedItems"`
	Errors         int           `json:"errors"`
	Duration       time.Duration `json:"duration"`
	StartTime      time.Time     `json:"startTime"`
	EndTime        time.Time     `json:"endTime"`
}

// ToMap converts the sync summary to a map for event publishing
func (s SyncSummary) ToMap() map[string]any {
	return map[string]any{
		"processedItems": s.ProcessedItems,
		"newItems":       s.NewItems,
		"updatedItems":   s.UpdatedItems,
		"errors":         s.Errors,
		"duration":       s.Duration.String(),
		"startTime":      s.StartTime,
		"endTime":        s.EndTime,
	}
}

// SyncState represents the current state of a sync operation stored in cache
type SyncState struct {
	ID         string         `json:"id"`
	UserID     uuid.UUID      `json:"userId"`
	Type       SyncType       `json:"type"`
	Status     SyncStatus     `json:"status"`
	Progress   int            `json:"progress"`
	Total      int            `json:"total"`
	Message    string         `json:"message"`
	StartTime  time.Time      `json:"startTime"`
	UpdateTime time.Time      `json:"updateTime"`
	EndTime    *time.Time     `json:"endTime,omitempty"`
	Summary    *SyncSummary   `json:"summary,omitempty"`
	Error      *string        `json:"error,omitempty"`
	Details    map[string]any `json:"details,omitempty"`
}

// CreateSyncState creates a new sync state with default values
func CreateSyncState(userID uuid.UUID, syncType SyncType) *SyncState {
	return &SyncState{
		ID:         uuid.New().String(),
		UserID:     userID,
		Type:       syncType,
		Status:     SyncStatusStarted,
		Progress:   0,
		Total:      0,
		Message:    "Sync initialized",
		StartTime:  time.Now(),
		UpdateTime: time.Now(),
	}
}

// UpdateProgress updates the sync state with progress information
func (s *SyncState) UpdateProgress(progress, total int, message string) {
	s.Progress = progress
	s.Total = total
	s.Message = message
	s.Status = SyncStatusInProgress
	s.UpdateTime = time.Now()
}

// CompleteSync marks the sync as completed with summary
func (s *SyncState) CompleteSync(summary SyncSummary) {
	s.Status = SyncStatusCompleted
	s.Summary = &summary
	s.UpdateTime = time.Now()
	endTime := time.Now()
	s.EndTime = &endTime
}

// ErrorSync marks the sync as failed with error details
func (s *SyncState) ErrorSync(errorMessage string, details map[string]any) {
	s.Status = SyncStatusError
	s.Error = &errorMessage
	s.Details = details
	s.UpdateTime = time.Now()
	endTime := time.Now()
	s.EndTime = &endTime
}

// CancelSync marks the sync as cancelled
func (s *SyncState) CancelSync(reason string) {
	s.Status = SyncStatusCancelled
	s.Message = reason
	s.UpdateTime = time.Now()
	endTime := time.Now()
	s.EndTime = &endTime
}

// ToMap converts the sync state to a map for storage in Valkey
func (s *SyncState) ToMap() map[string]any {
	return map[string]any{
		"id":         s.ID,
		"userId":     s.UserID.String(),
		"type":       string(s.Type),
		"status":     string(s.Status),
		"progress":   s.Progress,
		"total":      s.Total,
		"message":    s.Message,
		"startTime":  s.StartTime,
		"updateTime": s.UpdateTime,
		"endTime":    s.EndTime,
		"summary":    s.Summary,
		"error":      s.Error,
		"details":    s.Details,
	}
}