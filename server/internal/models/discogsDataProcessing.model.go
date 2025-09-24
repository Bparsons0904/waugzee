package models

import (
	"time"
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

// FileDownloadStatus represents the current status of a file download
type FileDownloadStatus string

const (
	FileDownloadStatusNotStarted  FileDownloadStatus = "not_started"
	FileDownloadStatusDownloading FileDownloadStatus = "downloading"
	FileDownloadStatusFailed      FileDownloadStatus = "failed"
	FileDownloadStatusValidated   FileDownloadStatus = "validated"
)

// FileDownloadInfo contains information about a file download
type FileDownloadInfo struct {
	Status       FileDownloadStatus `json:"status"`
	Downloaded   bool               `json:"downloaded"`
	Validated    bool               `json:"validated"`
	Size         int64              `json:"size"`
	DownloadedAt *time.Time         `json:"downloaded_at,omitempty"`
	ValidatedAt  *time.Time         `json:"validated_at,omitempty"`
	ErrorMessage *string            `json:"error_message,omitempty"`
}

// DiscogsDataProcessing tracks the state of Discogs data dump processing
type DiscogsDataProcessing struct {
	BaseDiscogModel

	// Processing identification
	YearMonth string `gorm:"uniqueIndex;not null" json:"year_month"`

	// Status tracking
	Status       ProcessingStatus `gorm:"not null;default:'not_started'" json:"status"`
	RetryCount   int              `gorm:"default:0"                      json:"retry_count"`
	ErrorMessage *string          `                                      json:"error_message,omitempty"`

	// Timing information
	StartedAt             *time.Time `json:"started_at,omitempty"`
	DownloadCompletedAt   *time.Time `json:"download_completed_at,omitempty"`
	ProcessingCompletedAt *time.Time `json:"processing_completed_at,omitempty"`

	// File checksums from CHECKSUM.txt
	FileChecksums *FileChecksums `gorm:"serializer:json" json:"file_checksums,omitempty"`

	// Processing statistics and file information
	ProcessingStats *ProcessingStats `gorm:"serializer:json" json:"processing_stats,omitempty"`
}

// FileChecksums represents the checksums for each Discogs data dump file
type FileChecksums struct {
	ArtistsDump  string `json:"artists_dump,omitempty"`
	LabelsDump   string `json:"labels_dump,omitempty"`
	MastersDump  string `json:"masters_dump,omitempty"`
	ReleasesDump string `json:"releases_dump,omitempty"`
}

// ProcessingStats tracks detailed information about file processing
type ProcessingStats struct {
	ArtistsFile  *FileDownloadInfo `json:"artists_file,omitempty"`
	LabelsFile   *FileDownloadInfo `json:"labels_file,omitempty"`
	MastersFile  *FileDownloadInfo `json:"masters_file,omitempty"`
	ReleasesFile *FileDownloadInfo `json:"releases_file,omitempty"`
}

// UpdateStatus updates the processing status with validation
func (d *DiscogsDataProcessing) UpdateStatus(newStatus ProcessingStatus) error {
	// Add status transition validation if needed
	d.Status = newStatus
	return nil
}

// IsReadyForProcessing returns true if files are downloaded and validated
func (d *DiscogsDataProcessing) IsReadyForProcessing() bool {
	return d.Status == ProcessingStatusReadyForProcessing &&
		d.FileChecksums != nil &&
		d.DownloadCompletedAt != nil
}

// GetFileChecksum returns the checksum for a specific file type
func (d *DiscogsDataProcessing) GetFileChecksum(fileType string) string {
	if d.FileChecksums == nil {
		return ""
	}

	switch fileType {
	case "artists":
		return d.FileChecksums.ArtistsDump
	case "labels":
		return d.FileChecksums.LabelsDump
	case "masters":
		return d.FileChecksums.MastersDump
	case "releases":
		return d.FileChecksums.ReleasesDump
	default:
		return ""
	}
}

// HasFileChecksum returns true if checksum exists for the given file type
func (d *DiscogsDataProcessing) HasFileChecksum(fileType string) bool {
	return d.GetFileChecksum(fileType) != ""
}

// GetAvailableFileTypes returns list of file types that have checksums
func (d *DiscogsDataProcessing) GetAvailableFileTypes() []string {
	if d.FileChecksums == nil {
		return []string{}
	}

	var types []string
	if d.FileChecksums.ArtistsDump != "" {
		types = append(types, "artists")
	}
	if d.FileChecksums.LabelsDump != "" {
		types = append(types, "labels")
	}
	if d.FileChecksums.MastersDump != "" {
		types = append(types, "masters")
	}
	if d.FileChecksums.ReleasesDump != "" {
		types = append(types, "releases")
	}

	return types
}

// InitializeProcessingStats initializes the processing stats if nil
func (d *DiscogsDataProcessing) InitializeProcessingStats() {
	if d.ProcessingStats == nil {
		d.ProcessingStats = &ProcessingStats{}
	}
}

// GetFileInfo returns file download info for a specific file type
func (d *DiscogsDataProcessing) GetFileInfo(fileType string) *FileDownloadInfo {
	if d.ProcessingStats == nil {
		return nil
	}

	switch fileType {
	case "artists":
		return d.ProcessingStats.ArtistsFile
	case "labels":
		return d.ProcessingStats.LabelsFile
	case "masters":
		return d.ProcessingStats.MastersFile
	case "releases":
		return d.ProcessingStats.ReleasesFile
	default:
		return nil
	}
}

// SetFileInfo sets file download info for a specific file type
func (d *DiscogsDataProcessing) SetFileInfo(fileType string, info *FileDownloadInfo) {
	d.InitializeProcessingStats()

	switch fileType {
	case "artists":
		d.ProcessingStats.ArtistsFile = info
	case "labels":
		d.ProcessingStats.LabelsFile = info
	case "masters":
		d.ProcessingStats.MastersFile = info
	case "releases":
		d.ProcessingStats.ReleasesFile = info
	}
}

// InitializeFileInfo initializes file info for a specific file type if it doesn't exist
func (d *DiscogsDataProcessing) InitializeFileInfo(fileType string) {
	if d.GetFileInfo(fileType) == nil {
		d.SetFileInfo(fileType, &FileDownloadInfo{
			Status:     FileDownloadStatusNotStarted,
			Downloaded: false,
			Validated:  false,
		})
	}
}

// TableName specifies the table name for GORM
func (DiscogsDataProcessing) TableName() string {
	return "discogs_data_processing"
}
