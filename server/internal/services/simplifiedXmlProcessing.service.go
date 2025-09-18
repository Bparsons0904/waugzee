package services

import (
	"context"
	"sync"
	"waugzee/internal/imports"
	"waugzee/internal/logger"
	"waugzee/internal/models"
	"waugzee/internal/repositories"
)

// SimplifiedResult holds processing result for streaming processing
type SimplifiedResult struct {
	Errors []string
}

type SimplifiedXMLProcessingService struct {
	discogsDataProcessingRepo repositories.DiscogsDataProcessingRepository
	parserService             *DiscogsParserService
	bufferManager             *BufferManager
	entityProcessor           *EntityProcessor
	batchCoordinator          *BatchCoordinator
	log                       logger.Logger
}

type ProcessingResult struct {
	TotalRecords     int
	ProcessedRecords int
	InsertedRecords  int
	UpdatedRecords   int
	ErroredRecords   int
	Errors           []string
}

func NewSimplifiedXMLProcessingService(
	discogsDataProcessingRepo repositories.DiscogsDataProcessingRepository,
	labelRepo repositories.LabelRepository,
	artistRepo repositories.ArtistRepository,
	masterRepo repositories.MasterRepository,
	releaseRepo repositories.ReleaseRepository,
	genreRepo repositories.GenreRepository,
	imageRepo repositories.ImageRepository,
	parserService *DiscogsParserService,
) *SimplifiedXMLProcessingService {
	bufferManager := NewBufferManager(
		labelRepo,
		artistRepo,
		masterRepo,
		releaseRepo,
		genreRepo,
		imageRepo,
	)

	batchCoordinator := NewBatchCoordinator(
		labelRepo,
		artistRepo,
		masterRepo,
		releaseRepo,
		genreRepo,
		imageRepo,
	)

	entityProcessor := NewEntityProcessor(parserService)

	return &SimplifiedXMLProcessingService{
		discogsDataProcessingRepo: discogsDataProcessingRepo,
		parserService:             parserService,
		bufferManager:             bufferManager,
		batchCoordinator:          batchCoordinator,
		entityProcessor:           entityProcessor,
		log:                       logger.New("simplifiedXMLProcessingService"),
	}
}

// ProcessFileToMap parses the entire file using channel-based architecture to extract ALL XML data
func (s *SimplifiedXMLProcessingService) ProcessFileToMap(
	ctx context.Context,
	filePath string,
	fileType string,
) (*SimplifiedResult, error) {
	log := s.log.Function("ProcessFileToMap")

	result := &SimplifiedResult{
		Errors: make([]string, 0),
	}

	// Create buffered channels for entity processing
	entityChan := make(chan EntityMessage, 10000)
	completionChan := make(chan CompletionMessage, 1)

	// Create processing buffers for related entities
	buffers := s.bufferManager.CreateProcessingBuffers()

	// Start buffer processing goroutines
	var wg sync.WaitGroup
	processingID := "simplified_processing"

	s.bufferManager.StartBufferProcessors(ctx, buffers, &wg, processingID, s.batchCoordinator)

	// Start parser in goroutine
	parseOptions := ParseOptions{
		FilePath: filePath,
		FileType: fileType,
	}

	go func() {
		defer close(entityChan)
		if err := s.parserService.ParseFileToChannel(ctx, parseOptions, entityChan, completionChan); err != nil {
			log.Error("Channel parsing failed", "error", err, "filePath", filePath)
		}
	}()

	// Process entities from channel
	channelProcessingDone := false

	for !channelProcessingDone {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()

		case entity, ok := <-entityChan:
			if !ok {
				continue
			}

			// Process entities through streaming channels
			switch entity.Type {
			case "label":
				if rawLabel, ok := entity.RawEntity.(*imports.Label); ok && rawLabel.ID > 0 {
					if err := s.entityProcessor.ProcessLabel(rawLabel, entity.ProcessingID, buffers); err != nil {
						result.Errors = append(result.Errors, err.Error())
					}
				}

			case "artist":
				if rawArtist, ok := entity.RawEntity.(*imports.Artist); ok && rawArtist.ID > 0 {
					if err := s.entityProcessor.ProcessArtist(rawArtist, entity.ProcessingID, buffers); err != nil {
						result.Errors = append(result.Errors, err.Error())
					}
				}

			case "master":
				if rawMaster, ok := entity.RawEntity.(*imports.Master); ok && rawMaster.ID > 0 {
					if err := s.entityProcessor.ProcessMaster(rawMaster, entity.ProcessingID, buffers); err != nil {
						result.Errors = append(result.Errors, err.Error())
					}
				}

			case "release":
				if rawRelease, ok := entity.RawEntity.(*imports.Release); ok && rawRelease.ID > 0 {
					if err := s.entityProcessor.ProcessRelease(rawRelease, entity.ProcessingID, buffers); err != nil {
						result.Errors = append(result.Errors, err.Error())
					}
				}
			}

		case _ = <-completionChan:
			channelProcessingDone = true
		}
	}

	// Close buffers to signal completion and wait for all buffer processors to finish
	s.bufferManager.CloseProcessingBuffers(buffers)
	wg.Wait()

	// Flush any remaining batches
	if err := s.batchCoordinator.FlushAllBatches(ctx, processingID); err != nil {
		log.Error("Failed to flush final batches", "error", err)
		result.Errors = append(result.Errors, err.Error())
	}

	return result, nil
}

func (s *SimplifiedXMLProcessingService) ProcessLabelsFile(
	ctx context.Context,
	filePath string,
	processingID string,
) (*ProcessingResult, error) {
	return s.ProcessFile(ctx, filePath, processingID, "labels")
}

func (s *SimplifiedXMLProcessingService) ProcessArtistsFile(
	ctx context.Context,
	filePath string,
	processingID string,
) (*ProcessingResult, error) {
	return s.ProcessFile(ctx, filePath, processingID, "artists")
}

func (s *SimplifiedXMLProcessingService) ProcessMastersFile(
	ctx context.Context,
	filePath string,
	processingID string,
) (*ProcessingResult, error) {
	return s.ProcessFile(ctx, filePath, processingID, "masters")
}

func (s *SimplifiedXMLProcessingService) ProcessReleasesFile(
	ctx context.Context,
	filePath string,
	processingID string,
) (*ProcessingResult, error) {
	return s.ProcessFile(ctx, filePath, processingID, "releases")
}

// ProcessFile is the consolidated generic method that handles all file types
func (s *SimplifiedXMLProcessingService) ProcessFile(
	ctx context.Context,
	filePath string,
	processingID string,
	fileType string,
) (*ProcessingResult, error) {
	log := s.log.Function("ProcessFile")

	// Update processing status to "processing"
	if err := s.updateProcessingStatus(ctx, processingID, models.ProcessingStatusProcessing, nil); err != nil {
		log.Error("failed to update processing status", "error", err, "processingID", processingID)
	}

	// Use simplified parsing approach
	simplifiedResult, err := s.ProcessFileToMap(ctx, filePath, fileType)
	if err != nil {
		// Update status to failed
		if statusErr := s.updateProcessingStatus(ctx, processingID, models.ProcessingStatusFailed, nil); statusErr != nil {
			log.Error(
				"failed to update processing status to failed",
				"error",
				statusErr,
				"processingID",
				processingID,
			)
		}
		return nil, log.Err("simplified parsing failed", err, "filePath", filePath)
	}

	// Convert SimplifiedResult to ProcessingResult for compatibility
	result := s.convertToProcessingResult(simplifiedResult)

	// Update final processing status to completed
	if err := s.updateProcessingStatus(ctx, processingID, models.ProcessingStatusCompleted, nil); err != nil {
		log.Error("failed to update final processing status", "error", err)
	}

	return result, nil
}

// convertToProcessingResult converts SimplifiedResult to ProcessingResult for compatibility
func (s *SimplifiedXMLProcessingService) convertToProcessingResult(
	simplifiedResult *SimplifiedResult,
) *ProcessingResult {
	return &ProcessingResult{
		TotalRecords:     0, // No longer tracked
		ProcessedRecords: 0, // No longer tracked
		InsertedRecords:  0, // No longer tracked
		UpdatedRecords:   0, // No longer tracked
		ErroredRecords:   len(simplifiedResult.Errors),
		Errors:           simplifiedResult.Errors,
	}
}

func (s *SimplifiedXMLProcessingService) updateProcessingStatus(
	ctx context.Context,
	processingID string,
	status models.ProcessingStatus,
	stats *models.ProcessingStats,
) error {
	processing, err := s.discogsDataProcessingRepo.GetByID(ctx, processingID)
	if err != nil {
		return err
	}

	processing.Status = status

	if stats != nil {
		processing.ProcessingStats = stats
	}

	return s.discogsDataProcessingRepo.Update(ctx, processing)
}

