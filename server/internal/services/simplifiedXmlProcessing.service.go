package services

import (
	"context"
	"runtime"
	"time"
	"waugzee/internal/logger"
	"waugzee/internal/models"
	"waugzee/internal/repositories"
)

// SimplifiedResult holds parsed entities in memory maps for simplified processing
type SimplifiedResult struct {
	Labels   map[int64]*models.Label   // DiscogsID -> Label
	Artists  map[int64]*models.Artist  // DiscogsID -> Artist
	Masters  map[int64]*models.Master  // DiscogsID -> Master
	Releases map[int64]*models.Release // DiscogsID -> Release

	// Basic stats
	TotalRecords   int
	ParsedRecords  int
	ErroredRecords int
	Errors         []string
}

type SimplifiedXMLProcessingService struct {
	discogsDataProcessingRepo repositories.DiscogsDataProcessingRepository
	parserService             *DiscogsParserService
	log                       logger.Logger
}

func NewSimplifiedXMLProcessingService(
	discogsDataProcessingRepo repositories.DiscogsDataProcessingRepository,
	parserService *DiscogsParserService,
) *SimplifiedXMLProcessingService {
	return &SimplifiedXMLProcessingService{
		discogsDataProcessingRepo: discogsDataProcessingRepo,
		parserService:             parserService,
		log:                       logger.New("simplifiedXMLProcessingService"),
	}
}

// ProcessFileToMap parses the entire file into memory maps without database operations
// This is a simplified approach that eliminates batching and complex state management
func (s *SimplifiedXMLProcessingService) ProcessFileToMap(
	ctx context.Context,
	filePath string,
	fileType string,
) (*SimplifiedResult, error) {
	log := s.log.Function("ProcessFileToMap")

	// Log initial memory allocation
	var startMemStats runtime.MemStats
	runtime.ReadMemStats(&startMemStats)
	startTime := time.Now()

	log.Info("Starting simplified file processing",
		"filePath", filePath,
		"fileType", fileType,
		"startMemoryMB", startMemStats.Alloc/1024/1024,
		"startHeapMB", startMemStats.HeapAlloc/1024/1024)

	// Initialize result with maps
	result := &SimplifiedResult{
		Labels:   make(map[int64]*models.Label),
		Artists:  make(map[int64]*models.Artist),
		Masters:  make(map[int64]*models.Master),
		Releases: make(map[int64]*models.Release),
		Errors:   make([]string, 0),
	}

	// Use the parser service to parse the entire file
	parseOptions := ParseOptions{
		FilePath: filePath,
		FileType: fileType,
	}

	parseResult, err := s.parserService.ParseFile(ctx, parseOptions)
	if err != nil {
		return nil, log.Err("parsing failed", err, "filePath", filePath)
	}

	// Convert parsed results to maps based on file type
	switch fileType {
	case "labels":
		for _, label := range parseResult.ParsedLabels {
			if label != nil && label.DiscogsID != nil {
				result.Labels[*label.DiscogsID] = label
				result.ParsedRecords++
			}
		}
	case "artists":
		for _, artist := range parseResult.ParsedArtists {
			if artist != nil && artist.DiscogsID != nil {
				result.Artists[*artist.DiscogsID] = artist
				result.ParsedRecords++
			}
		}
	case "masters":
		for _, master := range parseResult.ParsedMasters {
			if master != nil && master.DiscogsID != nil {
				result.Masters[*master.DiscogsID] = master
				result.ParsedRecords++
			}
		}
	case "releases":
		for _, release := range parseResult.ParsedReleases {
			if release != nil {
				result.Releases[release.DiscogsID] = release
				result.ParsedRecords++
			}
		}
	default:
		return nil, log.Err("unsupported file type", nil, "fileType", fileType)
	}

	// Copy basic stats from parse result
	result.TotalRecords = parseResult.TotalRecords
	result.ErroredRecords = parseResult.ErroredRecords
	result.Errors = parseResult.Errors

	// Log completion with final memory allocation
	var endMemStats runtime.MemStats
	runtime.ReadMemStats(&endMemStats)
	elapsed := time.Since(startTime)

	log.Info("Completed simplified file processing",
		"fileType", fileType,
		"totalRecords", result.TotalRecords,
		"parsedRecords", result.ParsedRecords,
		"erroredRecords", result.ErroredRecords,
		"mapSize", len(result.Labels)+len(result.Artists)+len(result.Masters)+len(result.Releases),
		"elapsedMs", elapsed.Milliseconds(),
		"finalMemoryMB", endMemStats.Alloc/1024/1024,
		"finalHeapMB", endMemStats.HeapAlloc/1024/1024,
		"memoryDeltaMB", int64(endMemStats.Alloc-startMemStats.Alloc)/1024/1024)

	return result, nil
}

func (s *SimplifiedXMLProcessingService) ProcessLabelsFile(
	ctx context.Context,
	filePath string,
	processingID string,
) (*ProcessingResult, error) {
	log := s.log.Function("ProcessLabelsFile")

	log.Info("Starting labels file processing with simplified approach",
		"filePath", filePath,
		"processingID", processingID)

	// Update processing status to "processing"
	if err := s.updateProcessingStatus(ctx, processingID, models.ProcessingStatusProcessing, nil); err != nil {
		log.Warn("failed to update processing status", "error", err, "processingID", processingID)
	}

	// Use simplified parsing approach
	simplifiedResult, err := s.ProcessFileToMap(ctx, filePath, "labels")
	if err != nil {
		// Update status to failed
		if statusErr := s.updateProcessingStatus(ctx, processingID, models.ProcessingStatusFailed, nil); statusErr != nil {
			log.Warn(
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
	result := &ProcessingResult{
		TotalRecords:     simplifiedResult.TotalRecords,
		ProcessedRecords: simplifiedResult.ParsedRecords,
		InsertedRecords:  0, // No database operations performed
		UpdatedRecords:   0, // No database operations performed
		ErroredRecords:   simplifiedResult.ErroredRecords,
		Errors:           simplifiedResult.Errors,
	}

	// Create processing stats for final status update
	finalStats := s.createProcessingStats(
		"labels",
		result.TotalRecords,
		result.ProcessedRecords,
		result.ErroredRecords,
	)

	// Update final processing status to completed
	if err := s.updateProcessingStatus(ctx, processingID, models.ProcessingStatusCompleted, finalStats); err != nil {
		log.Warn("failed to update final processing status", "error", err)
	}

	log.Info("Labels file processing completed with simplified approach (early return)",
		"fileType", "labels",
		"total", result.TotalRecords,
		"processed", result.ProcessedRecords,
		"errors", result.ErroredRecords,
		"labelsInMap", len(simplifiedResult.Labels))

	// Early return - no database operations performed
	return result, nil
}

func (s *SimplifiedXMLProcessingService) ProcessArtistsFile(
	ctx context.Context,
	filePath string,
	processingID string,
) (*ProcessingResult, error) {
	log := s.log.Function("ProcessArtistsFile")

	log.Info("Starting artists file processing with simplified approach",
		"filePath", filePath,
		"processingID", processingID)

	// Update processing status to "processing"
	if err := s.updateProcessingStatus(ctx, processingID, models.ProcessingStatusProcessing, nil); err != nil {
		log.Warn("failed to update processing status", "error", err, "processingID", processingID)
	}

	// Use simplified parsing approach
	simplifiedResult, err := s.ProcessFileToMap(ctx, filePath, "artists")
	if err != nil {
		// Update status to failed
		if statusErr := s.updateProcessingStatus(ctx, processingID, models.ProcessingStatusFailed, nil); statusErr != nil {
			log.Warn(
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
	result := &ProcessingResult{
		TotalRecords:     simplifiedResult.TotalRecords,
		ProcessedRecords: simplifiedResult.ParsedRecords,
		InsertedRecords:  0, // No database operations performed
		UpdatedRecords:   0, // No database operations performed
		ErroredRecords:   simplifiedResult.ErroredRecords,
		Errors:           simplifiedResult.Errors,
	}

	// Create processing stats for final status update
	finalStats := s.createProcessingStats(
		"artists",
		result.TotalRecords,
		result.ProcessedRecords,
		result.ErroredRecords,
	)

	// Update final processing status to completed
	if err := s.updateProcessingStatus(ctx, processingID, models.ProcessingStatusCompleted, finalStats); err != nil {
		log.Warn("failed to update final processing status", "error", err)
	}

	log.Info("Artists file processing completed with simplified approach (early return)",
		"fileType", "artists",
		"total", result.TotalRecords,
		"processed", result.ProcessedRecords,
		"errors", result.ErroredRecords,
		"artistsInMap", len(simplifiedResult.Artists))

	// Early return - no database operations performed
	return result, nil
}

func (s *SimplifiedXMLProcessingService) ProcessMastersFile(
	ctx context.Context,
	filePath string,
	processingID string,
) (*ProcessingResult, error) {
	log := s.log.Function("ProcessMastersFile")

	log.Info("Starting masters file processing with simplified approach",
		"filePath", filePath,
		"processingID", processingID)

	// Update processing status to "processing"
	if err := s.updateProcessingStatus(ctx, processingID, models.ProcessingStatusProcessing, nil); err != nil {
		log.Warn("failed to update processing status", "error", err, "processingID", processingID)
	}

	// Use simplified parsing approach
	simplifiedResult, err := s.ProcessFileToMap(ctx, filePath, "masters")
	if err != nil {
		// Update status to failed
		if statusErr := s.updateProcessingStatus(ctx, processingID, models.ProcessingStatusFailed, nil); statusErr != nil {
			log.Warn(
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
	result := &ProcessingResult{
		TotalRecords:     simplifiedResult.TotalRecords,
		ProcessedRecords: simplifiedResult.ParsedRecords,
		InsertedRecords:  0, // No database operations performed
		UpdatedRecords:   0, // No database operations performed
		ErroredRecords:   simplifiedResult.ErroredRecords,
		Errors:           simplifiedResult.Errors,
	}

	// Create processing stats for final status update
	finalStats := s.createProcessingStats(
		"masters",
		result.TotalRecords,
		result.ProcessedRecords,
		result.ErroredRecords,
	)

	// Update final processing status to completed
	if err := s.updateProcessingStatus(ctx, processingID, models.ProcessingStatusCompleted, finalStats); err != nil {
		log.Warn("failed to update final processing status", "error", err)
	}

	log.Info("Masters file processing completed with simplified approach (early return)",
		"fileType", "masters",
		"total", result.TotalRecords,
		"processed", result.ProcessedRecords,
		"errors", result.ErroredRecords,
		"mastersInMap", len(simplifiedResult.Masters))

	// Early return - no database operations performed
	return result, nil
}

func (s *SimplifiedXMLProcessingService) ProcessReleasesFile(
	ctx context.Context,
	filePath string,
	processingID string,
) (*ProcessingResult, error) {
	log := s.log.Function("ProcessReleasesFile")

	log.Info("Starting releases file processing with simplified approach",
		"filePath", filePath,
		"processingID", processingID)

	// Update processing status to "processing"
	if err := s.updateProcessingStatus(ctx, processingID, models.ProcessingStatusProcessing, nil); err != nil {
		log.Warn("failed to update processing status", "error", err, "processingID", processingID)
	}

	// Use simplified parsing approach
	simplifiedResult, err := s.ProcessFileToMap(ctx, filePath, "releases")
	if err != nil {
		// Update status to failed
		if statusErr := s.updateProcessingStatus(ctx, processingID, models.ProcessingStatusFailed, nil); statusErr != nil {
			log.Warn(
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
	result := &ProcessingResult{
		TotalRecords:     simplifiedResult.TotalRecords,
		ProcessedRecords: simplifiedResult.ParsedRecords,
		InsertedRecords:  0, // No database operations performed
		UpdatedRecords:   0, // No database operations performed
		ErroredRecords:   simplifiedResult.ErroredRecords,
		Errors:           simplifiedResult.Errors,
	}

	// Create processing stats for final status update
	finalStats := s.createProcessingStats(
		"releases",
		result.TotalRecords,
		result.ProcessedRecords,
		result.ErroredRecords,
	)

	// Update final processing status to completed
	if err := s.updateProcessingStatus(ctx, processingID, models.ProcessingStatusCompleted, finalStats); err != nil {
		log.Warn("failed to update final processing status", "error", err)
	}

	log.Info("Releases file processing completed with simplified approach (early return)",
		"fileType", "releases",
		"total", result.TotalRecords,
		"processed", result.ProcessedRecords,
		"errors", result.ErroredRecords,
		"releasesInMap", len(simplifiedResult.Releases))

	// Early return - no database operations performed
	return result, nil
}

// Helper method to create appropriate ProcessingStats based on entity type
func (s *SimplifiedXMLProcessingService) createProcessingStats(
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