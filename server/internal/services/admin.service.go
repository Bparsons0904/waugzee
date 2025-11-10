package services

import (
	"context"
	"fmt"
	"slices"
	"time"
	"waugzee/internal/database"
	"waugzee/internal/logger"
	"waugzee/internal/models"
	"waugzee/internal/repositories"

	"gorm.io/gorm"
)

type JobScheduler interface {
	TriggerJobByName(ctx context.Context, jobName string) error
}

type AdminServiceInterface interface {
	GetDownloadStatus(ctx context.Context) (*DownloadStatusResponse, error)
	TriggerDownload(ctx context.Context) error
	TriggerReprocess(ctx context.Context) error
}

type AdminService struct {
	db               *gorm.DB
	processingRepo   repositories.DiscogsDataProcessingRepository
	downloadService  *DownloadService
	xmlParserService *DiscogsXMLParserService
	schedulerService JobScheduler
	log              logger.Logger
}

func NewAdminService(
	db database.DB,
	processingRepo repositories.DiscogsDataProcessingRepository,
	downloadService *DownloadService,
	xmlParserService *DiscogsXMLParserService,
	schedulerService JobScheduler,
) *AdminService {
	return &AdminService{
		db:               db.SQL,
		processingRepo:   processingRepo,
		downloadService:  downloadService,
		xmlParserService: xmlParserService,
		schedulerService: schedulerService,
		log:              logger.New("services").File("admin_service"),
	}
}

type DownloadStatusResponse struct {
	YearMonth             string                                       `json:"year_month"`
	Status                models.ProcessingStatus                      `json:"status"`
	StartedAt             *time.Time                                   `json:"started_at,omitempty"`
	DownloadCompletedAt   *time.Time                                   `json:"download_completed_at,omitempty"`
	ProcessingCompletedAt *time.Time                                   `json:"processing_completed_at,omitempty"`
	FileChecksums         *models.FileChecksums                        `json:"file_checksums,omitempty"`
	Files                 *FileStatusInfo                              `json:"files,omitempty"`
	ProcessingSteps       map[models.ProcessingStep]*models.StepStatus `json:"processing_steps,omitempty"`
	RetryCount            int                                          `json:"retry_count"`
	ErrorMessage          *string                                      `json:"error_message,omitempty"`
}

type FileStatusInfo struct {
	Artists  *models.FileDownloadInfo `json:"artists,omitempty"`
	Labels   *models.FileDownloadInfo `json:"labels,omitempty"`
	Masters  *models.FileDownloadInfo `json:"masters,omitempty"`
	Releases *models.FileDownloadInfo `json:"releases,omitempty"`
}

func (s *AdminService) modelToResponse(
	record *models.DiscogsDataProcessing,
) *DownloadStatusResponse {
	if record == nil {
		return nil
	}

	resp := &DownloadStatusResponse{
		YearMonth:             record.YearMonth,
		Status:                record.Status,
		StartedAt:             record.StartedAt,
		DownloadCompletedAt:   record.DownloadCompletedAt,
		ProcessingCompletedAt: record.ProcessingCompletedAt,
		FileChecksums:         record.FileChecksums,
		RetryCount:            record.RetryCount,
		ErrorMessage:          record.ErrorMessage,
	}

	if record.ProcessingStats != nil {
		resp.Files = &FileStatusInfo{
			Artists:  record.ProcessingStats.ArtistsFile,
			Labels:   record.ProcessingStats.LabelsFile,
			Masters:  record.ProcessingStats.MastersFile,
			Releases: record.ProcessingStats.ReleasesFile,
		}
		resp.ProcessingSteps = record.ProcessingStats.ProcessingSteps
	}

	return resp
}

// Claude this entire file is fundamentally wrong. It should be a controller and not a service.
// Handler -> Controller -> Service/Repository

func (s *AdminService) GetDownloadStatus(ctx context.Context) (*DownloadStatusResponse, error) {
	log := s.log.Function("GetDownloadStatus")

	record, err := s.processingRepo.GetLatestProcessing(ctx)
	if err != nil {
		return nil, log.Err("failed to get latest processing record", err)
	}

	return s.modelToResponse(record), nil
}

func (s *AdminService) TriggerDownload(ctx context.Context) error {
	log := s.log.Function("TriggerDownload")

	currentTime := time.Now().UTC()
	yearMonth := currentTime.Format("2006-01")

	existingRecord, err := s.processingRepo.GetByYearMonth(ctx, yearMonth)
	// Claude Is it really an error if there is no existing record? Lets say the files drop during the day and I want to manaully
	// start it and not wait for the morning for it to start?  We should be able to start it.
	if err != nil && err != gorm.ErrRecordNotFound {
		return log.Err("failed to check for existing processing record", err)
	}

	// Claude this should be a switch
	if existingRecord != nil {
		if existingRecord.Status == models.ProcessingStatusDownloading ||
			existingRecord.Status == models.ProcessingStatusProcessing {
			// We have logger options for this, we should never use fmt
			return fmt.Errorf("Download or processing already in progress")
		}

		if existingRecord.Status == models.ProcessingStatusCompleted ||
			existingRecord.Status == models.ProcessingStatusFailed {
			if err := s.processingRepo.Delete(ctx, existingRecord.ID); err != nil {
				return log.Err("failed to delete old processing record", err)
			}
			log.Info(
				"Deleted old processing record",
				"yearMonth",
				yearMonth,
				"status",
				existingRecord.Status,
			)
		}
	}

	// Claude Do we want to have more than 1 processing record per month? Pretty sure we have unique yearMonth. so this would never work.
	// I feel like just reseting the existing record is fine, maybe we add to the data layer about that it was a manual run/rerun?
	newRecord := &models.DiscogsDataProcessing{
		YearMonth: yearMonth,
		Status:    models.ProcessingStatusNotStarted,
	}

	if _, err := s.processingRepo.Create(ctx, newRecord); err != nil {
		return log.Err("failed to create processing record", err)
	}

	log.Info("Created new processing record", "yearMonth", yearMonth)

	// Claude should DiscogsDownloadJob be a const or possibly an enum?
	if err := s.schedulerService.TriggerJobByName(ctx, "DiscogsDownloadJob"); err != nil {
		return log.Err("failed to trigger download job", err)
	}

	log.Info("Triggered download job successfully")
	return nil
}

func (s *AdminService) TriggerReprocess(ctx context.Context) error {
	log := s.log.Function("TriggerReprocess")

	record, err := s.processingRepo.GetLatestProcessing(ctx)
	if err != nil {
		return log.Err("failed to get latest processing record", err)
	}

	if record == nil {
		// Claude we should be using the logger package for all logging of errors
		return fmt.Errorf("No processing record found")
	}

	validStatuses := []models.ProcessingStatus{
		models.ProcessingStatusReadyForProcessing,
		models.ProcessingStatusProcessing,
		models.ProcessingStatusCompleted,
		models.ProcessingStatusFailed,
	}

	statusValid := slices.Contains(validStatuses, record.Status)

	if !statusValid {
		// Claude we should be using the logger package for all logging of errors
		return fmt.Errorf("Files must be downloaded before reprocessing")
	}

	record.InitializeProcessingStats()
	record.ProcessingStats.ProcessingSteps = make(map[models.ProcessingStep]*models.StepStatus)
	record.Status = models.ProcessingStatusReadyForProcessing

	if err := s.processingRepo.Update(ctx, record); err != nil {
		return log.Err("failed to update processing record", err)
	}

	log.Info("Reset processing record for reprocessing", "yearMonth", record.YearMonth)

	if err := s.schedulerService.TriggerJobByName(ctx, "DiscogsXMLParserJob"); err != nil {
		return log.Err("failed to trigger XML parser job", err)
	}

	log.Info("Triggered XML parser job successfully")
	return nil
}
