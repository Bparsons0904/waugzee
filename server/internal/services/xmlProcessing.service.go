package services

import (
	"context"
	"fmt"
	"strings"
	"waugzee/internal/logger"
	"waugzee/internal/models"
	"waugzee/internal/repositories"
)

const (
	XML_BATCH_SIZE           = 2000
	PROGRESS_REPORT_INTERVAL = 50000
)

type XMLProcessingService struct {
	labelRepo                 repositories.LabelRepository
	artistRepo                repositories.ArtistRepository
	masterRepo                repositories.MasterRepository
	releaseRepo               repositories.ReleaseRepository
	trackRepo                 repositories.TrackRepository
	genreRepo                 repositories.GenreRepository
	discogsDataProcessingRepo repositories.DiscogsDataProcessingRepository
	parserService             *DiscogsParserService
	log                       logger.Logger

	// Performance caches
	genreCache  map[string]*models.Genre // Cache genres by name to avoid repeated DB lookups
	artistCache map[int64]*models.Artist // Cache artists by Discogs ID to avoid repeated DB lookups
}

func NewXMLProcessingService(
	labelRepo repositories.LabelRepository,
	artistRepo repositories.ArtistRepository,
	masterRepo repositories.MasterRepository,
	releaseRepo repositories.ReleaseRepository,
	trackRepo repositories.TrackRepository,
	genreRepo repositories.GenreRepository,
	discogsDataProcessingRepo repositories.DiscogsDataProcessingRepository,
	parserService *DiscogsParserService,
) *XMLProcessingService {
	return &XMLProcessingService{
		labelRepo:                 labelRepo,
		artistRepo:                artistRepo,
		masterRepo:                masterRepo,
		releaseRepo:               releaseRepo,
		trackRepo:                 trackRepo,
		genreRepo:                 genreRepo,
		discogsDataProcessingRepo: discogsDataProcessingRepo,
		parserService:             parserService,
		log:                       logger.New("xmlProcessingService"),
		genreCache:                make(map[string]*models.Genre),
		artistCache:               make(map[int64]*models.Artist),
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



func (s *XMLProcessingService) ProcessLabelsFile(
	ctx context.Context,
	filePath string,
	processingID string,
) (*ProcessingResult, error) {
	return s.processFile(ctx, filePath, processingID, "labels")
}

// Generic file processing method that handles all entity types using the parser service
func (s *XMLProcessingService) processFile(
	ctx context.Context,
	filePath string,
	processingID string,
	fileType string,
) (*ProcessingResult, error) {
	log := s.log.Function("processFile")

	log.Info(
		"Starting file processing",
		"filePath",
		filePath,
		"processingID",
		processingID,
		"fileType",
		fileType,
	)

	// Update processing status to "processing"
	if err := s.updateProcessingStatus(ctx, processingID, models.ProcessingStatusProcessing, nil); err != nil {
		log.Warn("failed to update processing status", "error", err, "processingID", processingID)
	}

	result := &ProcessingResult{
		Errors: make([]string, 0),
	}

	// Use the parser service with a progress callback
	progressFunc := func(processed, total, errors int) {
		// Report progress every PROGRESS_REPORT_INTERVAL records
		if processed%PROGRESS_REPORT_INTERVAL == 0 {
			stats := s.createProcessingStats(fileType, total, processed, errors)
			if err := s.updateProcessingStats(ctx, processingID, stats); err != nil {
				log.Warn(
					"failed to update processing stats",
					"error",
					err,
					"recordCount",
					processed,
				)
			}
			log.Info(
				"Processing progress",
				"processed",
				processed,
				"total",
				total,
				"errors",
				errors,
			)
		}
	}

	// Configure parsing options
	options := ParseOptions{
		FilePath:     filePath,
		FileType:     fileType,
		BatchSize:    XML_BATCH_SIZE,
		ProgressFunc: progressFunc,
	}

	// Use parser service to parse the file
	parseResult, err := s.parserService.ParseFile(ctx, options)
	if err != nil {
		return nil, log.Err("parsing failed", err, "filePath", filePath)
	}

	// Process entities in batches for database operations
	if err := s.processParsedResults(ctx, parseResult, fileType, result); err != nil {
		return result, err
	}

	// Update totals from parse result
	result.TotalRecords = parseResult.TotalRecords
	result.ErroredRecords = parseResult.ErroredRecords
	result.Errors = append(result.Errors, parseResult.Errors...)

	// Update final processing status
	finalStats := s.createProcessingStats(
		fileType,
		result.TotalRecords,
		result.ProcessedRecords,
		result.ErroredRecords,
	)
	status := models.ProcessingStatusCompleted

	if err := s.updateProcessingStatus(ctx, processingID, status, finalStats); err != nil {
		log.Warn("failed to update final processing status", "error", err)
	}

	log.Info("File processing completed",
		"fileType", fileType,
		"total", result.TotalRecords,
		"processed", result.ProcessedRecords,
		"inserted", result.InsertedRecords,
		"updated", result.UpdatedRecords,
		"errors", result.ErroredRecords,
	)

	return result, nil
}

// Helper method to create appropriate ProcessingStats based on entity type
func (s *XMLProcessingService) createProcessingStats(
	fileType string,
	total, processed, errors int,
) *models.ProcessingStats {
	stats := &models.ProcessingStats{
		TotalRecords:  total,
		FailedRecords: errors,
	}

	switch fileType {
	case "labels":
		stats.LabelsProcessed = processed
	case "artists":
		stats.ArtistsProcessed = processed
	case "masters":
		stats.MastersProcessed = processed
	case "releases":
		stats.ReleasesProcessed = processed
	}

	return stats
}

// Helper method to process parsed results based on entity type
func (s *XMLProcessingService) processParsedResults(
	ctx context.Context,
	parseResult *ParseResult,
	fileType string,
	result *ProcessingResult,
) error {
	switch fileType {
	case "labels":
		return s.processLabelBatches(ctx, parseResult.ParsedLabels, result)
	case "artists":
		return s.processArtistBatches(ctx, parseResult.ParsedArtists, result)
	case "masters":
		return s.processMasterBatches(ctx, parseResult.ParsedMasters, result)
	case "releases":
		return s.processReleaseBatches(ctx, parseResult.ParsedReleases, result)
	default:
		return fmt.Errorf("unsupported file type: %s", fileType)
	}
}

// Batch processing methods for each entity type
func (s *XMLProcessingService) processLabelBatches(
	ctx context.Context,
	entities []*models.Label,
	result *ProcessingResult,
) error {
	totalBatches := (len(entities) + XML_BATCH_SIZE - 1) / XML_BATCH_SIZE

	for i := 0; i < len(entities); i += XML_BATCH_SIZE {
		end := min(i+XML_BATCH_SIZE, len(entities))
		batch := entities[i:end]

		if err := s.processBatch(ctx, batch, result); err != nil {
			return fmt.Errorf("failed to process label batch: %w", err)
		}

		s.logBatchProgress(i, XML_BATCH_SIZE, totalBatches, result)
	}
	return nil
}

func (s *XMLProcessingService) processArtistBatches(
	ctx context.Context,
	entities []*models.Artist,
	result *ProcessingResult,
) error {
	totalBatches := (len(entities) + XML_BATCH_SIZE - 1) / XML_BATCH_SIZE

	for i := 0; i < len(entities); i += XML_BATCH_SIZE {
		end := min(i+XML_BATCH_SIZE, len(entities))
		batch := entities[i:end]

		if err := s.processArtistBatch(ctx, batch, result); err != nil {
			return fmt.Errorf("failed to process artist batch: %w", err)
		}

		s.logBatchProgress(i, XML_BATCH_SIZE, totalBatches, result)
	}
	return nil
}

func (s *XMLProcessingService) processMasterBatches(
	ctx context.Context,
	entities []*models.Master,
	result *ProcessingResult,
) error {
	totalBatches := (len(entities) + XML_BATCH_SIZE - 1) / XML_BATCH_SIZE

	for i := 0; i < len(entities); i += XML_BATCH_SIZE {
		end := min(i+XML_BATCH_SIZE, len(entities))
		batch := entities[i:end]

		if err := s.processMasterBatch(ctx, batch, result); err != nil {
			return fmt.Errorf("failed to process master batch: %w", err)
		}

		s.logBatchProgress(i, XML_BATCH_SIZE, totalBatches, result)
	}
	return nil
}

func (s *XMLProcessingService) processReleaseBatches(
	ctx context.Context,
	entities []*models.Release,
	result *ProcessingResult,
) error {
	totalBatches := (len(entities) + XML_BATCH_SIZE - 1) / XML_BATCH_SIZE

	for i := 0; i < len(entities); i += XML_BATCH_SIZE {
		end := min(i+XML_BATCH_SIZE, len(entities))
		batch := entities[i:end]

		if err := s.processReleaseBatch(ctx, batch, result); err != nil {
			return fmt.Errorf("failed to process release batch: %w", err)
		}

		s.logBatchProgress(i, XML_BATCH_SIZE, totalBatches, result)
	}
	return nil
}

// Helper method to log batch processing progress
func (s *XMLProcessingService) logBatchProgress(
	i, batchSize, totalBatches int,
	result *ProcessingResult,
) {
	s.log.Info(
		"Processed batch",
		"batch",
		(i/batchSize)+1,
		"totalBatches",
		totalBatches,
		"inserted",
		result.InsertedRecords,
		"updated",
		result.UpdatedRecords,
	)
}

func (s *XMLProcessingService) processBatch(
	ctx context.Context,
	labels []*models.Label,
	result *ProcessingResult,
) error {
	inserted, updated, err := s.labelRepo.UpsertBatch(ctx, labels)
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to upsert label batch: %v", err)
		result.Errors = append(result.Errors, errorMsg)
		result.ErroredRecords += len(labels)
		return err
	}

	result.ProcessedRecords += len(labels)
	result.InsertedRecords += inserted
	result.UpdatedRecords += updated

	return nil
}

func (s *XMLProcessingService) updateProcessingStatus(
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

func (s *XMLProcessingService) updateProcessingStats(
	ctx context.Context,
	processingID string,
	stats *models.ProcessingStats,
) error {
	return s.updateProcessingStatus(ctx, processingID, models.ProcessingStatusProcessing, stats)
}

func (s *XMLProcessingService) ProcessArtistsFile(
	ctx context.Context,
	filePath string,
	processingID string,
) (*ProcessingResult, error) {
	return s.processFile(ctx, filePath, processingID, "artists")
}

func (s *XMLProcessingService) processArtistBatch(
	ctx context.Context,
	artists []*models.Artist,
	result *ProcessingResult,
) error {
	inserted, updated, err := s.artistRepo.UpsertBatch(ctx, artists)
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to upsert artist batch: %v", err)
		result.Errors = append(result.Errors, errorMsg)
		result.ErroredRecords += len(artists)
		return err
	}

	result.ProcessedRecords += len(artists)
	result.InsertedRecords += inserted
	result.UpdatedRecords += updated

	return nil
}

func (s *XMLProcessingService) ProcessMastersFile(
	ctx context.Context,
	filePath string,
	processingID string,
) (*ProcessingResult, error) {
	return s.processFile(ctx, filePath, processingID, "masters")
}

func (s *XMLProcessingService) processMasterBatch(
	ctx context.Context,
	masters []*models.Master,
	result *ProcessingResult,
) error {
	inserted, updated, err := s.masterRepo.UpsertBatch(ctx, masters)
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to upsert master batch: %v", err)
		result.Errors = append(result.Errors, errorMsg)
		result.ErroredRecords += len(masters)
		return err
	}

	result.ProcessedRecords += len(masters)
	result.InsertedRecords += inserted
	result.UpdatedRecords += updated

	return nil
}

func (s *XMLProcessingService) ProcessReleasesFile(
	ctx context.Context,
	filePath string,
	processingID string,
) (*ProcessingResult, error) {
	// Use the generic parser service - it already handles tracks automatically
	return s.processFile(ctx, filePath, processingID, "releases")
}

func (s *XMLProcessingService) processReleaseBatch(
	ctx context.Context,
	releases []*models.Release,
	result *ProcessingResult,
) error {
	inserted, updated, err := s.releaseRepo.UpsertBatch(ctx, releases)
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to upsert release batch: %v", err)
		result.Errors = append(result.Errors, errorMsg)
		result.ErroredRecords += len(releases)
		return err
	}

	result.ProcessedRecords += len(releases)
	result.InsertedRecords += inserted
	result.UpdatedRecords += updated

	return nil
}



// findOrCreateGenre finds an existing genre by name or creates a new one (with caching)
func (s *XMLProcessingService) findOrCreateGenre(ctx context.Context, name string) *models.Genre {
	if name == "" {
		return nil
	}

	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return nil
	}

	// Check cache first
	if cachedGenre, exists := s.genreCache[trimmedName]; exists {
		return cachedGenre
	}

	// Use the genre repository to find or create the genre
	genre, err := s.genreRepo.FindOrCreate(ctx, trimmedName)
	if err != nil {
		s.log.Warn("Failed to find or create genre", "name", trimmedName, "error", err)
		return nil
	}

	// Cache the result for future lookups
	s.genreCache[trimmedName] = genre
	return genre
}

// findOrCreateArtist finds an existing artist by Discogs ID or creates a new one (with caching)
func (s *XMLProcessingService) findOrCreateArtist(
	ctx context.Context,
	discogsID int64,
	name string,
) *models.Artist {
	if discogsID == 0 || name == "" {
		return nil
	}

	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return nil
	}

	// Check cache first
	if cachedArtist, exists := s.artistCache[discogsID]; exists {
		return cachedArtist
	}

	// Use the artist repository to find or create the artist
	artist, err := s.artistRepo.FindOrCreateByDiscogsID(ctx, discogsID, trimmedName)
	if err != nil {
		s.log.Warn(
			"Failed to find or create artist",
			"discogsID",
			discogsID,
			"name",
			trimmedName,
			"error",
			err,
		)
		return nil
	}

	// Cache the result for future lookups
	s.artistCache[discogsID] = artist
	return artist
}
