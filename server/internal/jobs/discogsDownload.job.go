package jobs

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
			processingRecord.UpdateStatus(models.ProcessingStatusFailed)

			if updateErr := j.repo.Update(txCtx, processingRecord); updateErr != nil {
				log.Warn("failed to update processing record with error", "error", updateErr)
			}

			return log.Err("download failed", err, "yearMonth", yearMonth)
		}

		log.Info("Discogs data processing check completed successfully", "yearMonth", yearMonth)
		return nil
	})
}

// performDownload handles the actual download process
func (j *DiscogsDownloadJob) performDownload(
	ctx context.Context,
	processingRecord *models.DiscogsDataProcessing,
	yearMonth string,
) error {
	log := j.log.Function("performDownload")

	log.Info("Starting checksum download", "yearMonth", yearMonth)

	// Download the CHECKSUM.txt file using the download service
	if err := j.download.DownloadChecksum(ctx, yearMonth); err != nil {
		return log.Err("failed to download checksum file", err, "yearMonth", yearMonth)
	}

	// Parse the downloaded checksum file
	checksumFile := filepath.Join(fmt.Sprintf("/tmp/discogs-%s", yearMonth), "CHECKSUM.txt")
	checksums, err := j.download.ParseChecksumFile(checksumFile)
	if err != nil {
		return log.Err("failed to parse checksum file", err, "checksumFile", checksumFile)
	}

	// Update processing record with checksums and transition to ready_for_processing
	processingRecord.FileChecksums = checksums
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

	// Clean up downloaded file to save space (we only need the parsed checksums)
	if err := os.Remove(checksumFile); err != nil {
		log.Warn("failed to clean up checksum file", "error", err, "file", checksumFile)
	}

	log.Info("Download completed successfully",
		"yearMonth", yearMonth,
		"status", processingRecord.Status,
		"foundArtists", checksums.ArtistsDump != "",
		"foundLabels", checksums.LabelsDump != "",
		"foundMasters", checksums.MastersDump != "",
		"foundReleases", checksums.ReleasesDump != "")

	return nil
}

func (j *DiscogsDownloadJob) Schedule() services.Schedule {
	return j.schedule
}
