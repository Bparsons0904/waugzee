package admin

import (
	"context"
	"fmt"
	"time"
	"waugzee/internal/constants"
	"waugzee/internal/logger"
	"waugzee/internal/models"
	"waugzee/internal/repositories"
	"waugzee/internal/services"

	"gorm.io/gorm"
)

type AdminControllerInterface interface {
	GetDownloadStatus(ctx context.Context) (*DownloadStatusResponse, error)
	TriggerDownload(ctx context.Context) error
	TriggerReprocess(ctx context.Context) error
	ResetStuckDownload(ctx context.Context) error
}

type AdminController struct {
	db                   *gorm.DB
	processingRepo       repositories.DiscogsDataProcessingRepository
	downloadService      *services.DownloadService
	xmlProcessingService *services.DiscogsXMLParserService
	schedulerService     *services.SchedulerService
	log                  logger.Logger
}

func NewAdminController(
	db *gorm.DB,
	processingRepo repositories.DiscogsDataProcessingRepository,
	downloadService *services.DownloadService,
	xmlProcessingService *services.DiscogsXMLParserService,
	schedulerService *services.SchedulerService,
) AdminControllerInterface {
	return &AdminController{
		db:                   db,
		processingRepo:       processingRepo,
		downloadService:      downloadService,
		xmlProcessingService: xmlProcessingService,
		schedulerService:     schedulerService,
		log:                  logger.New("adminController"),
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

func (c *AdminController) GetDownloadStatus(ctx context.Context) (*DownloadStatusResponse, error) {
	log := c.log.Function("GetDownloadStatus")

	record, err := c.processingRepo.GetLatestProcessing(ctx)
	if err != nil {
		return nil, log.Err("failed to get latest processing record", err)
	}

	return c.modelToResponse(record), nil
}

func (c *AdminController) modelToResponse(
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

func (c *AdminController) TriggerDownload(ctx context.Context) error {
	log := c.log.Function("TriggerDownload")

	yearMonth := time.Now().Format("2006-01")

	existingRecord, err := c.processingRepo.GetByYearMonth(ctx, yearMonth)
	if err != nil && err != gorm.ErrRecordNotFound {
		return log.Err("failed to check existing record", err)
	}

	if existingRecord != nil {
		switch existingRecord.Status {
		case models.ProcessingStatusDownloading, models.ProcessingStatusProcessing:
			return log.Err("download or processing already in progress",
				fmt.Errorf("download or processing already in progress"))

		case models.ProcessingStatusCompleted,
			models.ProcessingStatusFailed,
			models.ProcessingStatusReadyForProcessing:
			existingRecord.Status = models.ProcessingStatusNotStarted
			existingRecord.StartedAt = nil
			existingRecord.DownloadCompletedAt = nil
			existingRecord.ProcessingCompletedAt = nil
			existingRecord.ErrorMessage = nil
			existingRecord.ProcessingStats = nil
			existingRecord.FileChecksums = nil
			existingRecord.RetryCount = 0

			if err := c.processingRepo.Update(ctx, existingRecord); err != nil {
				return log.Err("failed to reset processing record", err)
			}

			log.Info("Reset existing processing record", "yearMonth", yearMonth)
		}
	} else {
		newRecord := &models.DiscogsDataProcessing{
			YearMonth:  yearMonth,
			Status:     models.ProcessingStatusNotStarted,
			RetryCount: 0,
		}

		if _, err := c.processingRepo.Create(ctx, newRecord); err != nil {
			return log.Err("failed to create processing record", err)
		}

		log.Info("Created new processing record", "yearMonth", yearMonth)
	}

	if err := c.schedulerService.TriggerJobByName(ctx, constants.JobDiscogsDownload); err != nil {
		return log.Err("failed to trigger download job", err)
	}

	return nil
}

func (c *AdminController) TriggerReprocess(ctx context.Context) error {
	log := c.log.Function("TriggerReprocess")

	record, err := c.processingRepo.GetLatestProcessing(ctx)
	if err != nil {
		return log.Err("failed to get latest processing record", err)
	}

	if record == nil {
		return log.Err("no processing record found", fmt.Errorf("no processing record found"))
	}

	statusValid := record.Status == models.ProcessingStatusReadyForProcessing ||
		record.Status == models.ProcessingStatusProcessing ||
		record.Status == models.ProcessingStatusCompleted ||
		record.Status == models.ProcessingStatusFailed

	if !statusValid {
		return log.Err(
			"files not downloaded",
			fmt.Errorf("files must be downloaded before reprocessing"),
		)
	}

	record.InitializeProcessingStats()
	record.ProcessingStats.ProcessingSteps = make(map[models.ProcessingStep]*models.StepStatus)
	record.Status = models.ProcessingStatusReadyForProcessing
	record.ProcessingCompletedAt = nil
	record.ErrorMessage = nil

	if err := c.processingRepo.Update(ctx, record); err != nil {
		return log.Err("failed to update processing record", err)
	}

	if err := c.schedulerService.TriggerJobByName(ctx, constants.JobDiscogsXMLParser); err != nil {
		return log.Err("failed to trigger xml parser job", err)
	}

	return nil
}

func (c *AdminController) ResetStuckDownload(ctx context.Context) error {
	log := c.log.Function("ResetStuckDownload")

	yearMonth := time.Now().Format("2006-01")

	record, err := c.processingRepo.GetByYearMonth(ctx, yearMonth)
	if err != nil {
		return log.Err("failed to get processing record", err, "yearMonth", yearMonth)
	}

	if record == nil {
		return log.Err("no processing record found", fmt.Errorf("no record for %s", yearMonth))
	}

	if record.Status != models.ProcessingStatusDownloading &&
		record.Status != models.ProcessingStatusProcessing &&
		record.Status != models.ProcessingStatusFailed {
		return log.Err(
			"cannot reset record in this state",
			fmt.Errorf("current status: %s", record.Status),
		)
	}

	if err := c.downloadService.CleanupDownloadDirectory(ctx, yearMonth); err != nil {
		return log.Err("failed to cleanup files", err)
	}

	record.Status = models.ProcessingStatusNotStarted
	record.StartedAt = nil
	record.DownloadCompletedAt = nil
	record.ProcessingCompletedAt = nil
	record.ErrorMessage = nil
	record.ProcessingStats = nil
	record.FileChecksums = nil
	record.RetryCount = 0

	if err := c.processingRepo.Update(ctx, record); err != nil {
		return log.Err("failed to reset processing record", err)
	}

	log.Info("Successfully reset stuck download", "yearMonth", yearMonth)
	return nil
}
