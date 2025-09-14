package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
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

type FileDownloadStatus string

const (
	FileDownloadStatusNotStarted  FileDownloadStatus = "not_started"
	FileDownloadStatusDownloading FileDownloadStatus = "downloading"
	FileDownloadStatusCompleted   FileDownloadStatus = "completed"
	FileDownloadStatusValidated   FileDownloadStatus = "validated"
	FileDownloadStatusFailed      FileDownloadStatus = "failed"
)

type FileDownloadInfo struct {
	Status       FileDownloadStatus `json:"status"`
	Downloaded   bool               `json:"downloaded"`
	Validated    bool               `json:"validated"`
	Size         int64              `json:"size,omitempty"`
	DownloadedAt *time.Time         `json:"downloadedAt,omitempty"`
	ValidatedAt  *time.Time         `json:"validatedAt,omitempty"`
	ErrorMessage *string            `json:"errorMessage,omitempty"`
}

type ProcessingStats struct {
	// File-level tracking for resumable downloads
	ArtistsFile  *FileDownloadInfo `json:"artistsFile,omitempty"`
	LabelsFile   *FileDownloadInfo `json:"labelsFile,omitempty"`
	MastersFile  *FileDownloadInfo `json:"mastersFile,omitempty"`
	ReleasesFile *FileDownloadInfo `json:"releasesFile,omitempty"`

	// Processing counters (existing functionality)
	ArtistsProcessed  int `json:"artistsProcessed"`
	LabelsProcessed   int `json:"labelsProcessed"`
	MastersProcessed  int `json:"mastersProcessed"`
	ReleasesProcessed int `json:"releasesProcessed"`
	TotalRecords      int `json:"totalRecords"`
	FailedRecords     int `json:"failedRecords"`
}

// Scan implements the Scanner interface for GORM JSONB support
func (ps *ProcessingStats) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("cannot scan non-byte value into ProcessingStats")
	}

	return json.Unmarshal(bytes, ps)
}

// Value implements the driver Valuer interface for GORM JSONB support
func (ps ProcessingStats) Value() (driver.Value, error) {
	if ps == (ProcessingStats{}) {
		return nil, nil
	}
	return json.Marshal(ps)
}

// GetFileInfo returns the FileDownloadInfo for a specific file type
func (ps *ProcessingStats) GetFileInfo(fileType string) *FileDownloadInfo {
	if ps == nil {
		return nil
	}

	switch fileType {
	case "artists":
		return ps.ArtistsFile
	case "labels":
		return ps.LabelsFile
	case "masters":
		return ps.MastersFile
	case "releases":
		return ps.ReleasesFile
	default:
		return nil
	}
}

// SetFileInfo sets the FileDownloadInfo for a specific file type
func (ps *ProcessingStats) SetFileInfo(fileType string, info *FileDownloadInfo) {
	if ps == nil {
		return
	}

	switch fileType {
	case "artists":
		ps.ArtistsFile = info
	case "labels":
		ps.LabelsFile = info
	case "masters":
		ps.MastersFile = info
	case "releases":
		ps.ReleasesFile = info
	}
}

// InitializeFileInfo ensures a FileDownloadInfo exists for the given file type
func (ps *ProcessingStats) InitializeFileInfo(fileType string) *FileDownloadInfo {
	if ps == nil {
		return nil
	}

	info := ps.GetFileInfo(fileType)
	if info == nil {
		info = &FileDownloadInfo{
			Status:     FileDownloadStatusNotStarted,
			Downloaded: false,
			Validated:  false,
		}
		ps.SetFileInfo(fileType, info)
	}

	return info
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

var yearMonthRegex = regexp.MustCompile(`^\d{4}-\d{2}$`)

func (ddp *DiscogsDataProcessing) BeforeCreate(tx *gorm.DB) (err error) {
	if ddp.YearMonth == "" {
		return gorm.ErrInvalidValue
	}
	if !yearMonthRegex.MatchString(ddp.YearMonth) {
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
	if !yearMonthRegex.MatchString(ddp.YearMonth) {
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

// Scan implements the Scanner interface for GORM JSONB support
func (fc *FileChecksums) Scan(value any) error {
	if value == nil {
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("cannot scan non-byte value into FileChecksums")
	}

	return json.Unmarshal(bytes, fc)
}

// Value implements the driver Valuer interface for GORM JSONB support
func (fc FileChecksums) Value() (driver.Value, error) {
	if fc == (FileChecksums{}) {
		return nil, nil
	}
	return json.Marshal(fc)
}
