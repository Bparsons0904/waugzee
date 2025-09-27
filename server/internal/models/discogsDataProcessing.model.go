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

// ProcessingStep represents an individual processing step
type ProcessingStep string

const (
	StepLabelsProcessing              ProcessingStep = "labels_processing"
	StepArtistsProcessing             ProcessingStep = "artists_processing"
	StepMastersProcessing             ProcessingStep = "masters_processing"
	StepReleasesProcessing            ProcessingStep = "releases_processing"
	StepMasterGenresCollection        ProcessingStep = "master_genres_collection"
	StepMasterGenresUpsert            ProcessingStep = "master_genres_upsert"
	StepMasterGenreAssociations       ProcessingStep = "master_genre_associations"
	StepReleaseGenresCollection       ProcessingStep = "release_genres_collection"
	StepReleaseGenresUpsert           ProcessingStep = "release_genres_upsert"
	StepReleaseGenreAssociations      ProcessingStep = "release_genre_associations"
	StepReleaseLabelAssociations      ProcessingStep = "release_label_associations"
	StepMasterArtistAssociations      ProcessingStep = "master_artist_associations"
	StepReleaseArtistAssociations     ProcessingStep = "release_artist_associations"
)

// StepStatus represents the completion status of a processing step
type StepStatus struct {
	Completed     bool       `json:"completed"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	ErrorMessage  *string    `json:"error_message,omitempty"`
	RecordsCount  *int64     `json:"records_count,omitempty"`
	Duration      *string    `json:"duration,omitempty"`
}

// ProcessingStats tracks detailed information about file processing
type ProcessingStats struct {
	ArtistsFile  *FileDownloadInfo `json:"artists_file,omitempty"`
	LabelsFile   *FileDownloadInfo `json:"labels_file,omitempty"`
	MastersFile  *FileDownloadInfo `json:"masters_file,omitempty"`
	ReleasesFile *FileDownloadInfo `json:"releases_file,omitempty"`

	// Individual processing steps tracking
	ProcessingSteps map[ProcessingStep]*StepStatus `json:"processing_steps,omitempty"`
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

// InitializeProcessingSteps initializes the processing steps map if nil
func (d *DiscogsDataProcessing) InitializeProcessingSteps() {
	d.InitializeProcessingStats()
	if d.ProcessingStats.ProcessingSteps == nil {
		d.ProcessingStats.ProcessingSteps = make(map[ProcessingStep]*StepStatus)
	}
}

// IsStepCompleted returns true if the specified step has been completed
func (d *DiscogsDataProcessing) IsStepCompleted(step ProcessingStep) bool {
	if d.ProcessingStats == nil || d.ProcessingStats.ProcessingSteps == nil {
		return false
	}

	stepStatus, exists := d.ProcessingStats.ProcessingSteps[step]
	return exists && stepStatus.Completed
}

// MarkStepCompleted marks a processing step as completed with optional metadata
func (d *DiscogsDataProcessing) MarkStepCompleted(step ProcessingStep, recordsCount *int64, duration *string) {
	d.InitializeProcessingSteps()

	now := time.Now().UTC()
	d.ProcessingStats.ProcessingSteps[step] = &StepStatus{
		Completed:    true,
		CompletedAt:  &now,
		RecordsCount: recordsCount,
		Duration:     duration,
	}
}

// MarkStepFailed marks a processing step as failed with error message
func (d *DiscogsDataProcessing) MarkStepFailed(step ProcessingStep, errorMessage string) {
	d.InitializeProcessingSteps()

	d.ProcessingStats.ProcessingSteps[step] = &StepStatus{
		Completed:    false,
		ErrorMessage: &errorMessage,
	}
}

// GetCompletedSteps returns a list of completed processing steps
func (d *DiscogsDataProcessing) GetCompletedSteps() []ProcessingStep {
	if d.ProcessingStats == nil || d.ProcessingStats.ProcessingSteps == nil {
		return []ProcessingStep{}
	}

	var completed []ProcessingStep
	for step, status := range d.ProcessingStats.ProcessingSteps {
		if status.Completed {
			completed = append(completed, step)
		}
	}

	return completed
}

// GetStepStatus returns the status of a specific processing step
func (d *DiscogsDataProcessing) GetStepStatus(step ProcessingStep) *StepStatus {
	if d.ProcessingStats == nil || d.ProcessingStats.ProcessingSteps == nil {
		return nil
	}

	return d.ProcessingStats.ProcessingSteps[step]
}

// AllStepsCompleted returns true if all processing steps have been completed
func (d *DiscogsDataProcessing) AllStepsCompleted() bool {
	allSteps := []ProcessingStep{
		StepLabelsProcessing,
		StepArtistsProcessing,
		StepMastersProcessing,
		StepReleasesProcessing,
		StepMasterGenresCollection,
		StepMasterGenresUpsert,
		StepMasterGenreAssociations,
		StepReleaseGenresCollection,
		StepReleaseGenresUpsert,
		StepReleaseGenreAssociations,
		StepReleaseLabelAssociations,
		StepMasterArtistAssociations,
		StepReleaseArtistAssociations,
	}

	for _, step := range allSteps {
		if !d.IsStepCompleted(step) {
			return false
		}
	}

	return true
}

// TableName specifies the table name for GORM
func (DiscogsDataProcessing) TableName() string {
	return "discogs_data_processing"
}
