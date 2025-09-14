package jobs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
	"waugzee/internal/logger"
	"waugzee/internal/models"
	"waugzee/internal/repositories"
	"waugzee/internal/services"
)

type DiscogsProcessingJob struct {
	repo          repositories.DiscogsDataProcessingRepository
	transaction   *services.TransactionService
	xmlProcessing *services.XMLProcessingService
	log           logger.Logger
	schedule      services.Schedule
}

func NewDiscogsProcessingJob(
	repo repositories.DiscogsDataProcessingRepository,
	transaction *services.TransactionService,
	xmlProcessing *services.XMLProcessingService,
	schedule services.Schedule,
) *DiscogsProcessingJob {
	return &DiscogsProcessingJob{
		repo:          repo,
		transaction:   transaction,
		xmlProcessing: xmlProcessing,
		log:           logger.New("discogsProcessingJob"),
		schedule:      schedule,
	}
}

func (j *DiscogsProcessingJob) Name() string {
	return "DiscogsDailyProcessingCheck"
}

func (j *DiscogsProcessingJob) Execute(ctx context.Context) error {
	log := j.log.Function("Execute")

	log.Info("Starting Discogs data processing check")

	// Step 1: Find records ready for processing
	readyRecords, err := j.repo.GetByStatus(ctx, models.ProcessingStatusReadyForProcessing)
	if err != nil {
		return log.Err("failed to find records ready for processing", err)
	}

	// Step 2: Find stuck processing records (older than 2 hours) and reset them
	stuckRecords, err := j.findAndResetStuckRecords(ctx)
	if err != nil {
		return log.Err("failed to handle stuck processing records", err)
	}

	// Combine ready and recovered records
	allRecords := append(readyRecords, stuckRecords...)

	if len(allRecords) == 0 {
		log.Info("No records found ready for processing")
		return nil
	}

	log.Info("Found records to process", "count", len(allRecords))

	// Step 3: Process each record
	successCount := 0
	failureCount := 0

	for _, record := range allRecords {
		if err := j.processRecord(ctx, record); err != nil {
			log.Error("Failed to process record",
				"error", err,
				"yearMonth", record.YearMonth,
				"id", record.ID)
			failureCount++
		} else {
			successCount++
		}
	}

	log.Info("Processing check completed",
		"totalRecords", len(allRecords),
		"successCount", successCount,
		"failureCount", failureCount)

	return nil
}

// findAndResetStuckRecords finds records stuck in "processing" status for > 2 hours and resets them
func (j *DiscogsProcessingJob) findAndResetStuckRecords(
	ctx context.Context,
) ([]*models.DiscogsDataProcessing, error) {
	log := j.log.Function("findAndResetStuckRecords")

	// Find records in processing status
	processingRecords, err := j.repo.GetByStatus(ctx, models.ProcessingStatusProcessing)
	if err != nil {
		return nil, log.Err("failed to find processing records", err)
	}

	var stuckRecords []*models.DiscogsDataProcessing
	twoHoursAgo := time.Now().UTC().Add(-2 * time.Hour)

	for _, record := range processingRecords {
		// Check if record has been processing for more than 2 hours
		if record.StartedAt != nil && record.StartedAt.Before(twoHoursAgo) {
			log.Warn("Found stuck processing record, resetting to ready_for_processing",
				"yearMonth", record.YearMonth,
				"id", record.ID,
				"stuckSince", record.StartedAt)

			// Use atomic transaction for stuck record reset
			updateErr := j.transaction.Execute(ctx, func(txCtx context.Context) error {
				// Reset to ready_for_processing status
				if err := record.UpdateStatus(models.ProcessingStatusReadyForProcessing); err != nil {
					return err
				}

				// Clear error message for retry
				record.ErrorMessage = nil

				return j.repo.Update(txCtx, record)
			})

			if updateErr != nil {
				log.Error("Failed to reset stuck record",
					"error", updateErr,
					"yearMonth", record.YearMonth,
					"id", record.ID)
				continue
			}

			stuckRecords = append(stuckRecords, record)
		}
	}

	if len(stuckRecords) > 0 {
		log.Info("Reset stuck records for retry", "count", len(stuckRecords))
	}

	return stuckRecords, nil
}

// processRecord handles the processing of a single record
func (j *DiscogsProcessingJob) processRecord(
	ctx context.Context,
	record *models.DiscogsDataProcessing,
) error {
	log := j.log.Function("processRecord")

	yearMonth := record.YearMonth
	log.Info("Starting processing for record", "yearMonth", yearMonth, "id", record.ID)

	// Use atomic transaction for status updates only
	err := j.transaction.Execute(ctx, func(txCtx context.Context) error {
		// Transition to processing status
		if err := record.UpdateStatus(models.ProcessingStatusProcessing); err != nil {
			return log.Err("failed to transition to processing status", err, "yearMonth", yearMonth)
		}

		// Set processing start time
		now := time.Now().UTC()
		record.StartedAt = &now

		if err := j.repo.Update(txCtx, record); err != nil {
			return log.Err(
				"failed to update processing record to processing",
				err,
				"yearMonth",
				yearMonth,
			)
		}
		return nil
	})
	if err != nil {
		return err
	}

	log.Info("Transitioned to processing status", "yearMonth", yearMonth, "status", record.Status)

	// Perform the actual processing (outside of transaction)
	if err := j.performProcessing(ctx, record, yearMonth); err != nil {
		// Use atomic transaction for error handling
		updateErr := j.transaction.Execute(ctx, func(txCtx context.Context) error {
			errorMsg := err.Error()
			record.ErrorMessage = &errorMsg
			if statusErr := record.UpdateStatus(models.ProcessingStatusFailed); statusErr != nil {
				log.Warn("failed to update processing record status to failed", "error", statusErr)
				return statusErr
			}

			if updateErr := j.repo.Update(txCtx, record); updateErr != nil {
				log.Warn("failed to update processing record with error", "error", updateErr)
				return updateErr
			}
			return nil
		})
		if updateErr != nil {
			log.Error("failed to update record with error state", "error", updateErr)
		}

		return log.Err("processing failed", err, "yearMonth", yearMonth)
	}

	log.Info("Record processing completed successfully", "yearMonth", yearMonth)
	return nil
}

// performProcessing handles the actual XML processing
func (j *DiscogsProcessingJob) performProcessing(
	ctx context.Context,
	processingRecord *models.DiscogsDataProcessing,
	yearMonth string,
) error {
	log := j.log.Function("performProcessing")

	downloadDir := fmt.Sprintf("/app/discogs-data/%s", yearMonth)

	// Process labels XML file (Phase 1: Labels only)
	if err := j.processLabelsXML(ctx, processingRecord, yearMonth, downloadDir); err != nil {
		return log.Err("labels XML processing failed", err, "yearMonth", yearMonth)
	}

	log.Info("All XML processing completed successfully", "yearMonth", yearMonth)
	return nil
}

// processLabelsXML handles the XML processing for labels file only (Phase 1)
func (j *DiscogsProcessingJob) processLabelsXML(
	ctx context.Context,
	processingRecord *models.DiscogsDataProcessing,
	yearMonth string,
	downloadDir string,
) error {
	log := j.log.Function("processLabelsXML")

	labelsFilePath := filepath.Join(downloadDir, "labels.xml.gz")

	// Check if labels file exists and is validated
	if processingRecord.ProcessingStats != nil {
		fileInfo := processingRecord.ProcessingStats.GetFileInfo("labels")
		if fileInfo == nil || fileInfo.Status != models.FileDownloadStatusValidated {
			log.Info("Labels file not available for processing, skipping", "yearMonth", yearMonth)
			// This is not an error - some months might not have all file types
			// Continue to completion status
			if err := j.completeProcessing(ctx, processingRecord, yearMonth); err != nil {
				return err
			}
			return nil
		}
	}

	// Check if file actually exists on disk
	if _, err := os.Stat(labelsFilePath); os.IsNotExist(err) {
		log.Info(
			"Labels file not found on disk, skipping processing",
			"yearMonth",
			yearMonth,
			"filePath",
			labelsFilePath,
		)
		// Continue to completion status
		if err := j.completeProcessing(ctx, processingRecord, yearMonth); err != nil {
			return err
		}
		return nil
	}

	log.Info("Starting labels XML processing", "yearMonth", yearMonth, "filePath", labelsFilePath)

	// Process the labels XML file
	result, err := j.xmlProcessing.ProcessLabelsFile(
		ctx,
		labelsFilePath,
		processingRecord.ID.String(),
	)
	if err != nil {
		return log.Err("failed to process labels XML file", err, "filePath", labelsFilePath)
	}

	log.Info("Labels XML processing completed",
		"yearMonth", yearMonth,
		"totalRecords", result.TotalRecords,
		"processedRecords", result.ProcessedRecords,
		"insertedRecords", result.InsertedRecords,
		"updatedRecords", result.UpdatedRecords,
		"erroredRecords", result.ErroredRecords,
	)

	// Clean up the processed file to save disk space
	if err := os.Remove(labelsFilePath); err != nil {
		log.Warn(
			"failed to clean up labels file after processing",
			"error",
			err,
			"filePath",
			labelsFilePath,
		)
	} else {
		log.Info("Cleaned up labels file after processing", "filePath", labelsFilePath)
	}

	// Complete the processing workflow
	if err := j.completeProcessing(ctx, processingRecord, yearMonth); err != nil {
		return err
	}

	return nil
}

// completeProcessing handles the final completion steps
func (j *DiscogsProcessingJob) completeProcessing(
	ctx context.Context,
	processingRecord *models.DiscogsDataProcessing,
	yearMonth string,
) error {
	log := j.log.Function("completeProcessing")

	// Use atomic transaction for completion status update
	return j.transaction.Execute(ctx, func(txCtx context.Context) error {
		// Transition to completed status
		if err := processingRecord.UpdateStatus(models.ProcessingStatusCompleted); err != nil {
			return log.Err(
				"failed to transition to completed status",
				err,
				"yearMonth",
				yearMonth,
			)
		}

		// Set completion time
		now := time.Now().UTC()
		processingRecord.ProcessingCompletedAt = &now

		if err := j.repo.Update(txCtx, processingRecord); err != nil {
			return log.Err(
				"failed to update processing record to completed",
				err,
				"yearMonth",
				yearMonth,
			)
		}

		log.Info("Processing workflow completed successfully",
			"yearMonth", yearMonth,
			"status", processingRecord.Status)

		return nil
	})
}

func (j *DiscogsProcessingJob) Schedule() services.Schedule {
	return j.schedule
}

