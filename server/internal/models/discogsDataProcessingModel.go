package models

import (
	"regexp"
	"slices"
	"time"

	"gorm.io/gorm"
)

type ProcessingStatus string

const (
	ProcessingStatusNotStarted         ProcessingStatus = "not_started"
	ProcessingStatusDownloading        ProcessingStatus = "downloading"
	ProcessingStatusReadyForProcessing ProcessingStatus = "ready_for_processing"
	ProcessingStatusProcessing         ProcessingStatus = "processing"
	ProcessingStatusCompleted          ProcessingStatus = "completed"
	ProcessingStatusFailed             ProcessingStatus = "failed"
)

type FileChecksums struct {
	ArtistsDump  string `json:"artistsDump"`
	LabelsDump   string `json:"labelsDump"`
	MastersDump  string `json:"mastersDump"`
	ReleasesDump string `json:"releasesDump"`
}

type ProcessingStats struct {
	ArtistsProcessed  int `json:"artistsProcessed"`
	LabelsProcessed   int `json:"labelsProcessed"`
	MastersProcessed  int `json:"mastersProcessed"`
	ReleasesProcessed int `json:"releasesProcessed"`
	TotalRecords      int `json:"totalRecords"`
	FailedRecords     int `json:"failedRecords"`
}

type DiscogsDataProcessing struct {
	BaseUUIDModel
	YearMonth             string           `gorm:"type:text;not null;uniqueIndex:idx_discogs_data_processing_year_month"             json:"yearMonth"                       validate:"required"`
	Status                ProcessingStatus `gorm:"type:text;not null;default:'not_started';index:idx_discogs_data_processing_status" json:"status"`
	FileChecksums         *FileChecksums   `gorm:"type:jsonb"                                                                        json:"fileChecksums,omitempty"`
	ProcessingStats       *ProcessingStats `gorm:"type:jsonb"                                                                        json:"processingStats,omitempty"`
	RetryCount            int              `gorm:"type:int;not null;default:0"                                                       json:"retryCount"`
	ErrorMessage          *string          `gorm:"type:text"                                                                         json:"errorMessage,omitempty"`
	StartedAt             *time.Time       `gorm:"type:timestamp"                                                                    json:"startedAt,omitempty"`
	DownloadCompletedAt   *time.Time       `gorm:"type:timestamp"                                                                    json:"downloadCompletedAt,omitempty"`
	ProcessingCompletedAt *time.Time       `gorm:"type:timestamp"                                                                    json:"processingCompletedAt,omitempty"`
	CompletedAt           *time.Time       `gorm:"type:timestamp"                                                                    json:"completedAt,omitempty"`
}

func (ddp *DiscogsDataProcessing) BeforeCreate(tx *gorm.DB) (err error) {
	if ddp.YearMonth == "" {
		return gorm.ErrInvalidValue
	}
	// Validate YYYY-MM format
	if matched, _ := regexp.MatchString(`^\d{4}-\d{2}$`, ddp.YearMonth); !matched {
		return gorm.ErrInvalidValue
	}
	if ddp.Status == "" {
		ddp.Status = ProcessingStatusNotStarted
	}
	return nil
}

func (ddp *DiscogsDataProcessing) BeforeUpdate(tx *gorm.DB) (err error) {
	if ddp.YearMonth == "" {
		return gorm.ErrInvalidValue
	}
	// Validate YYYY-MM format
	if matched, _ := regexp.MatchString(`^\d{4}-\d{2}$`, ddp.YearMonth); !matched {
		return gorm.ErrInvalidValue
	}
	return nil
}

// CanTransitionTo validates if the current status can transition to the new status
func (ddp *DiscogsDataProcessing) CanTransitionTo(newStatus ProcessingStatus) bool {
	validTransitions := map[ProcessingStatus][]ProcessingStatus{
		ProcessingStatusNotStarted: {
			ProcessingStatusDownloading,
			ProcessingStatusFailed,
		},
		ProcessingStatusDownloading: {
			ProcessingStatusReadyForProcessing,
			ProcessingStatusFailed,
		},
		ProcessingStatusReadyForProcessing: {
			ProcessingStatusProcessing,
			ProcessingStatusFailed,
		},
		ProcessingStatusProcessing: {
			ProcessingStatusCompleted,
			ProcessingStatusFailed,
		},
		ProcessingStatusCompleted: {
			// Completed is terminal, but allow re-processing if needed
			ProcessingStatusDownloading,
		},
		ProcessingStatusFailed: {
			// Allow retry from failed state
			ProcessingStatusDownloading,
			ProcessingStatusReadyForProcessing,
			ProcessingStatusProcessing,
		},
	}

	allowedStates, exists := validTransitions[ddp.Status]
	if !exists {
		return false
	}

	return slices.Contains(allowedStates, newStatus)
}

// UpdateStatus safely updates the status if the transition is valid
func (ddp *DiscogsDataProcessing) UpdateStatus(newStatus ProcessingStatus) error {
	if !ddp.CanTransitionTo(newStatus) {
		return gorm.ErrInvalidValue
	}
	ddp.Status = newStatus
	return nil
}

