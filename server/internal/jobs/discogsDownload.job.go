package jobs

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
	"waugzee/internal/logger"
	"waugzee/internal/models"
	"waugzee/internal/repositories"
	"waugzee/internal/services"

	"gorm.io/gorm"
)

type DiscogsDownloadJob struct {
	repo        repositories.DiscogsDataProcessingRepository
	transaction *services.TransactionService
	download    *services.DownloadService
	log         logger.Logger
	schedule    services.Schedule
}

func NewDiscogsDownloadJob(
	repo repositories.DiscogsDataProcessingRepository,
	transaction *services.TransactionService,
	download *services.DownloadService,
	schedule services.Schedule,
) *DiscogsDownloadJob {
	return &DiscogsDownloadJob{
		repo:        repo,
		transaction: transaction,
		download:    download,
		log:         logger.New("discogsDownloadJob"),
		schedule:    schedule,
	}
}

func (j *DiscogsDownloadJob) Name() string {
	return "DiscogsDailyDownloadCheck"
}

func (j *DiscogsDownloadJob) Execute(ctx context.Context) error {
	log := j.log.Function("Execute")

	// Get current year-month in YYYY-MM format
	now := time.Now().UTC()
	yearMonth := now.Format("2006-01")

	log.Info("Starting Discogs data processing check", "yearMonth", yearMonth)

	// Use transaction for all database operations
	return j.transaction.Execute(ctx, func(txCtx context.Context) error {
		// Check if there's already a processing record for this month
		existing, err := j.repo.GetByYearMonth(txCtx, yearMonth)
		if err != nil {
			// Record not found is expected for new months
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return log.Err(
					"failed to check existing processing record",
					err,
					"yearMonth",
					yearMonth,
				)
			}
		}

		// If record exists and is not in a failed state, skip processing
		if existing != nil {
			switch existing.Status {
			case models.ProcessingStatusCompleted:
				log.Info(
					"Data processing already completed for this month",
					"yearMonth",
					yearMonth,
					"status",
					existing.Status,
				)
				return nil
			case models.ProcessingStatusDownloading,
				models.ProcessingStatusReadyForProcessing,
				models.ProcessingStatusProcessing:
				log.Info(
					"Data processing already in progress for this month",
					"yearMonth",
					yearMonth,
					"status",
					existing.Status,
				)
				return nil
			case models.ProcessingStatusFailed:
				log.Info(
					"Found failed processing record, will retry",
					"yearMonth",
					yearMonth,
					"retryCount",
					existing.RetryCount,
				)
				// Continue to retry logic below
			case models.ProcessingStatusNotStarted:
				log.Info("Found not started record, continuing processing", "yearMonth", yearMonth)
				// Continue to processing logic below
			}
		}

		// Create or update processing record
		var processingRecord *models.DiscogsDataProcessing
		if existing == nil {
			// Create new processing record
			processing := &models.DiscogsDataProcessing{
				YearMonth:    yearMonth,
				Status:       models.ProcessingStatusNotStarted,
				RetryCount:   0,
				ErrorMessage: nil,
			}

			created, err := j.repo.Create(txCtx, processing)
			if err != nil {
				return log.Err("failed to create processing record", err, "yearMonth", yearMonth)
			}

			processingRecord = created
			log.Info("Created new processing record", "yearMonth", yearMonth, "id", created.ID)
		} else {
			processingRecord = existing
			// Update existing record for retry if in failed state
			if existing.Status == models.ProcessingStatusFailed {
				// Increment retry count and clear error message
				existing.RetryCount++
				existing.ErrorMessage = nil
				// Set started time for the retry
				now := time.Now().UTC()
				existing.StartedAt = &now

				if err := j.repo.Update(txCtx, existing); err != nil {
					return log.Err("failed to update processing record for retry", err, "yearMonth", yearMonth)
				}

				log.Info("Updated processing record for retry", "yearMonth", yearMonth, "retryCount", existing.RetryCount)
			}
		}

		// Only proceed with download if status is not_started or failed
		if processingRecord.Status != models.ProcessingStatusNotStarted &&
			processingRecord.Status != models.ProcessingStatusFailed {
			log.Info(
				"Processing record not in downloadable state",
				"status",
				processingRecord.Status,
			)
			return nil
		}

		// Transition to downloading status
		if err := processingRecord.UpdateStatus(models.ProcessingStatusDownloading); err != nil {
			return log.Err(
				"failed to transition to downloading status",
				err,
				"yearMonth",
				yearMonth,
			)
		}

		// Set started time if not already set
		if processingRecord.StartedAt == nil {
			now := time.Now().UTC()
			processingRecord.StartedAt = &now
		}

		if err := j.repo.Update(txCtx, processingRecord); err != nil {
			return log.Err(
				"failed to update processing record to downloading",
				err,
				"yearMonth",
				yearMonth,
			)
		}

		log.Info(
			"Transitioned to downloading status",
			"yearMonth",
			yearMonth,
			"status",
			processingRecord.Status,
		)

		// Perform the actual download
		if err := j.performDownload(txCtx, processingRecord, yearMonth); err != nil {
			// Update record with error information
			errorMsg := err.Error()
			processingRecord.ErrorMessage = &errorMsg
			if statusErr := processingRecord.UpdateStatus(models.ProcessingStatusFailed); statusErr != nil {
				log.Warn("failed to update processing record status to failed", "error", statusErr)
			}

			if updateErr := j.repo.Update(txCtx, processingRecord); updateErr != nil {
				log.Warn("failed to update processing record with error", "error", updateErr)
			}

			return log.Err("download failed", err, "yearMonth", yearMonth)
		}

		log.Info("Discogs data processing check completed successfully", "yearMonth", yearMonth)
		return nil
	})
}

// performDownload handles the actual download process with recovery support
func (j *DiscogsDownloadJob) performDownload(
	ctx context.Context,
	processingRecord *models.DiscogsDataProcessing,
	yearMonth string,
) error {
	log := j.log.Function("performDownload")

	// Initialize ProcessingStats if not present
	if processingRecord.ProcessingStats == nil {
		processingRecord.ProcessingStats = &models.ProcessingStats{}
	}

	downloadDir := fmt.Sprintf("/app/discogs-data/%s", yearMonth)

	// Step 1: Handle checksum file
	if err := j.handleChecksumFile(ctx, processingRecord, yearMonth); err != nil {
		return err
	}

	// Step 2: Check and recover existing files or download new ones
	if err := j.handleFileDownloads(ctx, processingRecord, yearMonth, downloadDir); err != nil {
		return err
	}

	// All downloads and validations completed successfully
	// Transition to ready_for_processing status
	if err := processingRecord.UpdateStatus(models.ProcessingStatusReadyForProcessing); err != nil {
		return log.Err(
			"failed to transition to ready_for_processing status",
			err,
			"yearMonth",
			yearMonth,
		)
	}

	// Set download completed time
	now := time.Now().UTC()
	processingRecord.DownloadCompletedAt = &now

	if err := j.repo.Update(ctx, processingRecord); err != nil {
		return log.Err(
			"failed to update processing record after download",
			err,
			"yearMonth",
			yearMonth,
		)
	}

	log.Info("Downloads completed successfully, ready for processing",
		"yearMonth", yearMonth,
		"status", processingRecord.Status)

	log.Info("Download workflow completed successfully",
		"yearMonth", yearMonth,
		"status", processingRecord.Status)

	return nil
}

// handleChecksumFile manages checksum file download and parsing
func (j *DiscogsDownloadJob) handleChecksumFile(
	ctx context.Context,
	processingRecord *models.DiscogsDataProcessing,
	yearMonth string,
) error {
	log := j.log.Function("handleChecksumFile")

	// Only download checksum if we don't already have it
	if processingRecord.FileChecksums == nil {
		log.Info("Downloading checksum file", "yearMonth", yearMonth)

		// Download the CHECKSUM.txt file using the download service
		if err := j.download.DownloadChecksum(ctx, yearMonth); err != nil {
			return log.Err("failed to download checksum file", err, "yearMonth", yearMonth)
		}

		// Parse the downloaded checksum file
		checksumFile := filepath.Join(
			fmt.Sprintf("/app/discogs-data/%s", yearMonth),
			"CHECKSUM.txt",
		)
		checksums, err := j.download.ParseChecksumFile(checksumFile)
		if err != nil {
			return log.Err("failed to parse checksum file", err, "checksumFile", checksumFile)
		}

		// Update processing record with checksums
		processingRecord.FileChecksums = checksums

		// Save checksums to database
		if err := j.repo.Update(ctx, processingRecord); err != nil {
			return log.Err(
				"failed to update processing record with checksums",
				err,
				"yearMonth",
				yearMonth,
			)
		}

		// Clean up downloaded checksum file to save space
		if err := os.Remove(checksumFile); err != nil {
			log.Warn("failed to clean up checksum file", "error", err, "file", checksumFile)
		}

		log.Info("Checksums parsed and saved successfully", "yearMonth", yearMonth)
	} else {
		log.Info("Using existing checksums from database", "yearMonth", yearMonth)
	}

	return nil
}

// handleFileDownloads manages individual file downloads with recovery support
func (j *DiscogsDownloadJob) handleFileDownloads(
	ctx context.Context,
	processingRecord *models.DiscogsDataProcessing,
	yearMonth string,
	downloadDir string,
) error {
	log := j.log.Function("handleFileDownloads")

	checksums := processingRecord.FileChecksums
	if checksums == nil {
		return log.Err(
			"checksums not available",
			fmt.Errorf("FileChecksums is nil"),
			"yearMonth",
			yearMonth,
		)
	}

	fileTypes := []struct {
		name     string
		checksum string
	}{
		{"artists", checksums.ArtistsDump},
		{"labels", checksums.LabelsDump},
		{"masters", checksums.MastersDump},
		{"releases", checksums.ReleasesDump},
	}

	// Use limited concurrent downloads with semaphore pattern
	const maxConcurrentDownloads = 3
	semaphore := make(chan struct{}, maxConcurrentDownloads)

	var wg sync.WaitGroup
	var mu sync.Mutex
	var downloadErrors []error

	// Launch concurrent downloads for all files with concurrency limit
	for _, ft := range fileTypes {
		if ft.checksum == "" {
			log.Info("Skipping file (no checksum available)", "fileType", ft.name)
			continue
		}

		wg.Add(1)
		go func(fileType, checksum string) {
			defer wg.Done()

			// Acquire semaphore slot
			semaphore <- struct{}{}
			defer func() { <-semaphore }() // Release semaphore slot

			if err := j.handleSingleFileDownload(ctx, processingRecord, yearMonth, downloadDir, fileType, checksum); err != nil {
				mu.Lock()
				downloadErrors = append(
					downloadErrors,
					fmt.Errorf("failed to handle file download for %s: %w", fileType, err),
				)
				mu.Unlock()
				_ = log.Err(
					"concurrent download failed",
					err,
					"fileType",
					fileType,
					"yearMonth",
					yearMonth,
				)
			}
		}(ft.name, ft.checksum)
	}

	// Wait for all downloads to complete
	wg.Wait()

	// Check if any downloads failed
	if len(downloadErrors) > 0 {
		return log.Err(
			"one or more concurrent downloads failed",
			fmt.Errorf("download failures: %v", downloadErrors),
			"yearMonth",
			yearMonth,
		)
	}

	return nil
}

// handleSingleFileDownload handles download/recovery for a single file
func (j *DiscogsDownloadJob) handleSingleFileDownload(
	ctx context.Context,
	processingRecord *models.DiscogsDataProcessing,
	yearMonth string,
	downloadDir string,
	fileType string,
	expectedChecksum string,
) error {
	log := j.log.Function("handleSingleFileDownload")

	filePath := filepath.Join(downloadDir, fmt.Sprintf("%s.xml.gz", fileType))

	// Initialize file tracking info
	processingRecord.ProcessingStats.InitializeFileInfo(fileType)
	fileInfo := processingRecord.ProcessingStats.GetFileInfo(fileType)

	// Check if file already exists and is validated
	if fileInfo.Status == models.FileDownloadStatusValidated {
		log.Info(
			"File already validated, skipping download",
			"fileType",
			fileType,
			"filePath",
			filePath,
		)
		return nil
	}

	// Check existing file status
	currentStatus, err := j.download.GetFileStatus(filePath, expectedChecksum)
	if err != nil {
		return log.Err(
			"failed to check file status",
			err,
			"fileType",
			fileType,
			"filePath",
			filePath,
		)
	}

	// If file exists and is validated, update our tracking and skip download
	if currentStatus.Status == models.FileDownloadStatusValidated {
		log.Info(
			"Found existing validated file",
			"fileType",
			fileType,
			"filePath",
			filePath,
			"size",
			currentStatus.Size,
		)
		*fileInfo = *currentStatus
		if err := j.repo.Update(ctx, processingRecord); err != nil {
			log.Warn("failed to update processing record with existing file status", "error", err)
		}
		return nil
	}

	// If file exists but is invalid, remove it
	if currentStatus.Status == models.FileDownloadStatusFailed && currentStatus.Downloaded {
		log.Warn("Removing invalid existing file", "fileType", fileType, "filePath", filePath)
		if removeErr := os.Remove(filePath); removeErr != nil {
			log.Warn("failed to remove invalid file", "error", removeErr, "file", filePath)
		}
	}

	// Download the file
	log.Info("Downloading file", "fileType", fileType, "yearMonth", yearMonth)

	// Update status to downloading
	now := time.Now().UTC()
	fileInfo.Status = models.FileDownloadStatusDownloading
	fileInfo.DownloadedAt = &now
	fileInfo.ErrorMessage = nil

	if err := j.repo.Update(ctx, processingRecord); err != nil {
		log.Warn("failed to update processing record with downloading status", "error", err)
	}

	// Perform the actual download
	if err := j.download.DownloadXMLFile(ctx, yearMonth, fileType); err != nil {
		errorMsg := err.Error()
		fileInfo.Status = models.FileDownloadStatusFailed
		fileInfo.ErrorMessage = &errorMsg
		if updateErr := j.repo.Update(ctx, processingRecord); updateErr != nil {
			log.Warn("failed to update processing record with error", "error", updateErr)
		}
		return log.Err("failed to download file", err, "fileType", fileType, "yearMonth", yearMonth)
	}

	// Validate the downloaded file
	if err := j.download.ValidateFileChecksum(filePath, expectedChecksum); err != nil {
		// Mark as failed but don't remove the file immediately (for debugging)
		errorMsg := "checksum validation failed"
		fileInfo.Status = models.FileDownloadStatusFailed
		fileInfo.Downloaded = true
		fileInfo.Validated = false
		fileInfo.ErrorMessage = &errorMsg

		if updateErr := j.repo.Update(ctx, processingRecord); updateErr != nil {
			log.Warn("failed to update processing record with validation error", "error", updateErr)
		}

		// Remove invalid file after updating status
		if removeErr := os.Remove(filePath); removeErr != nil {
			log.Warn("failed to remove invalid file", "error", removeErr, "file", filePath)
		}

		return log.Err(
			"file checksum validation failed",
			err,
			"fileType",
			fileType,
			"yearMonth",
			yearMonth,
		)
	}

	// File downloaded and validated successfully
	validatedAt := time.Now().UTC()
	fileInfo.Status = models.FileDownloadStatusValidated
	fileInfo.Downloaded = true
	fileInfo.Validated = true
	fileInfo.ValidatedAt = &validatedAt
	fileInfo.ErrorMessage = nil

	// Get file size
	if info, err := os.Stat(filePath); err == nil {
		fileInfo.Size = info.Size()
	}

	if err := j.repo.Update(ctx, processingRecord); err != nil {
		log.Warn("failed to update processing record with success status", "error", err)
	}

	log.Info(
		"File downloaded and validated successfully",
		"fileType",
		fileType,
		"yearMonth",
		yearMonth,
		"size",
		fileInfo.Size,
	)

	return nil
}


func (j *DiscogsDownloadJob) Schedule() services.Schedule {
	return j.schedule
}
