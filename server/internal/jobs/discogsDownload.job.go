package jobs

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	logger "github.com/Bparsons0904/goLogger"
	"waugzee/internal/models"
	"waugzee/internal/repositories"
	"waugzee/internal/services"

	"gorm.io/gorm"
)

type DiscogsDownloadJob struct {
	download *services.DownloadService
	repo     repositories.DiscogsDataProcessingRepository
	log      logger.Logger
	schedule services.Schedule
}

func NewDiscogsDownloadJob(
	download *services.DownloadService,
	repo repositories.DiscogsDataProcessingRepository,
	schedule services.Schedule,
) *DiscogsDownloadJob {
	log := logger.New("discogsDownloadJob")
	log.Info("Creating new Discogs download job", "schedule", schedule)

	return &DiscogsDownloadJob{
		download: download,
		repo:     repo,
		log:      log,
		schedule: schedule,
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

	// Check if there's already a processing record for this month
	existing, err := j.repo.GetByYearMonth(ctx, yearMonth)
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

		created, err := j.repo.Create(ctx, processing)
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

			if err := j.repo.Update(ctx, existing); err != nil {
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

	if err := j.repo.Update(ctx, processingRecord); err != nil {
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
	if err := j.performDownload(ctx, processingRecord, yearMonth); err != nil {
		// Check if it's a 404 error (data not available yet)
		if strings.Contains(err.Error(), "status code: 404") {
			log.Info(
				"Data not available yet for this month, will try again next day",
				"yearMonth",
				yearMonth,
			)
			j.download.BroadcastProgress(yearMonth, "not_available", "all", "waiting", 0, 0, err)
			// Reset status back to not_started so it will be retried next day
			if statusErr := processingRecord.UpdateStatus(models.ProcessingStatusNotStarted); statusErr != nil {
				log.Warn("failed to reset processing record status", "error", statusErr)
			}
			if updateErr := j.repo.Update(ctx, processingRecord); updateErr != nil {
				log.Warn("failed to update processing record", "error", updateErr)
			}
			return nil // Success - just not available yet
		}

		// For other errors, mark as failed
		j.download.BroadcastProgress(yearMonth, "failed", "all", "error", 0, 0, err)
		errorMsg := err.Error()
		processingRecord.ErrorMessage = &errorMsg
		if statusErr := processingRecord.UpdateStatus(models.ProcessingStatusFailed); statusErr != nil {
			log.Warn("failed to update processing record status to failed", "error", statusErr)
		}

		if updateErr := j.repo.Update(ctx, processingRecord); updateErr != nil {
			log.Warn("failed to update processing record with error", "error", updateErr)
		}

		return log.Err("download failed", err, "yearMonth", yearMonth)
	}

	log.Info("Discogs data processing check completed successfully", "yearMonth", yearMonth)
	return nil
}

// performDownload handles the actual download process and updates the processing record
func (j *DiscogsDownloadJob) performDownload(
	ctx context.Context,
	processingRecord *models.DiscogsDataProcessing,
	yearMonth string,
) error {
	log := j.log.Function("performDownload")

	log.Info("Starting checksum download", "yearMonth", yearMonth)
	j.download.BroadcastProgress(yearMonth, "downloading", "checksum", "starting", 0, 0, nil)

	// Download the CHECKSUM.txt file using the download service
	if err := j.download.DownloadChecksum(ctx, yearMonth); err != nil {
		j.download.BroadcastProgress(yearMonth, "failed", "checksum", "error", 0, 0, err)
		return log.Err("failed to download checksum file", err, "yearMonth", yearMonth)
	}

	j.download.BroadcastProgress(yearMonth, "completed", "checksum", "parsing", 0, 0, nil)

	// Parse the downloaded checksum file
	checksumFile := filepath.Join(fmt.Sprintf("/app/discogs-data/%s", yearMonth), "CHECKSUM.txt")
	checksums, err := j.download.ParseChecksumFile(checksumFile)
	if err != nil {
		j.download.BroadcastProgress(yearMonth, "failed", "checksum", "error", 0, 0, err)
		return log.Err("failed to parse checksum file", err, "checksumFile", checksumFile)
	}

	j.download.BroadcastProgress(yearMonth, "completed", "checksum", "completed", 0, 0, nil)

	// Update processing record with checksums
	processingRecord.FileChecksums = checksums

	// Initialize processing stats to track file downloads
	processingRecord.InitializeProcessingStats()

	if err := j.repo.Update(ctx, processingRecord); err != nil {
		return log.Err("failed to update processing record with checksums", err, "yearMonth", yearMonth)
	}

	// Download all XML data files
	log.Info("Starting XML file downloads", "yearMonth", yearMonth)

	fileTypes := []string{"artists", "labels", "masters", "releases"}
	for _, fileType := range fileTypes {
		// Initialize file info before download
		processingRecord.InitializeFileInfo(fileType)
		if err := j.repo.Update(ctx, processingRecord); err != nil {
			log.Warn("failed to initialize file info", "fileType", fileType, "error", err)
		}

		log.Info("Downloading XML file", "fileType", fileType, "yearMonth", yearMonth)
		j.download.BroadcastProgress(yearMonth, "downloading", fileType, "starting", 0, 0, nil)
		if err := j.download.DownloadXMLFile(ctx, yearMonth, fileType); err != nil {
			j.download.BroadcastProgress(yearMonth, "failed", fileType, "error", 0, 0, err)
			return log.Err("failed to download XML file", err, "fileType", fileType, "yearMonth", yearMonth)
		}
		j.download.BroadcastProgress(yearMonth, "completed", fileType, "completed", 0, 0, nil)
		log.Info("Downloaded XML file successfully", "fileType", fileType, "yearMonth", yearMonth)

		// Update file info after successful download
		filePath := filepath.Join(fmt.Sprintf("/app/discogs-data/%s", yearMonth), fmt.Sprintf("%s.xml.gz", fileType))
		checksum := processingRecord.GetFileChecksum(fileType)

		fileInfo, err := j.download.GetFileStatus(filePath, checksum)
		if err != nil {
			log.Warn("failed to get file status", "fileType", fileType, "error", err)
		} else {
			processingRecord.SetFileInfo(fileType, fileInfo)
			if err := j.repo.Update(ctx, processingRecord); err != nil {
				log.Warn("failed to update file info", "fileType", fileType, "error", err)
			}
		}
	}

	// Transition to ready_for_processing after all downloads complete
	if err := processingRecord.UpdateStatus(models.ProcessingStatusReadyForProcessing); err != nil {
		return log.Err("failed to transition to ready_for_processing status", err, "yearMonth", yearMonth)
	}

	// Set download completed time
	now := time.Now().UTC()
	processingRecord.DownloadCompletedAt = &now

	if err := j.repo.Update(ctx, processingRecord); err != nil {
		return log.Err("failed to update processing record after download", err, "yearMonth", yearMonth)
	}

	// Clean up downloaded checksum file to save space (we only need the parsed checksums)
	if err := os.Remove(checksumFile); err != nil {
		log.Warn("failed to clean up checksum file", "error", err, "file", checksumFile)
	}

	// Broadcast final completion status
	j.download.BroadcastProgress(yearMonth, "ready_for_processing", "all", "completed", 0, 0, nil)

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