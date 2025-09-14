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

	// Step 2: Find processing records that can be resumed or need reset
	resumableRecords, err := j.findAndHandleProcessingRecords(ctx)
	if err != nil {
		return log.Err("failed to handle processing records", err)
	}

	// Combine ready and resumable records
	allRecords := append(readyRecords, resumableRecords...)

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

// findAndHandleProcessingRecords finds records in processing status and either resumes or resets them
func (j *DiscogsProcessingJob) findAndHandleProcessingRecords(
	ctx context.Context,
) ([]*models.DiscogsDataProcessing, error) {
	log := j.log.Function("findAndHandleProcessingRecords")

	// Find records in processing status
	processingRecords, err := j.repo.GetByStatus(ctx, models.ProcessingStatusProcessing)
	if err != nil {
		return nil, log.Err("failed to find processing records", err)
	}

	var handledRecords []*models.DiscogsDataProcessing
	oneDayAgo := time.Now().UTC().Add(-24 * time.Hour)

	for _, record := range processingRecords {
		if record.StartedAt == nil {
			continue // Skip records without start time
		}

		// Check if record has been processing for more than 24 hours - reset completely
		if record.StartedAt.Before(oneDayAgo) {
			log.Warn("Found critically stuck processing record (>24hrs), resetting to ready_for_processing",
				"yearMonth", record.YearMonth,
				"id", record.ID,
				"stuckSince", record.StartedAt)

			updateErr := j.transaction.Execute(ctx, func(txCtx context.Context) error {
				// Complete reset to ready_for_processing status
				if err := record.UpdateStatus(models.ProcessingStatusReadyForProcessing); err != nil {
					return err
				}

				// Clear error message and stats for complete retry
				record.ErrorMessage = nil
				record.ProcessingStats = &models.ProcessingStats{}

				return j.repo.Update(txCtx, record)
			})

			if updateErr != nil {
				log.Error("Failed to reset critically stuck record",
					"error", updateErr,
					"yearMonth", record.YearMonth,
					"id", record.ID)
				continue
			}

			handledRecords = append(handledRecords, record)
		} else {
			// Record is in processing but not critically stuck - check if we can resume
			pendingFileTypes := j.checkPendingFileTypes(record)
			if len(pendingFileTypes) > 0 {
				log.Info("Found processing record with pending file types, will resume",
					"yearMonth", record.YearMonth,
					"id", record.ID,
					"pendingTypes", pendingFileTypes,
					"processingSince", record.StartedAt)

				// Add to processing list to resume where left off
				handledRecords = append(handledRecords, record)
			} else {
				// All files processed but not marked complete - complete it
				log.Info("Found processing record with all files processed, marking complete",
					"yearMonth", record.YearMonth,
					"id", record.ID)

				updateErr := j.transaction.Execute(ctx, func(txCtx context.Context) error {
					return j.completeProcessing(ctx, record, record.YearMonth)
				})

				if updateErr != nil {
					log.Error("Failed to complete processing record",
						"error", updateErr,
						"yearMonth", record.YearMonth,
						"id", record.ID)
				}
			}
		}
	}

	if len(handledRecords) > 0 {
		log.Info("Found processing records to handle", "count", len(handledRecords))
	}

	return handledRecords, nil
}

// checkPendingFileTypes returns file types that still need processing
func (j *DiscogsProcessingJob) checkPendingFileTypes(record *models.DiscogsDataProcessing) []string {
	var pending []string

	// Define all file types we should process
	fileTypes := []string{"labels", "artists", "masters", "releases"}

	for _, fileType := range fileTypes {
		// Check if this file type has been processed based on stats
		processed := false

		if record.ProcessingStats != nil {
			switch fileType {
			case "labels":
				processed = record.ProcessingStats.LabelsProcessed > 0
			case "artists":
				processed = record.ProcessingStats.ArtistsProcessed > 0
			case "masters":
				processed = record.ProcessingStats.MastersProcessed > 0
			case "releases":
				processed = record.ProcessingStats.ReleasesProcessed > 0
			}
		}

		// If not processed, check if file exists and is validated
		if !processed {
			if record.ProcessingStats != nil {
				fileInfo := record.ProcessingStats.GetFileInfo(fileType)
				if fileInfo != nil && fileInfo.Status == models.FileDownloadStatusValidated {
					pending = append(pending, fileType)
				}
			}
		}
	}

	return pending
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

	// Process all XML files in dependency order: Labels → Artists → Masters → Releases
	if err := j.processAllXMLFiles(ctx, processingRecord, yearMonth, downloadDir); err != nil {
		return log.Err("XML processing failed", err, "yearMonth", yearMonth)
	}

	log.Info("All XML processing completed successfully", "yearMonth", yearMonth)
	return nil
}

// processAllXMLFiles handles the XML processing for all file types in dependency order
func (j *DiscogsProcessingJob) processAllXMLFiles(
	ctx context.Context,
	processingRecord *models.DiscogsDataProcessing,
	yearMonth string,
	downloadDir string,
) error {
	log := j.log.Function("processAllXMLFiles")

	// Define processing order and file types
	fileTypes := []struct {
		name   string
		method func(context.Context, string, string) (*services.ProcessingResult, error)
	}{
		{"labels", j.xmlProcessing.ProcessLabelsFile},
		{"artists", j.xmlProcessing.ProcessArtistsFile},
		{"masters", j.xmlProcessing.ProcessMastersFile},
		{"releases", j.xmlProcessing.ProcessReleasesFile},
	}

	var totalProcessed int
	var totalInserted int
	var totalUpdated int
	var totalErrors int

	// Process each file type in dependency order
	for _, fileType := range fileTypes {
		filePath := filepath.Join(downloadDir, fmt.Sprintf("%s.xml.gz", fileType.name))

		// Check if this file type has already been processed
		alreadyProcessed := false
		if processingRecord.ProcessingStats != nil {
			switch fileType.name {
			case "labels":
				alreadyProcessed = processingRecord.ProcessingStats.LabelsProcessed > 0
			case "artists":
				alreadyProcessed = processingRecord.ProcessingStats.ArtistsProcessed > 0
			case "masters":
				alreadyProcessed = processingRecord.ProcessingStats.MastersProcessed > 0
			case "releases":
				alreadyProcessed = processingRecord.ProcessingStats.ReleasesProcessed > 0
			}
		}

		if alreadyProcessed {
			log.Info("File type already processed, skipping",
				"fileType", fileType.name,
				"yearMonth", yearMonth)
			continue
		}

		// Check if file exists and is validated
		if processingRecord.ProcessingStats != nil {
			fileInfo := processingRecord.ProcessingStats.GetFileInfo(fileType.name)
			if fileInfo == nil || fileInfo.Status != models.FileDownloadStatusValidated {
				log.Info("File not available for processing, skipping",
					"fileType", fileType.name,
					"yearMonth", yearMonth)
				continue
			}
		}

		// Check if file actually exists on disk
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			log.Info("File not found on disk, skipping processing",
				"fileType", fileType.name,
				"yearMonth", yearMonth,
				"filePath", filePath)
			continue
		}

		log.Info("Starting XML processing",
			"fileType", fileType.name,
			"yearMonth", yearMonth,
			"filePath", filePath)

		// Process the XML file
		result, err := fileType.method(ctx, filePath, processingRecord.ID.String())
		if err != nil {
			return log.Err("failed to process XML file", err,
				"fileType", fileType.name,
				"filePath", filePath)
		}

		log.Info("XML processing completed",
			"fileType", fileType.name,
			"yearMonth", yearMonth,
			"totalRecords", result.TotalRecords,
			"processedRecords", result.ProcessedRecords,
			"insertedRecords", result.InsertedRecords,
			"updatedRecords", result.UpdatedRecords,
			"erroredRecords", result.ErroredRecords)

		// Accumulate totals
		totalProcessed += result.ProcessedRecords
		totalInserted += result.InsertedRecords
		totalUpdated += result.UpdatedRecords
		totalErrors += result.ErroredRecords

		// TODO: Re-enable file cleanup after confirming everything works
		// Clean up the processed file to save disk space
		// if err := os.Remove(filePath); err != nil {
		// 	log.Warn("failed to clean up file after processing",
		// 		"error", err,
		// 		"fileType", fileType.name,
		// 		"filePath", filePath)
		// } else {
		// 	log.Info("Cleaned up file after processing",
		// 		"fileType", fileType.name,
		// 		"filePath", filePath)
		// }
		log.Info("File cleanup disabled during testing",
			"fileType", fileType.name,
			"filePath", filePath)
	}

	log.Info("All XML processing completed",
		"yearMonth", yearMonth,
		"totalProcessed", totalProcessed,
		"totalInserted", totalInserted,
		"totalUpdated", totalUpdated,
		"totalErrors", totalErrors)

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

