package services

import (
	"compress/gzip"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"
	"waugzee/internal/imports"
	"waugzee/internal/logger"
	"waugzee/internal/models"
	"waugzee/internal/repositories"
)

const (
	XML_BATCH_SIZE = 1000
	PROGRESS_REPORT_INTERVAL = 10000
)

type XMLProcessingService struct {
	labelRepo                 repositories.LabelRepository
	discogsDataProcessingRepo repositories.DiscogsDataProcessingRepository
	transactionService        *TransactionService
	log                       logger.Logger
}

func NewXMLProcessingService(
	labelRepo repositories.LabelRepository,
	discogsDataProcessingRepo repositories.DiscogsDataProcessingRepository,
	transactionService *TransactionService,
) *XMLProcessingService {
	return &XMLProcessingService{
		labelRepo:                 labelRepo,
		discogsDataProcessingRepo: discogsDataProcessingRepo,
		transactionService:        transactionService,
		log:                       logger.New("xmlProcessingService"),
	}
}

type ProcessingResult struct {
	TotalRecords     int
	ProcessedRecords int
	InsertedRecords  int
	UpdatedRecords   int
	ErroredRecords   int
	Errors           []string
}

func (s *XMLProcessingService) ProcessLabelsFile(ctx context.Context, filePath string, processingID string) (*ProcessingResult, error) {
	log := s.log.Function("ProcessLabelsFile")

	log.Info("Starting labels file processing", "filePath", filePath, "processingID", processingID)

	// Open and decompress the gzipped XML file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, log.Err("failed to open labels file", err, "filePath", filePath)
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, log.Err("failed to create gzip reader", err, "filePath", filePath)
	}
	defer gzipReader.Close()

	// Create XML decoder for streaming
	decoder := xml.NewDecoder(gzipReader)

	result := &ProcessingResult{
		Errors: make([]string, 0),
	}

	// Update processing status to "processing"
	if err := s.updateProcessingStatus(ctx, processingID, models.ProcessingStatusProcessing, nil); err != nil {
		log.Warn("failed to update processing status", "error", err, "processingID", processingID)
	}

	var labelBatch []*models.Label
	var recordCount int

	// Stream through the XML file
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("XML parsing error: %v", err))
			result.ErroredRecords++
			continue
		}

		// Look for label start elements
		if startElement, ok := token.(xml.StartElement); ok && startElement.Name.Local == "label" {
			var discogsLabel imports.Label
			if err := decoder.DecodeElement(&discogsLabel, &startElement); err != nil {
				errorMsg := fmt.Sprintf("Failed to decode label element: %v", err)
				result.Errors = append(result.Errors, errorMsg)
				result.ErroredRecords++
				log.Warn("Failed to decode label", "error", err)
				continue
			}

			// Convert Discogs label to our label model
			label := s.convertDiscogsLabel(&discogsLabel)
			if label == nil {
				result.ErroredRecords++
				continue
			}

			labelBatch = append(labelBatch, label)
			recordCount++
			result.TotalRecords++

			// Process batch when it reaches the limit
			if len(labelBatch) >= XML_BATCH_SIZE {
				if err := s.processBatch(ctx, labelBatch, result); err != nil {
					log.Err("failed to process label batch", err, "batchSize", len(labelBatch))
					return result, err
				}
				labelBatch = labelBatch[:0] // Reset batch
			}

			// Report progress every PROGRESS_REPORT_INTERVAL records
			if recordCount%PROGRESS_REPORT_INTERVAL == 0 {
				stats := &models.ProcessingStats{
					TotalRecords:    result.TotalRecords,
					LabelsProcessed: result.ProcessedRecords,
					FailedRecords:   result.ErroredRecords,
				}
				if err := s.updateProcessingStats(ctx, processingID, stats); err != nil {
					log.Warn("failed to update processing stats", "error", err, "recordCount", recordCount)
				}
				log.Info("Processing progress", "processed", recordCount, "inserted", result.InsertedRecords, "updated", result.UpdatedRecords, "errors", result.ErroredRecords)
			}
		}
	}

	// Process remaining batch
	if len(labelBatch) > 0 {
		if err := s.processBatch(ctx, labelBatch, result); err != nil {
			log.Err("failed to process final label batch", err, "batchSize", len(labelBatch))
			return result, err
		}
	}

	// Update final processing status
	finalStats := &models.ProcessingStats{
		TotalRecords:    result.TotalRecords,
		LabelsProcessed: result.ProcessedRecords,
		FailedRecords:   result.ErroredRecords,
	}

	status := models.ProcessingStatusCompleted

	if err := s.updateProcessingStatus(ctx, processingID, status, finalStats); err != nil {
		log.Warn("failed to update final processing status", "error", err)
	}

	log.Info("Labels file processing completed",
		"total", result.TotalRecords,
		"processed", result.ProcessedRecords,
		"inserted", result.InsertedRecords,
		"updated", result.UpdatedRecords,
		"errors", result.ErroredRecords,
	)

	return result, nil
}

func (s *XMLProcessingService) processBatch(ctx context.Context, labels []*models.Label, result *ProcessingResult) error {
	log := s.log.Function("processBatch")

	// Use transaction service for batch processing
	return s.transactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		inserted, updated, err := s.labelRepo.UpsertBatch(txCtx, labels)
		if err != nil {
			errorMsg := fmt.Sprintf("Failed to upsert label batch: %v", err)
			result.Errors = append(result.Errors, errorMsg)
			result.ErroredRecords += len(labels)
			return err
		}

		result.ProcessedRecords += len(labels)
		result.InsertedRecords += inserted
		result.UpdatedRecords += updated

		log.Info("Processed label batch", "size", len(labels), "inserted", inserted, "updated", updated)
		return nil
	})
}

func (s *XMLProcessingService) convertDiscogsLabel(discogsLabel *imports.Label) *models.Label {
	// Skip labels with invalid data
	if discogsLabel.Name == "" || discogsLabel.ID == 0 {
		return nil
	}

	label := &models.Label{
		Name: strings.TrimSpace(discogsLabel.Name),
	}

	// Set Discogs ID
	discogsID := int64(discogsLabel.ID)
	label.DiscogsID = &discogsID

	// Set optional fields
	if discogsLabel.ContactInfo != "" {
		// For now, we'll use ContactInfo as part of the profile/notes
		// In the future, we might want to parse contact info more specifically
	}

	if discogsLabel.Profile != "" {
		// We don't have a profile field in our Label model currently
		// This could be added if needed for future functionality
	}

	// For now, we don't have direct mappings for country, founded year, website from Discogs labels XML
	// These would need to be parsed from profile or contact info if needed

	return label
}

func (s *XMLProcessingService) updateProcessingStatus(ctx context.Context, processingID string, status models.ProcessingStatus, stats *models.ProcessingStats) error {
	return s.transactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
		processing, err := s.discogsDataProcessingRepo.GetByID(txCtx, processingID)
		if err != nil {
			return err
		}

		processing.Status = status

		if stats != nil {
			processing.ProcessingStats = stats
		}

		return s.discogsDataProcessingRepo.Update(txCtx, processing)
	})
}

func (s *XMLProcessingService) updateProcessingStats(ctx context.Context, processingID string, stats *models.ProcessingStats) error {
	return s.updateProcessingStatus(ctx, processingID, models.ProcessingStatusProcessing, stats)
}