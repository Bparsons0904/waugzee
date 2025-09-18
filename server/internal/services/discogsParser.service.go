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
	"strconv"
	"strings"
	"time"
	"waugzee/internal/imports"
	"waugzee/internal/logger"
	"waugzee/internal/models"
)

// EntityMessage represents a single entity sent through the processing channel
type EntityMessage struct {
	Type         string      // "label", "artist", "master", "release"
	RawEntity    interface{} // *imports.Label, *imports.Artist, *imports.Master, *imports.Release
	ProcessingID string
}

// CompletionMessage signals that file processing has been completed
type CompletionMessage struct {
	ProcessingID string
	FileType     string
	Completed    bool
}

// ParseResult represents the result of parsing a Discogs XML file
type ParseResult struct {
	TotalRecords     int
	ProcessedRecords int
	ErroredRecords   int
	Errors           []string
	ParsedLabels     []*models.Label
	ParsedArtists    []*models.Artist
	ParsedMasters    []*models.Master
	ParsedReleases   []*models.Release
}

// ParseOptions configures parsing behavior
type ParseOptions struct {
	FilePath string
	FileType string // "labels", "artists", "masters", "releases"
}

// EntityConfig defines the configuration for parsing different entity types
type EntityConfig struct {
	ElementName   string
	SetResultFunc func(*ParseResult, any)
}

// Entity configurations for each supported file type
var entityConfigs = map[string]EntityConfig{
	"labels": {
		ElementName: "label",
		SetResultFunc: func(result *ParseResult, converted any) {
			if label, ok := converted.(*models.Label); ok && label != nil {
				result.ParsedLabels = append(result.ParsedLabels, label)
			}
		},
	},
	"artists": {
		ElementName: "artist",
		SetResultFunc: func(result *ParseResult, converted any) {
			if artist, ok := converted.(*models.Artist); ok && artist != nil {
				result.ParsedArtists = append(result.ParsedArtists, artist)
			}
		},
	},
	"masters": {
		ElementName: "master",
		SetResultFunc: func(result *ParseResult, converted any) {
			if master, ok := converted.(*models.Master); ok && master != nil {
				result.ParsedMasters = append(result.ParsedMasters, master)
			}
		},
	},
	"releases": {
		ElementName: "release",
		SetResultFunc: func(result *ParseResult, converted any) {
			if release, ok := converted.(*models.Release); ok && release != nil {
				result.ParsedReleases = append(result.ParsedReleases, release)
			}
		},
	},
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
func (s *DiscogsParserService) ParseFile(
	ctx context.Context,
	options ParseOptions,
) (*ParseResult, error) {
	if options.FilePath == "" || options.FileType == "" {
		return nil, fmt.Errorf("filePath and fileType are required")
	}

	// Get entity configuration
	config, exists := entityConfigs[options.FileType]
	if !exists {
		return nil, fmt.Errorf("unsupported file type: %s", options.FileType)
	}

	// Open and decompress the gzipped XML file
	file, err := os.Open(options.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	// Parse the file
	return s.parseEntityFile(ctx, gzipReader, config)
}

// parseEntityFile handles parsing of any entity type using the provided configuration
func (s *DiscogsParserService) parseEntityFile(
	ctx context.Context,
	reader io.Reader,
	config EntityConfig,
) (*ParseResult, error) {
	decoder := xml.NewDecoder(reader)

	result := &ParseResult{
		Errors:         make([]string, 0),
		ParsedLabels:   make([]*models.Label, 0),
		ParsedArtists:  make([]*models.Artist, 0),
		ParsedMasters:  make([]*models.Master, 0),
		ParsedReleases: make([]*models.Release, 0),
	}

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

		// Look for entity start elements
		if startElement, ok := token.(xml.StartElement); ok &&
			startElement.Name.Local == config.ElementName {
			result.TotalRecords++

			// Decode the element based on entity type
			var converted any

			switch config.ElementName {
			case "label":
				var label imports.Label
				if err := decoder.DecodeElement(&label, &startElement); err != nil {
					result.Errors = append(
						result.Errors,
						fmt.Sprintf("Failed to decode label: %v", err),
					)
					result.ErroredRecords++
					continue
				}
				converted = s.convertDiscogsLabel(&label)
			case "artist":
				var artist imports.Artist
				if err := decoder.DecodeElement(&artist, &startElement); err != nil {
					result.Errors = append(
						result.Errors,
						fmt.Sprintf("Failed to decode artist: %v", err),
					)
					result.ErroredRecords++
					continue
				}
				converted = s.convertDiscogsArtist(&artist)
			case "master":
				var master imports.Master
				if err := decoder.DecodeElement(&master, &startElement); err != nil {
					result.Errors = append(
						result.Errors,
						fmt.Sprintf("Failed to decode master: %v", err),
					)
					result.ErroredRecords++
					continue
				}
				converted = s.convertDiscogsMaster(&master)
			case "release":
				var release imports.Release
				if err := decoder.DecodeElement(&release, &startElement); err != nil {
					result.Errors = append(
						result.Errors,
						fmt.Sprintf("Failed to decode release: %v", err),
					)
					result.ErroredRecords++
					continue
				}
				converted = s.convertDiscogsRelease(&release)
			}

			// Check if conversion was successful
			if converted != nil {
				config.SetResultFunc(result, converted)
				result.ProcessedRecords++
			} else {
				result.ErroredRecords++
			}
		}
	}

	return result, nil
}

// handleDecodeError handles XML decode errors consistently
func (s *DiscogsParserService) handleDecodeError(
	result *ParseResult,
	err error,
	startElement xml.StartElement,
	log logger.Logger,
) {
	errorMsg := fmt.Sprintf(
		"Failed to decode %s element at record %d: %v",
		startElement.Name.Local,
		result.TotalRecords,
		err,
	)
	log.Error("XML decode error",
		"error", err,
		"recordNumber", result.TotalRecords,
		"elementName", startElement.Name.Local,
		"elementAttrs", s.formatAttributes(startElement.Attr))
	result.Errors = append(result.Errors, errorMsg)
	result.ErroredRecords++
}

// logProgress logs parsing progress with performance metrics
func (s *DiscogsParserService) logProgress(
	result *ParseResult,
	progressInterval int,
	startTime time.Time,
	lastProgressTime *time.Time,
	log logger.Logger,
) {
	currentTime := time.Now()
	elapsed := currentTime.Sub(*lastProgressTime)
	recordsPerSecond := float64(progressInterval) / elapsed.Seconds()
	totalElapsed := currentTime.Sub(startTime)

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	successfulRecords := result.TotalRecords - result.ErroredRecords
	log.Info(
		"Parsing progress",
		"totalRecords",
		result.TotalRecords,
		"processedRecords",
		result.ProcessedRecords,
		"erroredRecords",
		result.ErroredRecords,
		"successfulRecords",
		successfulRecords,
		"recordsPerSecond",
		fmt.Sprintf("%.1f", recordsPerSecond),
		"totalElapsedMs",
		totalElapsed.Milliseconds(),
		"memoryUsageMB",
		memStats.Alloc/1024/1024,
		"successRate",
		fmt.Sprintf(
			"%.2f%%",
			float64(successfulRecords)/float64(result.TotalRecords)*100,
		),
	)

	*lastProgressTime = currentTime
}

// logFinalSummary logs final parsing results and errors
func (s *DiscogsParserService) logFinalSummary(
	result *ParseResult,
	entityType string,
	startTime time.Time,
	log logger.Logger,
) {
	totalElapsed := time.Since(startTime)
	overallRecordsPerSecond := float64(result.TotalRecords) / totalElapsed.Seconds()
	successfulRecords := result.TotalRecords - result.ErroredRecords

	log.Info(
		fmt.Sprintf(
			"%s file parsing completed",
			strings.ToUpper(string(entityType[0]))+entityType[1:],
		),
		"total",
		result.TotalRecords,
		"processed",
		result.ProcessedRecords,
		"errors",
		result.ErroredRecords,
		"successful",
		successfulRecords,
		"totalElapsedMs",
		totalElapsed.Milliseconds(),
		"overallRecordsPerSecond",
		fmt.Sprintf("%.1f", overallRecordsPerSecond),
		"successRate",
		fmt.Sprintf("%.2f%%", float64(successfulRecords)/float64(result.TotalRecords)*100),
		"errorSampleCount",
		len(result.Errors),
	)

	// Log first few errors for debugging
	if len(result.Errors) > 0 {
		maxErrorsToLog := min(len(result.Errors), 5)
		log.Error("Sample parsing errors",
			"sampleErrors", result.Errors[:maxErrorsToLog],
			"totalErrors", len(result.Errors))
	}
}

func (s *DiscogsParserService) convertDiscogsLabel(discogsLabel *imports.Label) *models.Label {
	// Skip labels with invalid data
	if discogsLabel.ID == 0 || len(discogsLabel.Name) == 0 {
		return nil
	}

	// Single trim operation with length check
	name := strings.TrimSpace(discogsLabel.Name)
	if len(name) == 0 {
		return nil
	}

	label := &models.Label{Name: name}

	// Set DiscogsID directly from Discogs ID
	label.DiscogsID = int64(discogsLabel.ID)

	return label
}

func (s *DiscogsParserService) convertDiscogsArtist(discogsArtist *imports.Artist) *models.Artist {
	// Skip artists with invalid data
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

	// Set DiscogsID directly from Discogs ID
	artist.DiscogsID = int64(discogsArtist.ID)

	return artist
}

func (s *DiscogsParserService) convertDiscogsImage(
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

	// Convert imageableID from string to int64
	imageableIDInt64, err := strconv.ParseInt(imageableID, 10, 64)
	if err != nil {
		return nil // Skip invalid imageableID
	}

	image := &models.Image{
		URL:           discogsImage.URI,
		ImageableID:   imageableIDInt64,
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

func (s *DiscogsParserService) convertDiscogsMaster(discogsMaster *imports.Master) *models.Master {
	// Skip masters with invalid data
	if discogsMaster.ID == 0 || len(discogsMaster.Title) == 0 {
		return nil
	}

	// Single trim operation with length check
	title := strings.TrimSpace(discogsMaster.Title)
	if len(title) == 0 {
		return nil
	}

	master := &models.Master{Title: title}

	// Set DiscogsID directly from Discogs ID
	master.DiscogsID = int64(discogsMaster.ID)

	// Set essential fields only
	if discogsMaster.MainRelease != 0 {
		mainRelease := int(discogsMaster.MainRelease)
		master.MainRelease = &mainRelease
	}

	if discogsMaster.Year != 0 {
		master.Year = &discogsMaster.Year
	}

	return master
}

func (s *DiscogsParserService) convertDiscogsRelease(
	discogsRelease *imports.Release,
) *models.Release {
	// Skip releases with invalid data
	if discogsRelease.ID == 0 || len(discogsRelease.Title) == 0 {
		return nil
	}

	title := strings.TrimSpace(discogsRelease.Title)
	if len(title) == 0 {
		return nil
	}

	release := &models.Release{
		DiscogsID: int64(discogsRelease.ID),
		Title:     title,
		Format:    models.FormatVinyl, // Default format
	}

	// Basic fields
	if len(discogsRelease.Country) > 0 {
		if country := strings.TrimSpace(discogsRelease.Country); len(country) > 0 {
			release.Country = &country
		}
	}

	// Year parsing
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

	// Format mapping
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

	// VINYL-ONLY FILTERING: Skip non-vinyl releases to dramatically reduce processing volume
	if release.Format != models.FormatVinyl {
		return nil
	}

	// Track count
	if trackCount := len(discogsRelease.TrackList); trackCount > 0 {
		release.TrackCount = &trackCount
	}

	// Set primary image URL from first image if available
	if len(discogsRelease.Images) > 0 && discogsRelease.Images[0].URI != "" {
		imageURL := strings.TrimSpace(discogsRelease.Images[0].URI)
		if imageURL != "" {
			release.ImageURL = &imageURL
		}
	}

	// Generate JSONB data for tracks, artists, and genres
	if err := s.generateReleaseJSONBData(release, discogsRelease); err != nil {
		// Log error but don't fail the entire release
		s.log.Warn("Failed to generate JSONB data", "releaseID", discogsRelease.ID, "error", err)
	}

	return release
}

// generateReleaseJSONBData creates JSONB data for tracks, artists, and genres
func (s *DiscogsParserService) generateReleaseJSONBData(release *models.Release, discogsRelease *imports.Release) error {
	// Generate tracks JSON
	tracks := make([]map[string]interface{}, 0, len(discogsRelease.TrackList))
	for _, track := range discogsRelease.TrackList {
		if track.Title == "" || track.Position == "" {
			continue // Skip invalid tracks
		}

		trackData := map[string]interface{}{
			"position": strings.TrimSpace(track.Position),
			"title":    strings.TrimSpace(track.Title),
		}

		// Add duration if available
		if track.Duration != "" {
			if duration := s.parseDurationToSeconds(track.Duration); duration > 0 {
				trackData["duration"] = duration
			}
		}

		tracks = append(tracks, trackData)
	}

	if len(tracks) > 0 {
		tracksJSON, err := json.Marshal(tracks)
		if err != nil {
			return fmt.Errorf("failed to marshal tracks JSON: %w", err)
		}
		release.TracksJSON = tracksJSON
	}

	// Generate artists JSON
	artists := make([]map[string]interface{}, 0, len(discogsRelease.Artists))
	for _, artist := range discogsRelease.Artists {
		if artist.ID <= 0 {
			continue // Skip invalid artists
		}

		artistData := map[string]interface{}{
			"id":   artist.ID,
			"name": strings.TrimSpace(artist.Name),
		}

		artists = append(artists, artistData)
	}

	if len(artists) > 0 {
		artistsJSON, err := json.Marshal(artists)
		if err != nil {
			return fmt.Errorf("failed to marshal artists JSON: %w", err)
		}
		release.ArtistsJSON = artistsJSON
	}

	// Generate genres JSON
	genres := make([]string, 0, len(discogsRelease.Genres))
	for _, genre := range discogsRelease.Genres {
		if trimmedGenre := strings.TrimSpace(genre); trimmedGenre != "" {
			genres = append(genres, trimmedGenre)
		}
	}

	if len(genres) > 0 {
		genresJSON, err := json.Marshal(genres)
		if err != nil {
			return fmt.Errorf("failed to marshal genres JSON: %w", err)
		}
		release.GenresJSON = genresJSON
	}

	return nil
}

// parseDurationToSeconds converts duration string formats to seconds
// Supports formats like "4:23", "1:23:45", "123" (seconds only)
func (s *DiscogsParserService) parseDurationToSeconds(duration string) int {
	if duration == "" {
		return 0
	}

	// If it's just a number, treat it as seconds
	if seconds, err := strconv.Atoi(duration); err == nil {
		return seconds
	}

	// Handle time format like "4:23" or "1:23:45"
	parts := strings.Split(duration, ":")
	if len(parts) < 2 {
		return 0
	}

	var totalSeconds int

	// Parse based on number of parts
	switch len(parts) {
	case 2: // mm:ss
		minutes, err1 := strconv.Atoi(parts[0])
		seconds, err2 := strconv.Atoi(parts[1])
		if err1 != nil || err2 != nil {
			return 0
		}
		totalSeconds = minutes*60 + seconds

	case 3: // hh:mm:ss
		hours, err1 := strconv.Atoi(parts[0])
		minutes, err2 := strconv.Atoi(parts[1])
		seconds, err3 := strconv.Atoi(parts[2])
		if err1 != nil || err2 != nil || err3 != nil {
			return 0
		}
		totalSeconds = hours*3600 + minutes*60 + seconds

	default:
		return 0
	}

	return totalSeconds
}

// validateAndLogXMLStructure reads and validates the beginning of the XML file
func (s *DiscogsParserService) validateAndLogXMLStructure(
	reader io.Reader,
	fileType string,
	log logger.Logger,
) error {
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
		return fmt.Errorf(
			"XML does not contain expected structure for %s: missing %s or %s",
			fileType,
			expectedRoot,
			expectedElement,
		)
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
func (s *DiscogsParserService) logSuccessfulRecord(
	recordType string,
	discogsRecord any,
	convertedRecord any,
	recordNumber int,
	log logger.Logger,
) {
	// Convert to JSON for readable logging
	discogsJSON, _ := json.Marshal(discogsRecord)
	convertedJSON, _ := json.Marshal(convertedRecord)

	log.Info("Successful record parse sample",
		"recordType", recordType,
		"recordNumber", recordNumber,
		"discogsRecord", string(discogsJSON),
		"convertedRecord", string(convertedJSON))
}

// findOrCreateGenre finds an existing genre by name or creates a new one
func (s *DiscogsParserService) findOrCreateGenre(name string) *models.Genre {
	if name == "" {
		return nil
	}

	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return nil
	}

	// For now, just create a new genre object without database interaction
	// The database layer will handle deduplication through unique constraints
	return &models.Genre{
		Name: trimmedName,
	}
}

// convertDiscogsArtists converts Discogs artist data to our Artist models
func (s *DiscogsParserService) convertDiscogsArtists(
	discogsArtists []imports.Artist,
) []models.Artist {
	if len(discogsArtists) == 0 {
		return nil
	}

	var artists []models.Artist
	for _, discogsArtist := range discogsArtists {
		if discogsArtist.Name == "" {
			continue
		}

		name := strings.TrimSpace(discogsArtist.Name)
		if name == "" {
			continue
		}

		artist := models.Artist{
			Name:     name,
			IsActive: true,
		}

		// Set DiscogsID from Discogs ID if available
		if discogsArtist.ID > 0 {
			artist.DiscogsID = int64(discogsArtist.ID)
		}

		// Note: Biography field not available in current Artist model
		// if len(discogsArtist.Profile) > 0 {
		//	profile := strings.TrimSpace(discogsArtist.Profile)
		//	if len(profile) > 0 {
		//		artist.Biography = &profile
		//	}
		// }

		artists = append(artists, artist)
	}

	return artists
}

// convertDiscogsGenres converts Discogs genres and styles to our Genre models
func (s *DiscogsParserService) convertDiscogsGenres(
	genres []string,
	styles []string,
) []models.Genre {
	var result []models.Genre
	genreMap := make(map[string]bool) // Track duplicates

	// Add genres
	for _, genre := range genres {
		if genre == "" {
			continue
		}
		name := strings.TrimSpace(genre)
		if name == "" || genreMap[name] {
			continue
		}
		genreMap[name] = true
		result = append(result, models.Genre{Name: name})
	}

	// Add styles (as sub-genres)
	for _, style := range styles {
		if style == "" {
			continue
		}
		name := strings.TrimSpace(style)
		if name == "" || genreMap[name] {
			continue
		}
		genreMap[name] = true
		result = append(result, models.Genre{Name: name})
	}

	return result
}

