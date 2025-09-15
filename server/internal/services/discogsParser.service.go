package services

import (
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"time"
	"waugzee/internal/imports"
	"waugzee/internal/logger"
	"waugzee/internal/models"
)

// ParseResult represents the result of parsing a Discogs XML file
type ParseResult struct {
	TotalRecords    int
	ProcessedRecords int
	ErroredRecords  int
	Errors          []string
	ParsedLabels    []*models.Label
	ParsedArtists   []*models.Artist
	ParsedMasters   []*models.Master
	ParsedReleases  []*models.Release
}

// ParseOptions configures parsing behavior
type ParseOptions struct {
	FilePath      string
	FileType      string // "labels", "artists", "masters", "releases"
	BatchSize     int    // Optional: batch size for processing (default: 2000)
	MaxRecords    int    // Optional: limit total records parsed (0 = no limit)
	ProgressFunc  func(processed, total, errors int) // Optional: progress callback
}

// DiscogsParserService provides pure XML parsing functionality without database operations
type DiscogsParserService struct {
	log logger.Logger
}

// NewDiscogsParserService creates a new parser service
func NewDiscogsParserService() *DiscogsParserService {
	return &DiscogsParserService{
		log: logger.New("discogsParserService"),
	}
}

// ParseFile parses a Discogs XML file and returns parsed models without database operations
func (s *DiscogsParserService) ParseFile(ctx context.Context, options ParseOptions) (*ParseResult, error) {
	log := s.log.Function("ParseFile")

	if options.FilePath == "" {
		return nil, log.Err("file path is required", nil)
	}

	if options.FileType == "" {
		return nil, log.Err("file type is required", nil)
	}

	// Set default batch size
	batchSize := options.BatchSize
	if batchSize <= 0 {
		batchSize = XML_BATCH_SIZE
	}

	log.Info("Starting XML file parsing",
		"filePath", options.FilePath,
		"fileType", options.FileType,
		"batchSize", batchSize,
		"maxRecords", options.MaxRecords)

	// Validate file exists and get info
	fileInfo, err := os.Stat(options.FilePath)
	if err != nil {
		return nil, log.Err("failed to stat file", err, "filePath", options.FilePath)
	}

	log.Info("File validation",
		"fileSize", fileInfo.Size(),
		"modTime", fileInfo.ModTime(),
		"isDir", fileInfo.IsDir())

	// Open and decompress the gzipped XML file
	file, err := os.Open(options.FilePath)
	if err != nil {
		return nil, log.Err("failed to open file", err, "filePath", options.FilePath)
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, log.Err("failed to create gzip reader - file may not be gzipped", err, "filePath", options.FilePath)
	}
	defer gzipReader.Close()

	// Read first few bytes to validate XML structure and log sample content
	if err := s.validateAndLogXMLStructure(gzipReader, options.FileType, log); err != nil {
		return nil, log.Err("XML structure validation failed", err)
	}

	// Reopen file for actual parsing (since we consumed some content for validation)
	file.Close()
	file, err = os.Open(options.FilePath)
	if err != nil {
		return nil, log.Err("failed to reopen file for parsing", err, "filePath", options.FilePath)
	}
	defer file.Close()

	gzipReader, err = gzip.NewReader(file)
	if err != nil {
		return nil, log.Err("failed to recreate gzip reader for parsing", err, "filePath", options.FilePath)
	}
	defer gzipReader.Close()

	// Route to appropriate parser based on file type
	switch options.FileType {
	case "labels":
		return s.parseLabelsFile(ctx, gzipReader, options, log)
	case "artists":
		return s.parseArtistsFile(ctx, gzipReader, options, log)
	case "masters":
		return s.parseMastersFile(ctx, gzipReader, options, log)
	case "releases":
		return s.parseReleasesFile(ctx, gzipReader, options, log)
	default:
		return nil, log.Err("unsupported file type", nil, "fileType", options.FileType)
	}
}

// parseLabelsFile handles parsing of labels XML files
func (s *DiscogsParserService) parseLabelsFile(ctx context.Context, reader io.Reader, options ParseOptions, log logger.Logger) (*ParseResult, error) {
	decoder := xml.NewDecoder(reader)

	result := &ParseResult{
		Errors:       make([]string, 0),
		ParsedLabels: make([]*models.Label, 0),
	}

	var batch []*models.Label
	batchSize := options.BatchSize
	if batchSize <= 0 {
		batchSize = XML_BATCH_SIZE
	}

	// Performance tracking
	startTime := time.Now()
	lastProgressTime := startTime
	progressInterval := 1000 // Log every 1000 records
	successfulSamples := 0
	maxSamplesToLog := 3

	log.Info("Starting labels parsing",
		"batchSize", batchSize,
		"progressInterval", progressInterval)

	// Stream through the XML file
	for {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

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
			// Increment total record count for each record we attempt to process
			result.TotalRecords++

			// Check max records limit before processing - count all attempted records
			if options.MaxRecords > 0 && result.TotalRecords > options.MaxRecords {
				log.Info("Reached max records limit - stopping early",
					"maxRecords", options.MaxRecords,
					"totalAttempted", result.TotalRecords-1) // -1 because we haven't processed this one
				result.TotalRecords-- // Adjust count since we're not processing this record
				break
			}

			var discogsLabel imports.Label
			if err := decoder.DecodeElement(&discogsLabel, &startElement); err != nil {
				errorMsg := fmt.Sprintf("Failed to decode label element at record %d: %v", result.TotalRecords, err)
				log.Error("XML decode error",
					"error", err,
					"recordNumber", result.TotalRecords,
					"elementName", startElement.Name.Local,
					"elementAttrs", s.formatAttributes(startElement.Attr))
				result.Errors = append(result.Errors, errorMsg)
				result.ErroredRecords++
				continue
			}

			// Convert Discogs label to our label model
			label := s.convertDiscogsLabel(&discogsLabel)
			if label == nil {
				log.Warn("Failed to convert Discogs label",
					"discogsID", discogsLabel.ID,
					"name", discogsLabel.Name,
					"recordNumber", result.TotalRecords)
				result.ErroredRecords++
				continue
			}

			// Log successful parse samples
			if successfulSamples < maxSamplesToLog {
				s.logSuccessfulRecord("label", &discogsLabel, label, result.TotalRecords, log)
				successfulSamples++
			}

			batch = append(batch, label)

			// Process batch when it reaches the limit
			if len(batch) >= batchSize {
				result.ParsedLabels = append(result.ParsedLabels, batch...)
				result.ProcessedRecords += len(batch)

				// Call progress callback if provided
				if options.ProgressFunc != nil {
					options.ProgressFunc(result.ProcessedRecords, result.TotalRecords, result.ErroredRecords)
				}

				batch = batch[:0] // Reset batch
			}

			// Log progress every N records (based on total attempted, not just successful)
			if result.TotalRecords%progressInterval == 0 {
				currentTime := time.Now()
				elapsed := currentTime.Sub(lastProgressTime)
				recordsPerSecond := float64(progressInterval) / elapsed.Seconds()
				totalElapsed := currentTime.Sub(startTime)

				var memStats runtime.MemStats
				runtime.ReadMemStats(&memStats)

				successfulRecords := result.TotalRecords - result.ErroredRecords
				log.Info("Parsing progress",
					"totalRecords", result.TotalRecords,
					"processedRecords", result.ProcessedRecords,
					"erroredRecords", result.ErroredRecords,
					"successfulRecords", successfulRecords,
					"recordsPerSecond", fmt.Sprintf("%.1f", recordsPerSecond),
					"totalElapsedMs", totalElapsed.Milliseconds(),
					"memoryUsageMB", memStats.Alloc/1024/1024,
					"successRate", fmt.Sprintf("%.2f%%", float64(successfulRecords)/float64(result.TotalRecords)*100))

				lastProgressTime = currentTime
			}
		}
	}

	// Process remaining batch
	if len(batch) > 0 {
		result.ParsedLabels = append(result.ParsedLabels, batch...)
		result.ProcessedRecords += len(batch)
	}

	totalElapsed := time.Since(startTime)
	overallRecordsPerSecond := float64(result.TotalRecords) / totalElapsed.Seconds()
	successfulRecords := result.TotalRecords - result.ErroredRecords

	log.Info("Labels file parsing completed",
		"total", result.TotalRecords,
		"processed", result.ProcessedRecords,
		"errors", result.ErroredRecords,
		"successful", successfulRecords,
		"totalElapsedMs", totalElapsed.Milliseconds(),
		"overallRecordsPerSecond", fmt.Sprintf("%.1f", overallRecordsPerSecond),
		"successRate", fmt.Sprintf("%.2f%%", float64(successfulRecords)/float64(result.TotalRecords)*100),
		"errorSampleCount", len(result.Errors))

	// Log first few errors for debugging
	if len(result.Errors) > 0 {
		maxErrorsToLog := 5
		if len(result.Errors) < maxErrorsToLog {
			maxErrorsToLog = len(result.Errors)
		}
		log.Error("Sample parsing errors",
			"sampleErrors", result.Errors[:maxErrorsToLog],
			"totalErrors", len(result.Errors))
	}

	return result, nil
}

// parseArtistsFile handles parsing of artists XML files
func (s *DiscogsParserService) parseArtistsFile(ctx context.Context, reader io.Reader, options ParseOptions, log logger.Logger) (*ParseResult, error) {
	decoder := xml.NewDecoder(reader)

	result := &ParseResult{
		Errors:        make([]string, 0),
		ParsedArtists: make([]*models.Artist, 0),
	}

	var batch []*models.Artist
	batchSize := options.BatchSize
	if batchSize <= 0 {
		batchSize = XML_BATCH_SIZE
	}

	// Performance tracking
	startTime := time.Now()
	lastProgressTime := startTime
	progressInterval := 1000 // Log every 1000 records
	successfulSamples := 0
	maxSamplesToLog := 3

	log.Info("Starting artists parsing",
		"batchSize", batchSize,
		"progressInterval", progressInterval)

	// Stream through the XML file
	for {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

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
			// Increment total record count for each record we attempt to process
			result.TotalRecords++

			// Check max records limit before processing - count all attempted records
			if options.MaxRecords > 0 && result.TotalRecords > options.MaxRecords {
				log.Info("Reached max records limit - stopping early",
					"maxRecords", options.MaxRecords,
					"totalAttempted", result.TotalRecords-1) // -1 because we haven't processed this one
				result.TotalRecords-- // Adjust count since we're not processing this record
				break
			}

			var discogsArtist imports.Artist
			if err := decoder.DecodeElement(&discogsArtist, &startElement); err != nil {
				errorMsg := fmt.Sprintf("Failed to decode artist element at record %d: %v", result.TotalRecords, err)
				log.Error("XML decode error",
					"error", err,
					"recordNumber", result.TotalRecords,
					"elementName", startElement.Name.Local,
					"elementAttrs", s.formatAttributes(startElement.Attr))
				result.Errors = append(result.Errors, errorMsg)
				result.ErroredRecords++
				continue
			}

			// Convert Discogs artist to our artist model
			artist := s.convertDiscogsArtist(&discogsArtist)
			if artist == nil {
				log.Warn("Failed to convert Discogs artist",
					"discogsID", discogsArtist.ID,
					"name", discogsArtist.Name,
					"recordNumber", result.TotalRecords)
				result.ErroredRecords++
				continue
			}

			// Log successful parse samples
			if successfulSamples < maxSamplesToLog {
				s.logSuccessfulRecord("artist", &discogsArtist, artist, result.TotalRecords, log)
				successfulSamples++
			}

			batch = append(batch, artist)

			// Process batch when it reaches the limit
			if len(batch) >= batchSize {
				result.ParsedArtists = append(result.ParsedArtists, batch...)
				result.ProcessedRecords += len(batch)

				// Call progress callback if provided
				if options.ProgressFunc != nil {
					options.ProgressFunc(result.ProcessedRecords, result.TotalRecords, result.ErroredRecords)
				}

				batch = batch[:0] // Reset batch
			}

			// Log progress every N records (based on total attempted, not just successful)
			if result.TotalRecords%progressInterval == 0 {
				currentTime := time.Now()
				elapsed := currentTime.Sub(lastProgressTime)
				recordsPerSecond := float64(progressInterval) / elapsed.Seconds()
				totalElapsed := currentTime.Sub(startTime)

				var memStats runtime.MemStats
				runtime.ReadMemStats(&memStats)

				successfulRecords := result.TotalRecords - result.ErroredRecords
				log.Info("Parsing progress",
					"totalRecords", result.TotalRecords,
					"processedRecords", result.ProcessedRecords,
					"erroredRecords", result.ErroredRecords,
					"successfulRecords", successfulRecords,
					"recordsPerSecond", fmt.Sprintf("%.1f", recordsPerSecond),
					"totalElapsedMs", totalElapsed.Milliseconds(),
					"memoryUsageMB", memStats.Alloc/1024/1024,
					"successRate", fmt.Sprintf("%.2f%%", float64(successfulRecords)/float64(result.TotalRecords)*100))

				lastProgressTime = currentTime
			}
		}
	}

	// Process remaining batch
	if len(batch) > 0 {
		result.ParsedArtists = append(result.ParsedArtists, batch...)
		result.ProcessedRecords += len(batch)
	}

	totalElapsed := time.Since(startTime)
	overallRecordsPerSecond := float64(result.TotalRecords) / totalElapsed.Seconds()
	successfulRecords := result.TotalRecords - result.ErroredRecords

	log.Info("Artists file parsing completed",
		"total", result.TotalRecords,
		"processed", result.ProcessedRecords,
		"errors", result.ErroredRecords,
		"successful", successfulRecords,
		"totalElapsedMs", totalElapsed.Milliseconds(),
		"overallRecordsPerSecond", fmt.Sprintf("%.1f", overallRecordsPerSecond),
		"successRate", fmt.Sprintf("%.2f%%", float64(successfulRecords)/float64(result.TotalRecords)*100),
		"errorSampleCount", len(result.Errors))

	// Log first few errors for debugging
	if len(result.Errors) > 0 {
		maxErrorsToLog := 5
		if len(result.Errors) < maxErrorsToLog {
			maxErrorsToLog = len(result.Errors)
		}
		log.Error("Sample parsing errors",
			"sampleErrors", result.Errors[:maxErrorsToLog],
			"totalErrors", len(result.Errors))
	}

	return result, nil
}

// parseMastersFile handles parsing of masters XML files
func (s *DiscogsParserService) parseMastersFile(ctx context.Context, reader io.Reader, options ParseOptions, log logger.Logger) (*ParseResult, error) {
	decoder := xml.NewDecoder(reader)

	result := &ParseResult{
		Errors:        make([]string, 0),
		ParsedMasters: make([]*models.Master, 0),
	}

	var batch []*models.Master
	batchSize := options.BatchSize
	if batchSize <= 0 {
		batchSize = XML_BATCH_SIZE
	}

	// Performance tracking
	startTime := time.Now()
	lastProgressTime := startTime
	progressInterval := 1000
	successfulSamples := 0
	maxSamplesToLog := 3

	log.Info("Starting masters parsing",
		"batchSize", batchSize,
		"progressInterval", progressInterval)

	// Stream through the XML file
	for {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

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
			// Increment total record count for each record we attempt to process
			result.TotalRecords++

			// Check max records limit before processing - count all attempted records
			if options.MaxRecords > 0 && result.TotalRecords > options.MaxRecords {
				log.Info("Reached max records limit - stopping early",
					"maxRecords", options.MaxRecords,
					"totalAttempted", result.TotalRecords-1) // -1 because we haven't processed this one
				result.TotalRecords-- // Adjust count since we're not processing this record
				break
			}

			var discogsMaster imports.Master
			if err := decoder.DecodeElement(&discogsMaster, &startElement); err != nil {
				errorMsg := fmt.Sprintf("Failed to decode master element at record %d: %v", result.TotalRecords, err)
				log.Error("XML decode error",
					"error", err,
					"recordNumber", result.TotalRecords,
					"elementName", startElement.Name.Local,
					"elementAttrs", s.formatAttributes(startElement.Attr))
				result.Errors = append(result.Errors, errorMsg)
				result.ErroredRecords++
				continue
			}

			// Convert Discogs master to our master model
			master := s.convertDiscogsMaster(&discogsMaster)
			if master == nil {
				log.Warn("Failed to convert Discogs master",
					"discogsID", discogsMaster.ID,
					"title", discogsMaster.Title,
					"recordNumber", result.TotalRecords)
				result.ErroredRecords++
				continue
			}

			// Log successful parse samples
			if successfulSamples < maxSamplesToLog {
				s.logSuccessfulRecord("master", &discogsMaster, master, result.TotalRecords, log)
				successfulSamples++
			}

			batch = append(batch, master)

			// Process batch when it reaches the limit
			if len(batch) >= batchSize {
				result.ParsedMasters = append(result.ParsedMasters, batch...)
				result.ProcessedRecords += len(batch)

				// Call progress callback if provided
				if options.ProgressFunc != nil {
					options.ProgressFunc(result.ProcessedRecords, result.TotalRecords, result.ErroredRecords)
				}

				batch = batch[:0] // Reset batch
			}

			// Log progress every N records (based on total attempted, not just successful)
			if result.TotalRecords%progressInterval == 0 {
				currentTime := time.Now()
				elapsed := currentTime.Sub(lastProgressTime)
				recordsPerSecond := float64(progressInterval) / elapsed.Seconds()
				totalElapsed := currentTime.Sub(startTime)

				var memStats runtime.MemStats
				runtime.ReadMemStats(&memStats)

				successfulRecords := result.TotalRecords - result.ErroredRecords
				log.Info("Parsing progress",
					"totalRecords", result.TotalRecords,
					"processedRecords", result.ProcessedRecords,
					"erroredRecords", result.ErroredRecords,
					"successfulRecords", successfulRecords,
					"recordsPerSecond", fmt.Sprintf("%.1f", recordsPerSecond),
					"totalElapsedMs", totalElapsed.Milliseconds(),
					"memoryUsageMB", memStats.Alloc/1024/1024,
					"successRate", fmt.Sprintf("%.2f%%", float64(successfulRecords)/float64(result.TotalRecords)*100))

				lastProgressTime = currentTime
			}
		}
	}

	// Process remaining batch
	if len(batch) > 0 {
		result.ParsedMasters = append(result.ParsedMasters, batch...)
		result.ProcessedRecords += len(batch)
	}

	totalElapsed := time.Since(startTime)
	overallRecordsPerSecond := float64(result.TotalRecords) / totalElapsed.Seconds()
	successfulRecords := result.TotalRecords - result.ErroredRecords

	log.Info("Masters file parsing completed",
		"total", result.TotalRecords,
		"processed", result.ProcessedRecords,
		"errors", result.ErroredRecords,
		"successful", successfulRecords,
		"totalElapsedMs", totalElapsed.Milliseconds(),
		"overallRecordsPerSecond", fmt.Sprintf("%.1f", overallRecordsPerSecond),
		"successRate", fmt.Sprintf("%.2f%%", float64(successfulRecords)/float64(result.TotalRecords)*100),
		"errorSampleCount", len(result.Errors))

	// Log first few errors for debugging
	if len(result.Errors) > 0 {
		maxErrorsToLog := 5
		if len(result.Errors) < maxErrorsToLog {
			maxErrorsToLog = len(result.Errors)
		}
		log.Error("Sample parsing errors",
			"sampleErrors", result.Errors[:maxErrorsToLog],
			"totalErrors", len(result.Errors))
	}

	return result, nil
}

// parseReleasesFile handles parsing of releases XML files
func (s *DiscogsParserService) parseReleasesFile(ctx context.Context, reader io.Reader, options ParseOptions, log logger.Logger) (*ParseResult, error) {
	decoder := xml.NewDecoder(reader)

	result := &ParseResult{
		Errors:         make([]string, 0),
		ParsedReleases: make([]*models.Release, 0),
	}

	var batch []*models.Release
	batchSize := options.BatchSize
	if batchSize <= 0 {
		batchSize = XML_BATCH_SIZE
	}

	// Performance tracking
	startTime := time.Now()
	lastProgressTime := startTime
	progressInterval := 1000
	successfulSamples := 0
	maxSamplesToLog := 3

	log.Info("Starting releases parsing",
		"batchSize", batchSize,
		"progressInterval", progressInterval)

	// Stream through the XML file
	for {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

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
		if startElement, ok := token.(xml.StartElement); ok && startElement.Name.Local == "release" {
			// Increment total record count for each record we attempt to process
			result.TotalRecords++

			// Check max records limit before processing - count all attempted records
			if options.MaxRecords > 0 && result.TotalRecords > options.MaxRecords {
				log.Info("Reached max records limit - stopping early",
					"maxRecords", options.MaxRecords,
					"totalAttempted", result.TotalRecords-1) // -1 because we haven't processed this one
				result.TotalRecords-- // Adjust count since we're not processing this record
				break
			}

			var discogsRelease imports.Release
			if err := decoder.DecodeElement(&discogsRelease, &startElement); err != nil {
				errorMsg := fmt.Sprintf("Failed to decode release element at record %d: %v", result.TotalRecords, err)
				log.Error("XML decode error",
					"error", err,
					"recordNumber", result.TotalRecords,
					"elementName", startElement.Name.Local,
					"elementAttrs", s.formatAttributes(startElement.Attr))
				result.Errors = append(result.Errors, errorMsg)
				result.ErroredRecords++
				continue
			}

			// Convert Discogs release to our release model
			release := s.convertDiscogsRelease(&discogsRelease)
			if release == nil {
				log.Warn("Failed to convert Discogs release",
					"discogsID", discogsRelease.ID,
					"title", discogsRelease.Title,
					"recordNumber", result.TotalRecords)
				result.ErroredRecords++
				continue
			}

			// Log successful parse samples
			if successfulSamples < maxSamplesToLog {
				s.logSuccessfulRecord("release", &discogsRelease, release, result.TotalRecords, log)
				successfulSamples++
			}

			batch = append(batch, release)

			// Process batch when it reaches the limit
			if len(batch) >= batchSize {
				result.ParsedReleases = append(result.ParsedReleases, batch...)
				result.ProcessedRecords += len(batch)

				// Call progress callback if provided
				if options.ProgressFunc != nil {
					options.ProgressFunc(result.ProcessedRecords, result.TotalRecords, result.ErroredRecords)
				}

				batch = batch[:0] // Reset batch
			}

			// Log progress every N records (based on total attempted, not just successful)
			if result.TotalRecords%progressInterval == 0 {
				currentTime := time.Now()
				elapsed := currentTime.Sub(lastProgressTime)
				recordsPerSecond := float64(progressInterval) / elapsed.Seconds()
				totalElapsed := currentTime.Sub(startTime)

				var memStats runtime.MemStats
				runtime.ReadMemStats(&memStats)

				successfulRecords := result.TotalRecords - result.ErroredRecords
				log.Info("Parsing progress",
					"totalRecords", result.TotalRecords,
					"processedRecords", result.ProcessedRecords,
					"erroredRecords", result.ErroredRecords,
					"successfulRecords", successfulRecords,
					"recordsPerSecond", fmt.Sprintf("%.1f", recordsPerSecond),
					"totalElapsedMs", totalElapsed.Milliseconds(),
					"memoryUsageMB", memStats.Alloc/1024/1024,
					"successRate", fmt.Sprintf("%.2f%%", float64(successfulRecords)/float64(result.TotalRecords)*100))

				lastProgressTime = currentTime
			}
		}
	}

	// Process remaining batch
	if len(batch) > 0 {
		result.ParsedReleases = append(result.ParsedReleases, batch...)
		result.ProcessedRecords += len(batch)
	}

	totalElapsed := time.Since(startTime)
	overallRecordsPerSecond := float64(result.TotalRecords) / totalElapsed.Seconds()
	successfulRecords := result.TotalRecords - result.ErroredRecords

	log.Info("Releases file parsing completed",
		"total", result.TotalRecords,
		"processed", result.ProcessedRecords,
		"errors", result.ErroredRecords,
		"successful", successfulRecords,
		"totalElapsedMs", totalElapsed.Milliseconds(),
		"overallRecordsPerSecond", fmt.Sprintf("%.1f", overallRecordsPerSecond),
		"successRate", fmt.Sprintf("%.2f%%", float64(successfulRecords)/float64(result.TotalRecords)*100),
		"errorSampleCount", len(result.Errors))

	// Log first few errors for debugging
	if len(result.Errors) > 0 {
		maxErrorsToLog := 5
		if len(result.Errors) < maxErrorsToLog {
			maxErrorsToLog = len(result.Errors)
		}
		log.Error("Sample parsing errors",
			"sampleErrors", result.Errors[:maxErrorsToLog],
			"totalErrors", len(result.Errors))
	}

	return result, nil
}

// Conversion methods - extracted from XMLProcessingService for reuse

func (s *DiscogsParserService) convertDiscogsLabel(discogsLabel *imports.Label) *models.Label {
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

	return label
}

func (s *DiscogsParserService) convertDiscogsArtist(discogsArtist *imports.Artist) *models.Artist {
	// Skip artists with invalid data (avoid string ops on invalid data)
	if discogsArtist.ID == 0 || len(discogsArtist.Name) == 0 {
		return nil
	}

	// Single trim operation with length check
	name := strings.TrimSpace(discogsArtist.Name)
	if len(name) == 0 {
		return nil
	}

	artist := &models.Artist{
		Name:     name,
		IsActive: true, // Default to active
	}

	// Set Discogs ID
	discogsID := int64(discogsArtist.ID)
	artist.DiscogsID = &discogsID

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

	return artist
}

func (s *DiscogsParserService) convertDiscogsMaster(discogsMaster *imports.Master) *models.Master {
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

	return master
}

func (s *DiscogsParserService) convertDiscogsRelease(discogsRelease *imports.Release) *models.Release {
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
			if _, err := fmt.Sscanf(yearStr, "%d", &year); err == nil && year > 1800 && year < 3000 {
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

	return release
}

// validateAndLogXMLStructure reads and validates the beginning of the XML file
func (s *DiscogsParserService) validateAndLogXMLStructure(reader io.Reader, fileType string, log logger.Logger) error {
	// Create a buffered reader to peek at content
	bufReader := bufio.NewReader(reader)

	// Read first 2KB for structure analysis
	headerBytes := make([]byte, 2048)
	n, err := bufReader.Read(headerBytes)
	if err != nil && err != io.EOF {
		return fmt.Errorf("failed to read XML header: %w", err)
	}

	headerContent := string(headerBytes[:n])
	log.Info("XML file header validation",
		"bytesRead", n,
		"firstLine", s.getFirstLine(headerContent),
		"hasXMLDeclaration", strings.Contains(headerContent, "<?xml"),
		"fileType", fileType)

	// Check for expected root elements based on file type
	expectedRoot := ""
	expectedElement := ""
	switch fileType {
	case "labels":
		expectedRoot = "<labels>"
		expectedElement = "<label"
	case "artists":
		expectedRoot = "<artists>"
		expectedElement = "<artist"
	case "masters":
		expectedRoot = "<masters>"
		expectedElement = "<master"
	case "releases":
		expectedRoot = "<releases>"
		expectedElement = "<release"
	}

	hasRoot := strings.Contains(headerContent, expectedRoot)
	hasElement := strings.Contains(headerContent, expectedElement)

	log.Info("XML structure analysis",
		"expectedRoot", expectedRoot,
		"hasExpectedRoot", hasRoot,
		"expectedElement", expectedElement,
		"hasExpectedElement", hasElement,
		"sampleContent", s.getSampleContent(headerContent, 200))

	if !hasRoot && !hasElement {
		return fmt.Errorf("XML does not contain expected structure for %s: missing %s or %s", fileType, expectedRoot, expectedElement)
	}

	return nil
}

// getFirstLine extracts the first line from content
func (s *DiscogsParserService) getFirstLine(content string) string {
	lines := strings.Split(content, "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0])
	}
	return ""
}

// getSampleContent returns a cleaned sample of the content for logging
func (s *DiscogsParserService) getSampleContent(content string, maxLen int) string {
	// Remove excessive whitespace and newlines for cleaner logging
	cleaned := strings.ReplaceAll(content, "\n", " ")
	cleaned = strings.ReplaceAll(cleaned, "\t", " ")
	// Compress multiple spaces
	for strings.Contains(cleaned, "  ") {
		cleaned = strings.ReplaceAll(cleaned, "  ", " ")
	}

	if len(cleaned) > maxLen {
		return cleaned[:maxLen] + "..."
	}
	return cleaned
}

// formatAttributes formats XML attributes for logging
func (s *DiscogsParserService) formatAttributes(attrs []xml.Attr) string {
	if len(attrs) == 0 {
		return "none"
	}

	var attrStrs []string
	for _, attr := range attrs {
		attrStrs = append(attrStrs, fmt.Sprintf("%s=%q", attr.Name.Local, attr.Value))
	}
	return strings.Join(attrStrs, ", ")
}

// logSuccessfulRecord logs details of successfully parsed records for debugging
func (s *DiscogsParserService) logSuccessfulRecord(recordType string, discogsRecord interface{}, convertedRecord interface{}, recordNumber int, log logger.Logger) {
	// Convert to JSON for readable logging
	discogsJSON, _ := json.Marshal(discogsRecord)
	convertedJSON, _ := json.Marshal(convertedRecord)

	log.Info("Successful record parse sample",
		"recordType", recordType,
		"recordNumber", recordNumber,
		"discogsRecord", string(discogsJSON),
		"convertedRecord", string(convertedJSON))
}