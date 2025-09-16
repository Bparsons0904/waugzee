package services

import (
	"compress/gzip"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"waugzee/internal/imports"
	"waugzee/internal/logger"
	"waugzee/internal/models"
	"waugzee/internal/repositories"

	"github.com/google/uuid"
)

// ProcessingLimits represents limits for processing operations
type ProcessingLimits struct {
	MaxRecords   int `json:"maxRecords,omitempty"`   // Optional: limit total records parsed (0 = no limit)
	MaxBatchSize int `json:"maxBatchSize,omitempty"` // Optional: batch size for processing (default: 2000)
}

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
	transactionService        *TransactionService
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
	transactionService *TransactionService,
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
		transactionService:        transactionService,
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

// ReleaseWithTracks holds a release and its associated track data during processing
type ReleaseWithTracks struct {
	Release *models.Release
	Tracks  []imports.Track
}

// convertDiscogsTracks converts Discogs track data to our Track models
func (s *XMLProcessingService) convertDiscogsTracks(
	discogsTracks []imports.Track,
	releaseID string,
) []*models.Track {
	if len(discogsTracks) == 0 {
		return nil
	}

	releaseUUID, err := uuid.Parse(releaseID)
	if err != nil {
		s.log.Error("Failed to parse release UUID for tracks", "releaseID", releaseID, "error", err)
		return nil
	}

	tracks := make([]*models.Track, 0, len(discogsTracks))

	for _, discogsTrack := range discogsTracks {
		// Skip tracks with no title
		title := strings.TrimSpace(discogsTrack.Title)
		if len(title) == 0 {
			continue
		}

		// Skip tracks with no position
		position := strings.TrimSpace(discogsTrack.Position)
		if len(position) == 0 {
			continue
		}

		track := &models.Track{
			ReleaseID: releaseUUID,
			Position:  position,
			Title:     title,
		}

		// Parse duration if present (format examples: "3:45", "2:30", "10:15")
		if len(discogsTrack.Duration) > 0 {
			duration := s.parseDuration(discogsTrack.Duration)
			if duration > 0 {
				track.Duration = &duration
			}
		}

		tracks = append(tracks, track)
	}

	return tracks
}

// parseDuration converts a duration string like "3:45" to seconds
func (s *XMLProcessingService) parseDuration(durationStr string) int {
	durationStr = strings.TrimSpace(durationStr)
	if len(durationStr) == 0 {
		return 0
	}

	// Handle formats like "3:45", "12:30", "1:23:45" (hours:minutes:seconds)
	parts := strings.Split(durationStr, ":")
	if len(parts) < 2 {
		return 0
	}

	var totalSeconds int

	if len(parts) == 2 {
		// MM:SS format
		minutes, err := strconv.Atoi(parts[0])
		if err != nil {
			return 0
		}
		seconds, err := strconv.Atoi(parts[1])
		if err != nil {
			return 0
		}
		totalSeconds = minutes*60 + seconds
	} else if len(parts) == 3 {
		// HH:MM:SS format
		hours, err := strconv.Atoi(parts[0])
		if err != nil {
			return 0
		}
		minutes, err := strconv.Atoi(parts[1])
		if err != nil {
			return 0
		}
		seconds, err := strconv.Atoi(parts[2])
		if err != nil {
			return 0
		}
		totalSeconds = hours*3600 + minutes*60 + seconds
	}

	return totalSeconds
}

// processTracksForRelease creates and saves tracks for a specific release
func (s *XMLProcessingService) processTracksForRelease(
	ctx context.Context,
	discogsTracks []imports.Track,
	releaseID string,
) error {
	if len(discogsTracks) == 0 {
		return nil
	}

	tracks := s.convertDiscogsTracks(discogsTracks, releaseID)
	if len(tracks) == 0 {
		return nil
	}

	// Delete existing tracks for this release first (in case of re-processing)
	if err := s.trackRepo.DeleteByReleaseID(ctx, releaseID); err != nil {
		return fmt.Errorf("failed to delete existing tracks for release %s: %w", releaseID, err)
	}

	// Create new tracks
	if err := s.trackRepo.CreateBatch(ctx, tracks); err != nil {
		return fmt.Errorf("failed to create tracks for release %s: %w", releaseID, err)
	}

	return nil
}

// processReleaseBatchWithTracks processes releases and their associated tracks
func (s *XMLProcessingService) processReleaseBatchWithTracks(
	ctx context.Context,
	releasesWithTracks []ReleaseWithTracks,
	result *ProcessingResult,
) error {
	log := s.log.Function("processReleaseBatchWithTracks")

	if len(releasesWithTracks) == 0 {
		return nil
	}

	// Extract just the releases for batch upsertion
	releases := make([]*models.Release, len(releasesWithTracks))
	for i, rwt := range releasesWithTracks {
		releases[i] = rwt.Release
	}

	// First, upsert all releases
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

	// Now process tracks for each release
	var trackErrors []string
	for _, rwt := range releasesWithTracks {
		if len(rwt.Tracks) > 0 {
			if err := s.processTracksForRelease(ctx, rwt.Tracks, rwt.Release.ID.String()); err != nil {
				errorMsg := fmt.Sprintf(
					"Failed to process tracks for release %s: %v",
					rwt.Release.ID,
					err,
				)
				trackErrors = append(trackErrors, errorMsg)
				log.Warn("Track processing failed", "releaseID", rwt.Release.ID, "error", err)
			}
		}
	}

	// Add track errors to results
	if len(trackErrors) > 0 {
		result.Errors = append(result.Errors, trackErrors...)
		log.Info("Track processing completed with errors",
			"totalReleases", len(releasesWithTracks),
			"trackErrors", len(trackErrors))
	} else {
		log.Info("Track processing completed successfully",
			"totalReleases", len(releasesWithTracks))
	}

	return nil
}

// Wrapper methods that accept limits

// ProcessLabelsFileWithLimits wraps ProcessLabelsFile with support for processing limits
func (s *XMLProcessingService) ProcessLabelsFileWithLimits(
	ctx context.Context,
	filePath string,
	processingID string,
	limits *ProcessingLimits,
) (*ProcessingResult, error) {
	// ProcessLabelsFile already uses the parser service which supports limits
	// We need to pass the limits to the parser service through the existing method
	return s.processLabelsFileWithLimits(ctx, filePath, processingID, limits)
}

// ProcessArtistsFileWithLimits wraps ProcessArtistsFile with support for processing limits
func (s *XMLProcessingService) ProcessArtistsFileWithLimits(
	ctx context.Context,
	filePath string,
	processingID string,
	limits *ProcessingLimits,
) (*ProcessingResult, error) {
	return s.processArtistsFileWithLimits(ctx, filePath, processingID, limits)
}

// ProcessMastersFileWithLimits wraps ProcessMastersFile with support for processing limits
func (s *XMLProcessingService) ProcessMastersFileWithLimits(
	ctx context.Context,
	filePath string,
	processingID string,
	limits *ProcessingLimits,
) (*ProcessingResult, error) {
	return s.processMastersFileWithLimits(ctx, filePath, processingID, limits)
}

// ProcessReleasesFileWithLimits wraps ProcessReleasesFile with support for processing limits
func (s *XMLProcessingService) ProcessReleasesFileWithLimits(
	ctx context.Context,
	filePath string,
	processingID string,
	limits *ProcessingLimits,
) (*ProcessingResult, error) {
	return s.processReleasesFileWithLimits(ctx, filePath, processingID, limits)
}

// ProcessReleasesFileWithTracksAndLimits wraps ProcessReleasesFileWithTracks with support for processing limits
func (s *XMLProcessingService) ProcessReleasesFileWithTracksAndLimits(
	ctx context.Context,
	filePath string,
	processingID string,
	limits *ProcessingLimits,
) (*ProcessingResult, error) {
	return s.processReleasesFileWithTracksAndLimits(ctx, filePath, processingID, limits)
}

func (s *XMLProcessingService) ProcessLabelsFile(
	ctx context.Context,
	filePath string,
	processingID string,
) (*ProcessingResult, error) {
	log := s.log.Function("ProcessLabelsFile")

	log.Info("Starting labels file processing", "filePath", filePath, "processingID", processingID)

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
			stats := &models.ProcessingStats{
				TotalRecords:    total,
				LabelsProcessed: processed,
				FailedRecords:   errors,
			}
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
		FileType:     "labels",
		BatchSize:    XML_BATCH_SIZE,
		ProgressFunc: progressFunc,
	}

	// Use parser service to parse the file
	parseResult, err := s.parserService.ParseFile(ctx, options)
	if err != nil {
		return nil, log.Err("parsing failed", err, "filePath", filePath)
	}

	// Process labels in batches for database operations
	labels := parseResult.ParsedLabels
	totalBatches := (len(labels) + XML_BATCH_SIZE - 1) / XML_BATCH_SIZE

	for i := 0; i < len(labels); i += XML_BATCH_SIZE {
		end := min(i+XML_BATCH_SIZE, len(labels))

		batch := labels[i:end]
		if err := s.processBatch(ctx, batch, result); err != nil {
			log.Err("failed to process label batch", err, "batchSize", len(batch))
			return result, err
		}

		log.Info(
			"Processed batch",
			"batch",
			(i/XML_BATCH_SIZE)+1,
			"totalBatches",
			totalBatches,
			"inserted",
			result.InsertedRecords,
			"updated",
			result.UpdatedRecords,
		)
	}

	// Update totals from parse result
	result.TotalRecords = parseResult.TotalRecords
	result.ErroredRecords = parseResult.ErroredRecords
	result.Errors = append(result.Errors, parseResult.Errors...)

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

// Implementation methods with limits support

func (s *XMLProcessingService) processLabelsFileWithLimits(
	ctx context.Context,
	filePath string,
	processingID string,
	limits *ProcessingLimits,
) (*ProcessingResult, error) {
	log := s.log.Function("processLabelsFileWithLimits")

	log.Info(
		"Starting labels file processing with limits",
		"filePath",
		filePath,
		"processingID",
		processingID,
		"limits",
		limits,
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
			stats := &models.ProcessingStats{
				TotalRecords:    total,
				LabelsProcessed: processed,
				FailedRecords:   errors,
			}
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

	// Configure parsing options with limits
	options := ParseOptions{
		FilePath:     filePath,
		FileType:     "labels",
		BatchSize:    XML_BATCH_SIZE,
		ProgressFunc: progressFunc,
	}

	// Apply limits if provided
	if limits != nil {
		if limits.MaxRecords > 0 {
			options.MaxRecords = limits.MaxRecords
		}
		if limits.MaxBatchSize > 0 {
			options.BatchSize = limits.MaxBatchSize
		}
	}

	// Use parser service to parse the file
	parseResult, err := s.parserService.ParseFile(ctx, options)
	if err != nil {
		return nil, log.Err("parsing failed", err, "filePath", filePath)
	}

	// Process labels in batches for database operations
	labels := parseResult.ParsedLabels
	batchSize := XML_BATCH_SIZE
	if limits != nil && limits.MaxBatchSize > 0 {
		batchSize = limits.MaxBatchSize
	}
	totalBatches := (len(labels) + batchSize - 1) / batchSize

	for i := 0; i < len(labels); i += batchSize {
		end := min(i+batchSize, len(labels))

		batch := labels[i:end]
		if err := s.processBatch(ctx, batch, result); err != nil {
			log.Err("failed to process label batch", err, "batchSize", len(batch))
			return result, err
		}

		log.Info(
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

	// Update totals from parse result
	result.TotalRecords = parseResult.TotalRecords
	result.ErroredRecords = parseResult.ErroredRecords
	result.Errors = append(result.Errors, parseResult.Errors...)

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

func (s *XMLProcessingService) processArtistsFileWithLimits(
	ctx context.Context,
	filePath string,
	processingID string,
	limits *ProcessingLimits,
) (*ProcessingResult, error) {
	log := s.log.Function("processArtistsFileWithLimits")

	log.Info(
		"Starting artists file processing with limits",
		"filePath",
		filePath,
		"processingID",
		processingID,
		"limits",
		limits,
	)

	// Configure parsing options with limits
	options := ParseOptions{
		FilePath:  filePath,
		FileType:  "artists",
		BatchSize: XML_BATCH_SIZE,
	}

	// Apply limits if provided
	if limits != nil {
		if limits.MaxRecords > 0 {
			options.MaxRecords = limits.MaxRecords
		}
		if limits.MaxBatchSize > 0 {
			options.BatchSize = limits.MaxBatchSize
		}
	}

	// Use parser service to parse the file
	parseResult, err := s.parserService.ParseFile(ctx, options)
	if err != nil {
		return nil, log.Err("parsing failed", err, "filePath", filePath)
	}

	result := &ProcessingResult{
		Errors:         make([]string, 0),
		TotalRecords:   parseResult.TotalRecords,
		ErroredRecords: parseResult.ErroredRecords,
	}
	result.Errors = append(result.Errors, parseResult.Errors...)

	// Update processing status to "processing"
	if err := s.updateProcessingStatus(ctx, processingID, models.ProcessingStatusProcessing, nil); err != nil {
		log.Warn("failed to update processing status", "error", err, "processingID", processingID)
	}

	// Process artists in batches for database operations
	artists := parseResult.ParsedArtists
	batchSize := XML_BATCH_SIZE
	if limits != nil && limits.MaxBatchSize > 0 {
		batchSize = limits.MaxBatchSize
	}
	totalBatches := (len(artists) + batchSize - 1) / batchSize

	for i := 0; i < len(artists); i += batchSize {
		end := i + batchSize
		if end > len(artists) {
			end = len(artists)
		}

		batch := artists[i:end]
		if err := s.processArtistBatch(ctx, batch, result); err != nil {
			log.Err("failed to process artist batch", err, "batchSize", len(batch))
			return result, err
		}

		log.Info(
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

	// Update final processing status
	finalStats := &models.ProcessingStats{
		TotalRecords:     result.TotalRecords,
		ArtistsProcessed: result.ProcessedRecords,
		FailedRecords:    result.ErroredRecords,
	}

	status := models.ProcessingStatusCompleted

	if err := s.updateProcessingStatus(ctx, processingID, status, finalStats); err != nil {
		log.Warn("failed to update final processing status", "error", err)
	}

	log.Info("Artists file processing completed",
		"total", result.TotalRecords,
		"processed", result.ProcessedRecords,
		"inserted", result.InsertedRecords,
		"updated", result.UpdatedRecords,
		"errors", result.ErroredRecords,
	)

	return result, nil
}

func (s *XMLProcessingService) processMastersFileWithLimits(
	ctx context.Context,
	filePath string,
	processingID string,
	limits *ProcessingLimits,
) (*ProcessingResult, error) {
	log := s.log.Function("processMastersFileWithLimits")

	log.Info(
		"Starting masters file processing with limits",
		"filePath",
		filePath,
		"processingID",
		processingID,
		"limits",
		limits,
	)

	// Configure parsing options with limits
	options := ParseOptions{
		FilePath:  filePath,
		FileType:  "masters",
		BatchSize: XML_BATCH_SIZE,
	}

	// Apply limits if provided
	if limits != nil {
		if limits.MaxRecords > 0 {
			options.MaxRecords = limits.MaxRecords
		}
		if limits.MaxBatchSize > 0 {
			options.BatchSize = limits.MaxBatchSize
		}
	}

	// Use parser service to parse the file
	parseResult, err := s.parserService.ParseFile(ctx, options)
	if err != nil {
		return nil, log.Err("parsing failed", err, "filePath", filePath)
	}

	result := &ProcessingResult{
		Errors:         make([]string, 0),
		TotalRecords:   parseResult.TotalRecords,
		ErroredRecords: parseResult.ErroredRecords,
	}
	result.Errors = append(result.Errors, parseResult.Errors...)

	// Update processing status to "processing"
	if err := s.updateProcessingStatus(ctx, processingID, models.ProcessingStatusProcessing, nil); err != nil {
		log.Warn("failed to update processing status", "error", err, "processingID", processingID)
	}

	// Process masters in batches for database operations
	masters := parseResult.ParsedMasters
	batchSize := XML_BATCH_SIZE
	if limits != nil && limits.MaxBatchSize > 0 {
		batchSize = limits.MaxBatchSize
	}
	totalBatches := (len(masters) + batchSize - 1) / batchSize

	for i := 0; i < len(masters); i += batchSize {
		end := i + batchSize
		if end > len(masters) {
			end = len(masters)
		}

		batch := masters[i:end]
		if err := s.processMasterBatch(ctx, batch, result); err != nil {
			log.Err("failed to process master batch", err, "batchSize", len(batch))
			return result, err
		}

		log.Info(
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

	// Update final processing status
	finalStats := &models.ProcessingStats{
		TotalRecords:     result.TotalRecords,
		MastersProcessed: result.ProcessedRecords,
		FailedRecords:    result.ErroredRecords,
	}

	status := models.ProcessingStatusCompleted

	if err := s.updateProcessingStatus(ctx, processingID, status, finalStats); err != nil {
		log.Warn("failed to update final processing status", "error", err)
	}

	log.Info("Masters file processing completed",
		"total", result.TotalRecords,
		"processed", result.ProcessedRecords,
		"inserted", result.InsertedRecords,
		"updated", result.UpdatedRecords,
		"errors", result.ErroredRecords,
	)

	return result, nil
}

func (s *XMLProcessingService) processReleasesFileWithLimits(
	ctx context.Context,
	filePath string,
	processingID string,
	limits *ProcessingLimits,
) (*ProcessingResult, error) {
	log := s.log.Function("processReleasesFileWithLimits")

	log.Info(
		"Starting releases file processing with limits",
		"filePath",
		filePath,
		"processingID",
		processingID,
		"limits",
		limits,
	)

	// Configure parsing options with limits
	options := ParseOptions{
		FilePath:  filePath,
		FileType:  "releases",
		BatchSize: XML_BATCH_SIZE,
	}

	// Apply limits if provided
	if limits != nil {
		if limits.MaxRecords > 0 {
			options.MaxRecords = limits.MaxRecords
		}
		if limits.MaxBatchSize > 0 {
			options.BatchSize = limits.MaxBatchSize
		}
	}

	// Use parser service to parse the file
	parseResult, err := s.parserService.ParseFile(ctx, options)
	if err != nil {
		return nil, log.Err("parsing failed", err, "filePath", filePath)
	}

	result := &ProcessingResult{
		Errors:         make([]string, 0),
		TotalRecords:   parseResult.TotalRecords,
		ErroredRecords: parseResult.ErroredRecords,
	}
	result.Errors = append(result.Errors, parseResult.Errors...)

	// Update processing status to "processing"
	if err := s.updateProcessingStatus(ctx, processingID, models.ProcessingStatusProcessing, nil); err != nil {
		log.Warn("failed to update processing status", "error", err, "processingID", processingID)
	}

	// Process releases in batches for database operations
	releases := parseResult.ParsedReleases
	batchSize := XML_BATCH_SIZE
	if limits != nil && limits.MaxBatchSize > 0 {
		batchSize = limits.MaxBatchSize
	}
	totalBatches := (len(releases) + batchSize - 1) / batchSize

	for i := 0; i < len(releases); i += batchSize {
		end := i + batchSize
		if end > len(releases) {
			end = len(releases)
		}

		batch := releases[i:end]
		if err := s.processReleaseBatch(ctx, batch, result); err != nil {
			log.Err("failed to process release batch", err, "batchSize", len(batch))
			return result, err
		}

		log.Info(
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

	// Update final processing status
	finalStats := &models.ProcessingStats{
		TotalRecords:      result.TotalRecords,
		ReleasesProcessed: result.ProcessedRecords,
		FailedRecords:     result.ErroredRecords,
	}

	status := models.ProcessingStatusCompleted

	if err := s.updateProcessingStatus(ctx, processingID, status, finalStats); err != nil {
		log.Warn("failed to update final processing status", "error", err)
	}

	log.Info("Releases file processing completed",
		"total", result.TotalRecords,
		"processed", result.ProcessedRecords,
		"inserted", result.InsertedRecords,
		"updated", result.UpdatedRecords,
		"errors", result.ErroredRecords,
	)

	return result, nil
}

func (s *XMLProcessingService) processReleasesFileWithTracksAndLimits(
	ctx context.Context,
	filePath string,
	processingID string,
	limits *ProcessingLimits,
) (*ProcessingResult, error) {
	log := s.log.Function("processReleasesFileWithTracksAndLimits")

	log.Info(
		"Starting releases file processing with tracks and limits",
		"filePath",
		filePath,
		"processingID",
		processingID,
		"limits",
		limits,
	)

	// For releases with tracks, we need to use a different approach since the parser service doesn't handle tracks
	// We'll modify the existing ProcessReleasesFileWithTracks to respect limits

	// Open and decompress the gzipped XML file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, log.Err("failed to open releases file", err, "filePath", filePath)
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

	var releaseWithTracksBatch []ReleaseWithTracks
	var recordCount int
	batchSize := XML_BATCH_SIZE
	if limits != nil && limits.MaxBatchSize > 0 {
		batchSize = limits.MaxBatchSize
	}

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

		// Look for release start elements
		if startElement, ok := token.(xml.StartElement); ok &&
			startElement.Name.Local == "release" {
			result.TotalRecords++

			// Check max records limit before processing
			if limits != nil && limits.MaxRecords > 0 && result.TotalRecords > limits.MaxRecords {
				log.Info("Reached max records limit - stopping early",
					"maxRecords", limits.MaxRecords,
					"totalAttempted", result.TotalRecords-1)
				result.TotalRecords-- // Adjust count since we're not processing this record
				break
			}

			var discogsRelease imports.Release
			if err := decoder.DecodeElement(&discogsRelease, &startElement); err != nil {
				errorMsg := fmt.Sprintf("Failed to decode release element: %v", err)
				result.Errors = append(result.Errors, errorMsg)
				result.ErroredRecords++
				log.Warn("Failed to decode release", "error", err)
				continue
			}

			// Convert Discogs release to our release model
			release := s.convertDiscogsRelease(ctx, &discogsRelease)
			if release == nil {
				result.ErroredRecords++
				continue
			}

			// Create ReleaseWithTracks including track data
			releaseWithTracks := ReleaseWithTracks{
				Release: release,
				Tracks:  discogsRelease.Tracklist,
			}

			releaseWithTracksBatch = append(releaseWithTracksBatch, releaseWithTracks)
			recordCount++

			// Process batch when it reaches the limit
			if len(releaseWithTracksBatch) >= batchSize {
				if err := s.processReleaseBatchWithTracks(ctx, releaseWithTracksBatch, result); err != nil {
					log.Err(
						"failed to process release batch with tracks",
						err,
						"batchSize",
						len(releaseWithTracksBatch),
					)
					return result, err
				}
				releaseWithTracksBatch = releaseWithTracksBatch[:0] // Reset batch
			}

			// Report progress every PROGRESS_REPORT_INTERVAL records
			if recordCount%PROGRESS_REPORT_INTERVAL == 0 {
				stats := &models.ProcessingStats{
					TotalRecords:      result.TotalRecords,
					ReleasesProcessed: result.ProcessedRecords,
					FailedRecords:     result.ErroredRecords,
				}
				if err := s.updateProcessingStats(ctx, processingID, stats); err != nil {
					log.Warn(
						"failed to update processing stats",
						"error",
						err,
						"recordCount",
						recordCount,
					)
				}
				log.Info(
					"Processing progress",
					"processed",
					recordCount,
					"inserted",
					result.InsertedRecords,
					"updated",
					result.UpdatedRecords,
					"errors",
					result.ErroredRecords,
				)
			}
		}
	}

	// Process remaining batch
	if len(releaseWithTracksBatch) > 0 {
		if err := s.processReleaseBatchWithTracks(ctx, releaseWithTracksBatch, result); err != nil {
			log.Err(
				"failed to process final release batch with tracks",
				err,
				"batchSize",
				len(releaseWithTracksBatch),
			)
			return result, err
		}
	}

	// Update final processing status
	finalStats := &models.ProcessingStats{
		TotalRecords:      result.TotalRecords,
		ReleasesProcessed: result.ProcessedRecords,
		FailedRecords:     result.ErroredRecords,
	}
	status := models.ProcessingStatusCompleted
	if err := s.updateProcessingStatus(ctx, processingID, status, finalStats); err != nil {
		log.Warn("failed to update final processing status", "error", err)
	}

	log.Info("Releases file processing with tracks and limits completed",
		"total", result.TotalRecords,
		"processed", result.ProcessedRecords,
		"inserted", result.InsertedRecords,
		"updated", result.UpdatedRecords,
		"errors", result.ErroredRecords,
	)

	return result, nil
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

func (s *XMLProcessingService) convertDiscogsLabel(discogsLabel *imports.Label) *models.Label {
	// Skip labels with invalid data (avoid string ops on invalid data)
	if discogsLabel.ID == 0 || len(discogsLabel.Name) == 0 {
		return nil
	}

	// Single trim operation with length check
	name := strings.TrimSpace(discogsLabel.Name)
	if len(name) == 0 {
		return nil
	}

	label := &models.Label{Name: name}

	// Set Discogs ID (avoid pointer allocation for primitive)
	discogsID := int64(discogsLabel.ID)
	label.DiscogsID = &discogsID

	// Only process optional fields if they contain data
	// Removed unused ContactInfo and Profile processing to reduce allocations

	return label
}

func (s *XMLProcessingService) updateProcessingStatus(
	ctx context.Context,
	processingID string,
	status models.ProcessingStatus,
	stats *models.ProcessingStats,
) error {
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
	log := s.log.Function("ProcessArtistsFile")

	log.Info("Starting artists file processing", "filePath", filePath, "processingID", processingID)

	// Open and decompress the gzipped XML file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, log.Err("failed to open artists file", err, "filePath", filePath)
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

	var artistBatch []*models.Artist
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

		// Look for artist start elements
		if startElement, ok := token.(xml.StartElement); ok && startElement.Name.Local == "artist" {
			var discogsArtist imports.Artist
			if err := decoder.DecodeElement(&discogsArtist, &startElement); err != nil {
				errorMsg := fmt.Sprintf("Failed to decode artist element: %v", err)
				result.Errors = append(result.Errors, errorMsg)
				result.ErroredRecords++
				log.Warn("Failed to decode artist", "error", err)
				continue
			}

			// Convert Discogs artist to our artist model
			artist := s.convertDiscogsArtist(ctx, &discogsArtist)
			if artist == nil {
				result.ErroredRecords++
				continue
			}

			artistBatch = append(artistBatch, artist)
			recordCount++
			result.TotalRecords++

			// Process batch when it reaches the limit
			if len(artistBatch) >= XML_BATCH_SIZE {
				if err := s.processArtistBatch(ctx, artistBatch, result); err != nil {
					log.Err("failed to process artist batch", err, "batchSize", len(artistBatch))
					return result, err
				}
				artistBatch = artistBatch[:0] // Reset batch
			}

			// Report progress every PROGRESS_REPORT_INTERVAL records
			if recordCount%PROGRESS_REPORT_INTERVAL == 0 {
				stats := &models.ProcessingStats{
					TotalRecords:     result.TotalRecords,
					ArtistsProcessed: result.ProcessedRecords,
					FailedRecords:    result.ErroredRecords,
				}
				if err := s.updateProcessingStats(ctx, processingID, stats); err != nil {
					log.Warn(
						"failed to update processing stats",
						"error",
						err,
						"recordCount",
						recordCount,
					)
				}
				log.Info(
					"Processing progress",
					"processed",
					recordCount,
					"inserted",
					result.InsertedRecords,
					"updated",
					result.UpdatedRecords,
					"errors",
					result.ErroredRecords,
				)
			}
		}
	}

	// Process remaining batch
	if len(artistBatch) > 0 {
		if err := s.processArtistBatch(ctx, artistBatch, result); err != nil {
			log.Err("failed to process final artist batch", err, "batchSize", len(artistBatch))
			return result, err
		}
	}

	// Update final processing status
	finalStats := &models.ProcessingStats{
		TotalRecords:     result.TotalRecords,
		ArtistsProcessed: result.ProcessedRecords,
		FailedRecords:    result.ErroredRecords,
	}

	status := models.ProcessingStatusCompleted

	if err := s.updateProcessingStatus(ctx, processingID, status, finalStats); err != nil {
		log.Warn("failed to update final processing status", "error", err)
	}

	log.Info("Artists file processing completed",
		"total", result.TotalRecords,
		"processed", result.ProcessedRecords,
		"inserted", result.InsertedRecords,
		"updated", result.UpdatedRecords,
		"errors", result.ErroredRecords,
	)

	return result, nil
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

func (s *XMLProcessingService) convertDiscogsArtist(
	ctx context.Context,
	discogsArtist *imports.Artist,
) *models.Artist {
	// Skip artists with invalid data (avoid string ops on invalid data)
	if discogsArtist.ID == 0 || len(discogsArtist.Name) == 0 {
		return nil
	}

	// Single trim operation with length check
	name := strings.TrimSpace(discogsArtist.Name)
	if len(name) == 0 {
		return nil
	}

	// Use findOrCreateArtist to get existing artist or create new one
	discogsID := int64(discogsArtist.ID)
	artist := s.findOrCreateArtist(ctx, discogsID, name)
	if artist == nil {
		return nil
	}

	// Only process biography if we have real name or profile data
	if len(discogsArtist.RealName) > 0 {
		realName := strings.TrimSpace(discogsArtist.RealName)
		if len(realName) > 0 {
			biography := "Real name: " + realName
			if len(discogsArtist.Profile) > 0 {
				if profile := strings.TrimSpace(discogsArtist.Profile); len(profile) > 0 {
					biography += "\n\n" + profile
				}
			}
			artist.Biography = &biography
		}
	} else if len(discogsArtist.Profile) > 0 {
		if profile := strings.TrimSpace(discogsArtist.Profile); len(profile) > 0 {
			artist.Biography = &profile
		}
	}

	// Process images if available (even if currently empty in XML dumps)
	// This sets up infrastructure for future API integration or XML dump improvements
	if len(discogsArtist.Images) > 0 {
		for _, discogsImage := range discogsArtist.Images {
			if image := s.convertDiscogsImage(&discogsImage, artist.ID.String(), models.ImageableTypeArtist); image != nil {
				artist.Images = append(artist.Images, *image)
			}
		}
	}

	return artist
}

func (s *XMLProcessingService) convertDiscogsImage(
	discogsImage *imports.DiscogsImage,
	imageableID, imageableType string,
) *models.Image {
	// Skip images with no URI (common in current XML dumps)
	if len(discogsImage.URI) == 0 {
		return nil
	}

	// Determine image type based on Discogs type
	imageType := models.ImageTypePrimary
	if discogsImage.Type == "secondary" {
		imageType = models.ImageTypeSecondary
	}

	image := &models.Image{
		URL:           discogsImage.URI,
		ImageableID:   imageableID,
		ImageableType: imageableType,
		ImageType:     imageType,
	}

	// Set optional fields if available
	if discogsImage.Width > 0 {
		image.Width = &discogsImage.Width
	}
	if discogsImage.Height > 0 {
		image.Height = &discogsImage.Height
	}
	if len(discogsImage.URI150) > 0 {
		image.DiscogsURI150 = &discogsImage.URI150
	}
	if len(discogsImage.Type) > 0 {
		image.DiscogsType = &discogsImage.Type
	}

	// Store original Discogs URI for reference
	image.DiscogsURI = &discogsImage.URI

	return image
}

func (s *XMLProcessingService) ProcessMastersFile(
	ctx context.Context,
	filePath string,
	processingID string,
) (*ProcessingResult, error) {
	log := s.log.Function("ProcessMastersFile")

	log.Info("Starting masters file processing", "filePath", filePath, "processingID", processingID)

	// Preload entity caches for performance
	if err := s.preloadEntityCaches(ctx); err != nil {
		log.Warn("Failed to preload entity caches, continuing without cache", "error", err)
	}

	// Open and decompress the gzipped XML file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, log.Err("failed to open masters file", err, "filePath", filePath)
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

	var masterBatch []*models.Master
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

		// Look for master start elements
		if startElement, ok := token.(xml.StartElement); ok && startElement.Name.Local == "master" {
			var discogsMaster imports.Master
			if err := decoder.DecodeElement(&discogsMaster, &startElement); err != nil {
				errorMsg := fmt.Sprintf("Failed to decode master element: %v", err)
				result.Errors = append(result.Errors, errorMsg)
				result.ErroredRecords++
				log.Warn("Failed to decode master", "error", err)
				continue
			}

			// Convert Discogs master to our master model
			master := s.convertDiscogsMaster(ctx, &discogsMaster)
			if master == nil {
				result.ErroredRecords++
				continue
			}

			masterBatch = append(masterBatch, master)
			recordCount++
			result.TotalRecords++

			// Process batch when it reaches the limit
			if len(masterBatch) >= XML_BATCH_SIZE {
				if err := s.processMasterBatch(ctx, masterBatch, result); err != nil {
					log.Err("failed to process master batch", err, "batchSize", len(masterBatch))
					return result, err
				}
				masterBatch = masterBatch[:0] // Reset batch
			}

			// Report progress every PROGRESS_REPORT_INTERVAL records
			if recordCount%PROGRESS_REPORT_INTERVAL == 0 {
				stats := &models.ProcessingStats{
					TotalRecords:     result.TotalRecords,
					MastersProcessed: result.ProcessedRecords,
					FailedRecords:    result.ErroredRecords,
				}
				if err := s.updateProcessingStats(ctx, processingID, stats); err != nil {
					log.Warn(
						"failed to update processing stats",
						"error",
						err,
						"recordCount",
						recordCount,
					)
				}
				log.Info(
					"Processing progress",
					"processed",
					recordCount,
					"inserted",
					result.InsertedRecords,
					"updated",
					result.UpdatedRecords,
					"errors",
					result.ErroredRecords,
				)
			}
		}
	}

	// Process remaining batch
	if len(masterBatch) > 0 {
		if err := s.processMasterBatch(ctx, masterBatch, result); err != nil {
			log.Err("failed to process final master batch", err, "batchSize", len(masterBatch))
			return result, err
		}
	}

	// Update final processing status
	finalStats := &models.ProcessingStats{
		TotalRecords:     result.TotalRecords,
		MastersProcessed: result.ProcessedRecords,
		FailedRecords:    result.ErroredRecords,
	}

	status := models.ProcessingStatusCompleted

	if err := s.updateProcessingStatus(ctx, processingID, status, finalStats); err != nil {
		log.Warn("failed to update final processing status", "error", err)
	}

	log.Info("Masters file processing completed",
		"total", result.TotalRecords,
		"processed", result.ProcessedRecords,
		"inserted", result.InsertedRecords,
		"updated", result.UpdatedRecords,
		"errors", result.ErroredRecords,
	)

	return result, nil
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

func (s *XMLProcessingService) convertDiscogsMaster(
	ctx context.Context,
	discogsMaster *imports.Master,
) *models.Master {
	// Skip masters with invalid data (avoid string ops on invalid data)
	if discogsMaster.ID == 0 || len(discogsMaster.Title) == 0 {
		return nil
	}

	// Single trim operation with length check
	title := strings.TrimSpace(discogsMaster.Title)
	if len(title) == 0 {
		return nil
	}

	master := &models.Master{Title: title}

	// Set Discogs ID
	discogsID := int64(discogsMaster.ID)
	master.DiscogsID = &discogsID

	// Set optional fields only if they have values
	if discogsMaster.MainRelease != 0 {
		mainRelease := int64(discogsMaster.MainRelease)
		master.MainRelease = &mainRelease
	}

	if discogsMaster.Year != 0 {
		master.Year = &discogsMaster.Year
	}

	if len(discogsMaster.Notes) > 0 {
		if notes := strings.TrimSpace(discogsMaster.Notes); len(notes) > 0 {
			master.Notes = &notes
		}
	}

	if len(discogsMaster.DataQuality) > 0 {
		if dataQuality := strings.TrimSpace(discogsMaster.DataQuality); len(dataQuality) > 0 {
			master.DataQuality = &dataQuality
		}
	}

	// Convert genres
	for _, genreName := range discogsMaster.Genres {
		if genre := s.findOrCreateGenre(ctx, genreName); genre != nil {
			master.Genres = append(master.Genres, *genre)
		}
	}

	// Convert styles as genres (Discogs treats them as sub-genres)
	for _, styleName := range discogsMaster.Styles {
		if genre := s.findOrCreateGenre(ctx, styleName); genre != nil {
			master.Genres = append(master.Genres, *genre)
		}
	}

	// Convert artists
	for _, discogsArtist := range discogsMaster.Artists {
		if artist := s.convertDiscogsArtist(ctx, &discogsArtist); artist != nil {
			master.Artists = append(master.Artists, *artist)
		}
	}

	return master
}

func (s *XMLProcessingService) ProcessReleasesFile(
	ctx context.Context,
	filePath string,
	processingID string,
) (*ProcessingResult, error) {
	log := s.log.Function("ProcessReleasesFile")

	log.Info(
		"Starting releases file processing",
		"filePath",
		filePath,
		"processingID",
		processingID,
	)

	// Open and decompress the gzipped XML file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, log.Err("failed to open releases file", err, "filePath", filePath)
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

	var releaseBatch []*models.Release
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

		// Look for release start elements
		if startElement, ok := token.(xml.StartElement); ok &&
			startElement.Name.Local == "release" {
			var discogsRelease imports.Release
			if err := decoder.DecodeElement(&discogsRelease, &startElement); err != nil {
				errorMsg := fmt.Sprintf("Failed to decode release element: %v", err)
				result.Errors = append(result.Errors, errorMsg)
				result.ErroredRecords++
				log.Warn("Failed to decode release", "error", err)
				continue
			}

			// Convert Discogs release to our release model
			release := s.convertDiscogsRelease(ctx, &discogsRelease)
			if release == nil {
				result.ErroredRecords++
				continue
			}

			releaseBatch = append(releaseBatch, release)
			recordCount++
			result.TotalRecords++

			// Process batch when it reaches the limit
			if len(releaseBatch) >= XML_BATCH_SIZE {
				if err := s.processReleaseBatch(ctx, releaseBatch, result); err != nil {
					log.Err("failed to process release batch", err, "batchSize", len(releaseBatch))
					return result, err
				}
				releaseBatch = releaseBatch[:0] // Reset batch
			}

			// Report progress every PROGRESS_REPORT_INTERVAL records
			if recordCount%PROGRESS_REPORT_INTERVAL == 0 {
				stats := &models.ProcessingStats{
					TotalRecords:      result.TotalRecords,
					ReleasesProcessed: result.ProcessedRecords,
					FailedRecords:     result.ErroredRecords,
				}
				if err := s.updateProcessingStats(ctx, processingID, stats); err != nil {
					log.Warn(
						"failed to update processing stats",
						"error",
						err,
						"recordCount",
						recordCount,
					)
				}
				log.Info(
					"Processing progress",
					"processed",
					recordCount,
					"inserted",
					result.InsertedRecords,
					"updated",
					result.UpdatedRecords,
					"errors",
					result.ErroredRecords,
				)
			}
		}
	}

	// Process remaining batch
	if len(releaseBatch) > 0 {
		if err := s.processReleaseBatch(ctx, releaseBatch, result); err != nil {
			log.Err("failed to process final release batch", err, "batchSize", len(releaseBatch))
			return result, err
		}
	}

	// Update final processing status
	finalStats := &models.ProcessingStats{
		TotalRecords:      result.TotalRecords,
		ReleasesProcessed: result.ProcessedRecords,
		FailedRecords:     result.ErroredRecords,
	}

	status := models.ProcessingStatusCompleted

	if err := s.updateProcessingStatus(ctx, processingID, status, finalStats); err != nil {
		log.Warn("failed to update final processing status", "error", err)
	}

	log.Info("Releases file processing completed",
		"total", result.TotalRecords,
		"processed", result.ProcessedRecords,
		"inserted", result.InsertedRecords,
		"updated", result.UpdatedRecords,
		"errors", result.ErroredRecords,
	)

	return result, nil
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

func (s *XMLProcessingService) convertDiscogsRelease(
	ctx context.Context,
	discogsRelease *imports.Release,
) *models.Release {
	// Skip releases with invalid data (avoid string ops on invalid data)
	if discogsRelease.ID == 0 || len(discogsRelease.Title) == 0 {
		return nil
	}

	// Single trim operation with length check
	title := strings.TrimSpace(discogsRelease.Title)
	if len(title) == 0 {
		return nil
	}

	release := &models.Release{
		Title:     title,
		DiscogsID: int64(discogsRelease.ID),
		Format:    models.FormatVinyl, // Default format
	}

	// Set optional fields only if they have values
	if len(discogsRelease.Country) > 0 {
		if country := strings.TrimSpace(discogsRelease.Country); len(country) > 0 {
			release.Country = &country
		}
	}

	// Optimized year parsing - avoid string allocation if possible
	if len(discogsRelease.Released) >= 4 {
		yearStr := discogsRelease.Released[:4]
		if len(yearStr) == 4 {
			var year int
			if _, err := fmt.Sscanf(yearStr, "%d", &year); err == nil && year > 1800 &&
				year < 3000 {
				release.Year = &year
			}
		}
	}

	// Handle formats - map to our format enum (optimized string processing)
	if len(discogsRelease.Formats) > 0 {
		firstFormat := discogsRelease.Formats[0]
		if len(firstFormat.Name) > 0 {
			formatName := strings.ToLower(strings.TrimSpace(firstFormat.Name))
			switch {
			case strings.Contains(formatName, "vinyl") || strings.Contains(formatName, "lp") || strings.Contains(formatName, "12\"") || strings.Contains(formatName, "7\""):
				release.Format = models.FormatVinyl
			case strings.Contains(formatName, "cd"):
				release.Format = models.FormatCD
			case strings.Contains(formatName, "cassette") || strings.Contains(formatName, "tape"):
				release.Format = models.FormatCassette
			case strings.Contains(formatName, "digital"):
				release.Format = models.FormatDigital
			default:
				release.Format = models.FormatOther
			}
		}
	}

	// Handle track count (simple length check)
	if trackCount := len(discogsRelease.Tracklist); trackCount > 0 {
		release.TrackCount = &trackCount
	}

	// Convert genres (combining genres and styles like DiscogsParserService)
	for _, genreName := range discogsRelease.Genres {
		if genre := s.findOrCreateGenre(ctx, genreName); genre != nil {
			release.Genres = append(release.Genres, *genre)
		}
	}

	// Convert styles as genres (Discogs treats them as sub-genres)
	for _, styleName := range discogsRelease.Styles {
		if genre := s.findOrCreateGenre(ctx, styleName); genre != nil {
			release.Genres = append(release.Genres, *genre)
		}
	}

	// Convert artists
	for _, discogsArtist := range discogsRelease.Artists {
		if artist := s.findOrCreateArtist(ctx, int64(discogsArtist.ID), strings.TrimSpace(discogsArtist.Name)); artist != nil {
			release.Artists = append(release.Artists, *artist)
		}
	}

	return release
}

// ProcessReleasesFileWithTracks processes releases file including track data
func (s *XMLProcessingService) ProcessReleasesFileWithTracks(
	ctx context.Context,
	filePath string,
	processingID string,
) (*ProcessingResult, error) {
	log := s.log.Function("ProcessReleasesFileWithTracks")
	log.Info(
		"Starting releases file processing with tracks",
		"filePath",
		filePath,
		"processingID",
		processingID,
	)

	// Open and decompress the gzipped XML file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, log.Err("failed to open releases file", err, "filePath", filePath)
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

	var releaseWithTracksBatch []ReleaseWithTracks
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

		// Look for release start elements
		if startElement, ok := token.(xml.StartElement); ok &&
			startElement.Name.Local == "release" {
			var discogsRelease imports.Release
			if err := decoder.DecodeElement(&discogsRelease, &startElement); err != nil {
				errorMsg := fmt.Sprintf("Failed to decode release element: %v", err)
				result.Errors = append(result.Errors, errorMsg)
				result.ErroredRecords++
				log.Warn("Failed to decode release", "error", err)
				continue
			}

			// Convert Discogs release to our release model
			release := s.convertDiscogsRelease(ctx, &discogsRelease)
			if release == nil {
				result.ErroredRecords++
				continue
			}

			// Create ReleaseWithTracks including track data
			releaseWithTracks := ReleaseWithTracks{
				Release: release,
				Tracks:  discogsRelease.Tracklist,
			}

			releaseWithTracksBatch = append(releaseWithTracksBatch, releaseWithTracks)
			recordCount++
			result.TotalRecords++

			// Process batch when it reaches the limit
			if len(releaseWithTracksBatch) >= XML_BATCH_SIZE {
				if err := s.processReleaseBatchWithTracks(ctx, releaseWithTracksBatch, result); err != nil {
					log.Err(
						"failed to process release batch with tracks",
						err,
						"batchSize",
						len(releaseWithTracksBatch),
					)
					return result, err
				}
				releaseWithTracksBatch = releaseWithTracksBatch[:0] // Reset batch
			}

			// Report progress every PROGRESS_REPORT_INTERVAL records
			if recordCount%PROGRESS_REPORT_INTERVAL == 0 {
				stats := &models.ProcessingStats{
					TotalRecords:      result.TotalRecords,
					ReleasesProcessed: result.ProcessedRecords,
					FailedRecords:     result.ErroredRecords,
				}
				if err := s.updateProcessingStats(ctx, processingID, stats); err != nil {
					log.Warn(
						"failed to update processing stats",
						"error",
						err,
						"recordCount",
						recordCount,
					)
				}
				log.Info(
					"Processing progress",
					"processed",
					recordCount,
					"inserted",
					result.InsertedRecords,
					"updated",
					result.UpdatedRecords,
					"errors",
					result.ErroredRecords,
				)
			}
		}
	}

	// Process remaining batch
	if len(releaseWithTracksBatch) > 0 {
		if err := s.processReleaseBatchWithTracks(ctx, releaseWithTracksBatch, result); err != nil {
			log.Err(
				"failed to process final release batch with tracks",
				err,
				"batchSize",
				len(releaseWithTracksBatch),
			)
			return result, err
		}
	}

	// Update final processing status
	finalStats := &models.ProcessingStats{
		TotalRecords:      result.TotalRecords,
		ReleasesProcessed: result.ProcessedRecords,
		FailedRecords:     result.ErroredRecords,
	}
	status := models.ProcessingStatusCompleted
	if err := s.updateProcessingStatus(ctx, processingID, status, finalStats); err != nil {
		log.Warn("failed to update final processing status", "error", err)
	}

	log.Info("Releases file processing with tracks completed",
		"total", result.TotalRecords,
		"processed", result.ProcessedRecords,
		"inserted", result.InsertedRecords,
		"updated", result.UpdatedRecords,
		"errors", result.ErroredRecords,
	)

	return result, nil
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

// preloadEntityCaches loads existing genres and artists into memory caches for faster lookup
func (s *XMLProcessingService) preloadEntityCaches(ctx context.Context) error {
	log := s.log.Function("preloadEntityCaches")

	// Preload all genres
	genres, err := s.genreRepo.GetAll(ctx)
	if err != nil {
		return log.Err("failed to preload genres", err)
	}

	for _, genre := range genres {
		s.genreCache[genre.Name] = genre
	}

	log.Info("Preloaded entity caches", "genres", len(genres))
	return nil
}

