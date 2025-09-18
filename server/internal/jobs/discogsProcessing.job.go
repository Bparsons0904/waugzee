package jobs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
	"waugzee/internal/logger"
	"waugzee/internal/models"
	"waugzee/internal/repositories"
	"waugzee/internal/services"
)

type DiscogsProcessingJob struct {
	repo                    repositories.DiscogsDataProcessingRepository
	xmlProcessing           *services.XMLProcessingService
	simplifiedXMLProcessing *services.SimplifiedXMLProcessingService
	log                     logger.Logger
	schedule                services.Schedule
}

func NewDiscogsProcessingJob(
	repo repositories.DiscogsDataProcessingRepository,
	xmlProcessing *services.XMLProcessingService,
	simplifiedXMLProcessing *services.SimplifiedXMLProcessingService,
	schedule services.Schedule,
) *DiscogsProcessingJob {
	return &DiscogsProcessingJob{
		repo:                    repo,
		xmlProcessing:           xmlProcessing,
		simplifiedXMLProcessing: simplifiedXMLProcessing,
		log:                     logger.New("discogsProcessingJob"),
		schedule:                schedule,
	}
}

func (j *DiscogsProcessingJob) Name() string {
	return "DiscogsDailyProcessingCheck"
}

func (j *DiscogsProcessingJob) Execute(ctx context.Context) error {
	log := j.log.Function("Execute")
	log.Info("Starting Discogs data processing check")

	yearMonth := time.Now().UTC().Format("2006-01")
	record, err := j.repo.GetByYearMonth(ctx, yearMonth)
	if err != nil {
		return log.Err("failed to find record ready for processing", err)
	}

	switch record.Status {
	case models.ProcessingStatusReadyForProcessing:
		log.Info("Starting processing for record", "yearMonth", yearMonth, "id", record.ID)
	case models.ProcessingStatusProcessing:
		log.Info(
			"Data processing already in progress for this month, attempting to resume",
			"yearMonth",
			yearMonth,
		)
	case models.ProcessingStatusCompleted:
		log.Info("Data processing already completed for this month", "yearMonth", yearMonth)
		return nil
	case models.ProcessingStatusFailed:
		log.Info("Data processing already failed for this month", "yearMonth", yearMonth)
		return nil
	}

	log.Info("Starting record processing", "yearMonth", record.YearMonth, "id", record.ID)
	if err := j.processRecord(ctx, record); err != nil {
		return log.Err("Failed to process record", err,
			"yearMonth", record.YearMonth,
			"id", record.ID)
	}

	return nil
}

func (j *DiscogsProcessingJob) processRecord(
	ctx context.Context,
	record *models.DiscogsDataProcessing,
) error {
	log := j.log.Function("processRecord")

	yearMonth := record.YearMonth
	log.Info("Starting processing for record", "yearMonth", yearMonth, "id", record.ID)

	// Only transition to processing if not already in processing status
	if record.Status != models.ProcessingStatusProcessing {
		if err := record.UpdateStatus(models.ProcessingStatusProcessing); err != nil {
			return log.Err("failed to transition to processing status", err, "yearMonth", yearMonth)
		}

		now := time.Now().UTC()
		record.StartedAt = &now
	}

	// Initialize ProcessingStats if nil to prevent GORM errors
	if record.ProcessingStats == nil {
		record.ProcessingStats = &models.ProcessingStats{}
	}

	if err := j.repo.Update(ctx, record); err != nil {
		return log.Err(
			"failed to update processing record to processing",
			err,
			"yearMonth",
			yearMonth,
		)
	}

	log.Info("Transitioned to processing status", "yearMonth", yearMonth, "status", record.Status)
	downloadDir := fmt.Sprintf("%s/%s", services.DiscogsDataDir, yearMonth)

	fileTypes := []struct {
		name   string
		method func(context.Context, string, string) (*services.ProcessingResult, error)
	}{
		{"labels", j.simplifiedXMLProcessing.ProcessLabelsFile},
		{"artists", j.simplifiedXMLProcessing.ProcessArtistsFile},
		{"masters", j.simplifiedXMLProcessing.ProcessMastersFile},
		{"releases", j.simplifiedXMLProcessing.ProcessReleasesFile},
	}

	var totalProcessed int
	var totalInserted int
	var totalUpdated int
	var totalErrors int

	// Prepare for concurrent processing
	var wg sync.WaitGroup
	errorChan := make(chan error, len(fileTypes))
	resultsChan := make(chan *services.ProcessingResult, len(fileTypes))

	// Mutex to protect record updates since multiple goroutines will update ProcessingStats
	var recordMutex sync.Mutex

	// Process all file types concurrently
	for _, fileType := range fileTypes {
		// Create local copies for goroutine closure
		ft := fileType
		filePath := filepath.Join(downloadDir, fmt.Sprintf("%s.xml.gz", ft.name))

		// Check if this file type has already been completed
		recordMutex.Lock()
		if record.ProcessingStats == nil {
			record.ProcessingStats = &models.ProcessingStats{}
		}
		if record.ProcessingStats.IsFileProcessingCompleted(ft.name) {
			log.Info("File type already processed successfully, skipping",
				"fileType", ft.name,
				"yearMonth", yearMonth,
				"status", record.ProcessingStats.GetFileProcessingStatus(ft.name))
			recordMutex.Unlock()
			continue
		}

		// Check if file exists and is validated
		if record.ProcessingStats != nil {
			fileInfo := record.ProcessingStats.GetFileInfo(ft.name)
			if fileInfo == nil || fileInfo.Status != models.FileDownloadStatusValidated {
				log.Info("File not available for processing, skipping",
					"fileType", ft.name,
					"yearMonth", yearMonth)
				recordMutex.Unlock()
				continue
			}
		}
		recordMutex.Unlock()

		// Check if file actually exists on disk
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			log.Info("File not found on disk, skipping processing",
				"fileType", ft.name,
				"yearMonth", yearMonth,
				"filePath", filePath)
			continue
		}

		// Start goroutine for this file
		wg.Add(1)
		go func(fileType struct {
			name   string
			method func(context.Context, string, string) (*services.ProcessingResult, error)
		}, filePath string,
		) {
			defer wg.Done()

			// Set processing status to indicate we're starting this file
			recordMutex.Lock()
			if record.ProcessingStats == nil {
				record.ProcessingStats = &models.ProcessingStats{}
			}
			record.ProcessingStats.SetFileProcessingStatus(
				fileType.name,
				models.FileProcessingStatusProcessing,
			)
			if updateErr := j.repo.Update(ctx, record); updateErr != nil {
				log.Warn(
					"failed to update file processing status to processing",
					"error",
					updateErr,
				)
			}
			recordMutex.Unlock()

			log.Info("Starting XML processing",
				"fileType", fileType.name,
				"yearMonth", yearMonth,
				"filePath", filePath,
				"processingStatus", record.ProcessingStats.GetFileProcessingStatus(fileType.name))

			// Process the XML file
			result, err := fileType.method(ctx, filePath, record.ID.String())
			if err != nil {
				// Update file processing status to failed
				recordMutex.Lock()
				if record.ProcessingStats == nil {
					record.ProcessingStats = &models.ProcessingStats{}
				}
				record.ProcessingStats.SetFileProcessingStatus(
					fileType.name,
					models.FileProcessingStatusFailed,
				)

				// Update overall record with error state
				errorMsg := err.Error()
				record.ErrorMessage = &errorMsg
				if statusErr := record.UpdateStatus(models.ProcessingStatusFailed); statusErr != nil {
					log.Warn(
						"failed to update processing record status to failed",
						"error",
						statusErr,
					)
				}

				if updateErr := j.repo.Update(ctx, record); updateErr != nil {
					log.Warn("failed to update processing record with error", "error", updateErr)
				}
				recordMutex.Unlock()

				// Send error to channel but don't return to avoid deadlock
				errorChan <- fmt.Errorf("failed to process XML file %s: %w", fileType.name, err)
				return
			}

			// Mark file processing as completed
			recordMutex.Lock()
			if record.ProcessingStats == nil {
				record.ProcessingStats = &models.ProcessingStats{}
			}
			record.ProcessingStats.SetFileProcessingStatus(
				fileType.name,
				models.FileProcessingStatusCompleted,
			)

			// Update the record to persist the completion status
			if updateErr := j.repo.Update(ctx, record); updateErr != nil {
				log.Warn("failed to update file processing status to completed", "error", updateErr)
			}
			recordMutex.Unlock()

			log.Info("XML processing completed",
				"fileType", fileType.name,
				"yearMonth", yearMonth,
				"totalRecords", result.TotalRecords,
				"processedRecords", result.ProcessedRecords,
				"insertedRecords", result.InsertedRecords,
				"updatedRecords", result.UpdatedRecords,
				"erroredRecords", result.ErroredRecords,
				"processingStatus", record.ProcessingStats.GetFileProcessingStatus(fileType.name))

			log.Info("File cleanup disabled during testing",
				"fileType", fileType.name,
				"filePath", filePath)

			// Send result to channel
			resultsChan <- result
		}(ft, filePath)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errorChan)
	close(resultsChan)

	// Check for errors
	var processingErrors []error
	for err := range errorChan {
		processingErrors = append(processingErrors, err)
	}

	// If we have any errors, return the first one (following original behavior)
	if len(processingErrors) > 0 {
		return processingErrors[0]
	}

	// Collect all results
	for result := range resultsChan {
		totalProcessed += result.ProcessedRecords
		totalInserted += result.InsertedRecords
		totalUpdated += result.UpdatedRecords
		totalErrors += result.ErroredRecords
	}

	log.Info("All XML processing completed",
		"yearMonth", yearMonth,
		"totalProcessed", totalProcessed,
		"totalInserted", totalInserted,
		"totalUpdated", totalUpdated,
		"totalErrors", totalErrors)

	// Complete the processing workflow
	if err := j.completeProcessing(ctx, record, yearMonth); err != nil {
		// Update record with error state
		errorMsg := err.Error()
		record.ErrorMessage = &errorMsg
		if statusErr := record.UpdateStatus(models.ProcessingStatusFailed); statusErr != nil {
			log.Warn("failed to update processing record status to failed", "error", statusErr)
		}

		// Ensure ProcessingStats is initialized for error update
		if record.ProcessingStats == nil {
			record.ProcessingStats = &models.ProcessingStats{}
		}

		if updateErr := j.repo.Update(ctx, record); updateErr != nil {
			log.Warn("failed to update processing record with error", "error", updateErr)
		}

		return log.Err("processing failed", err, "yearMonth", yearMonth)
	}

	log.Info("Record processing completed successfully", "yearMonth", yearMonth)
	return nil
}

// completeProcessing handles the final completion steps
func (j *DiscogsProcessingJob) completeProcessing(
	ctx context.Context,
	processingRecord *models.DiscogsDataProcessing,
	yearMonth string,
) error {
	log := j.log.Function("completeProcessing")

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

	if err := j.repo.Update(ctx, processingRecord); err != nil {
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
}

func (j *DiscogsProcessingJob) Schedule() services.Schedule {
	return j.schedule
}
