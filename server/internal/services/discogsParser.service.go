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

	"github.com/google/uuid"
)

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
	FilePath     string
	FileType     string                             // "labels", "artists", "masters", "releases"
	BatchSize    int                                // Optional: batch size for processing (default: 2000)
	MaxRecords   int                                // Optional: limit total records parsed (0 = no limit)
	ProgressFunc func(processed, total, errors int) // Optional: progress callback
}

// EntityConfig defines the configuration for parsing different entity types
type EntityConfig struct {
	ElementName     string
	ConvertFunc     func(interface{}) interface{}
	SetResultFunc   func(*ParseResult, interface{})
	DebugLogFields  func(interface{}) map[string]interface{}
}

// Entity configurations for each supported file type
var entityConfigs = map[string]EntityConfig{
	"labels": {
		ElementName: "label",
		ConvertFunc: func(raw interface{}) interface{} {
			return raw.(*imports.Label)
		},
		SetResultFunc: func(result *ParseResult, converted interface{}) {
			if label, ok := converted.(*models.Label); ok && label != nil {
				result.ParsedLabels = append(result.ParsedLabels, label)
			}
		},
		DebugLogFields: func(raw interface{}) map[string]interface{} {
			label := raw.(*imports.Label)
			return map[string]interface{}{
				"parsedID":          label.ID,
				"parsedName":        label.Name,
				"parsedContactInfo": label.ContactInfo,
				"parsedProfile":     label.Profile,
				"parsedParentLabel": label.ParentLabel,
				"parsedURLs":        label.URLs,
				"parsedSubLabels":   label.SubLabels,
			}
		},
	},
	"artists": {
		ElementName: "artist",
		ConvertFunc: func(raw interface{}) interface{} {
			return raw.(*imports.Artist)
		},
		SetResultFunc: func(result *ParseResult, converted interface{}) {
			if artist, ok := converted.(*models.Artist); ok && artist != nil {
				result.ParsedArtists = append(result.ParsedArtists, artist)
			}
		},
		DebugLogFields: func(raw interface{}) map[string]interface{} {
			artist := raw.(*imports.Artist)
			return map[string]interface{}{
				"parsedID":       artist.ID,
				"parsedName":     artist.Name,
				"parsedRealName": artist.RealName,
				"parsedProfile":  artist.Profile,
				"parsedURLs":     artist.URLs,
				"parsedNameVars": artist.NameVars,
				"parsedAliases":  artist.Aliases,
				"parsedMembers":  artist.Members,
				"parsedGroups":   artist.Groups,
			}
		},
	},
	"masters": {
		ElementName: "master",
		ConvertFunc: func(raw interface{}) interface{} {
			return raw.(*imports.Master)
		},
		SetResultFunc: func(result *ParseResult, converted interface{}) {
			if master, ok := converted.(*models.Master); ok && master != nil {
				result.ParsedMasters = append(result.ParsedMasters, master)
			}
		},
		DebugLogFields: func(raw interface{}) map[string]interface{} {
			master := raw.(*imports.Master)
			return map[string]interface{}{
				"parsedID":          master.ID,
				"parsedMainRelease": master.MainRelease,
				"parsedTitle":       master.Title,
				"parsedYear":        master.Year,
				"parsedNotes":       master.Notes,
				"parsedDataQuality": master.DataQuality,
				"parsedArtists":     master.Artists,
				"parsedGenres":      master.Genres,
				"parsedStyles":      master.Styles,
				"parsedVideos":      master.Videos,
			}
		},
	},
	"releases": {
		ElementName: "release",
		ConvertFunc: func(raw interface{}) interface{} {
			return raw.(*imports.Release)
		},
		SetResultFunc: func(result *ParseResult, converted interface{}) {
			if release, ok := converted.(*models.Release); ok && release != nil {
				result.ParsedReleases = append(result.ParsedReleases, release)
			}
		},
		DebugLogFields: func(raw interface{}) map[string]interface{} {
			release := raw.(*imports.Release)
			return map[string]interface{}{
				"parsedID":          release.ID,
				"parsedStatus":      release.Status,
				"parsedTitle":       release.Title,
				"parsedCountry":     release.Country,
				"parsedReleased":    release.Released,
				"parsedNotes":       release.Notes,
				"parsedDataQuality": release.DataQuality,
				"parsedMasterID":    release.MasterID,
				"parsedArtists":     release.Artists,
				"parsedLabels":      release.Labels,
				"parsedFormats":     release.Formats,
				"parsedGenres":      release.Genres,
				"parsedStyles":      release.Styles,
				"parsedTracklist":   release.TrackList,
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
	log := s.log.Function("ParseFile")

	if options.FilePath == "" {
		return nil, log.Err("file path is required", nil)
	}

	if options.FileType == "" {
		return nil, log.Err("file type is required", nil)
	}

	// Get entity configuration
	config, exists := entityConfigs[options.FileType]
	if !exists {
		return nil, log.Err("unsupported file type", nil, "fileType", options.FileType)
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
		return nil, log.Err(
			"failed to create gzip reader - file may not be gzipped",
			err,
			"filePath",
			options.FilePath,
		)
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
		return nil, log.Err(
			"failed to recreate gzip reader for parsing",
			err,
			"filePath",
			options.FilePath,
		)
	}
	defer gzipReader.Close()

	// Use generic parser with entity-specific configuration
	return s.parseEntityFile(ctx, gzipReader, options, config, log)
}

// parseEntityFile handles parsing of any entity type using the provided configuration
func (s *DiscogsParserService) parseEntityFile(
	ctx context.Context,
	reader io.Reader,
	options ParseOptions,
	config EntityConfig,
	log logger.Logger,
) (*ParseResult, error) {
	decoder := xml.NewDecoder(reader)

	result := &ParseResult{
		Errors:         make([]string, 0),
		ParsedLabels:   make([]*models.Label, 0),
		ParsedArtists:  make([]*models.Artist, 0),
		ParsedMasters:  make([]*models.Master, 0),
		ParsedReleases: make([]*models.Release, 0),
	}

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

	log.Info("Starting entity parsing",
		"entityType", config.ElementName,
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

		// Look for entity start elements
		if startElement, ok := token.(xml.StartElement); ok && startElement.Name.Local == config.ElementName {
			// Increment total record count for each record we attempt to process
			result.TotalRecords++

			// Check max records limit before processing - count all attempted records
			if options.MaxRecords > 0 && result.TotalRecords > options.MaxRecords {
				log.Info(
					"Reached max records limit - stopping early",
					"maxRecords",
					options.MaxRecords,
					"totalAttempted",
					result.TotalRecords-1,
				)
				result.TotalRecords-- // Adjust count since we're not processing this record
				break
			}

			// Decode the element based on entity type
			var discogsEntity interface{}
			var converted interface{}

			switch config.ElementName {
			case "label":
				var label imports.Label
				if err := decoder.DecodeElement(&label, &startElement); err != nil {
					s.handleDecodeError(result, err, startElement, log)
					continue
				}
				discogsEntity = &label
				converted = s.convertDiscogsLabel(&label)
			case "artist":
				var artist imports.Artist
				if err := decoder.DecodeElement(&artist, &startElement); err != nil {
					s.handleDecodeError(result, err, startElement, log)
					continue
				}
				discogsEntity = &artist
				converted = s.convertDiscogsArtist(&artist)
			case "master":
				var master imports.Master
				if err := decoder.DecodeElement(&master, &startElement); err != nil {
					s.handleDecodeError(result, err, startElement, log)
					continue
				}
				discogsEntity = &master
				converted = s.convertDiscogsMaster(&master)
			case "release":
				var release imports.Release
				if err := decoder.DecodeElement(&release, &startElement); err != nil {
					s.handleDecodeError(result, err, startElement, log)
					continue
				}
				discogsEntity = &release
				converted = s.convertDiscogsRelease(&release)
			}

			// Log detailed XML structure for first few records to debug parsing issues
			if result.TotalRecords <= 2 {
				debugFields := config.DebugLogFields(discogsEntity)
				debugFields["recordNumber"] = result.TotalRecords
				debugFields["elementName"] = startElement.Name.Local
				debugFields["elementAttrs"] = s.formatAttributes(startElement.Attr)
				log.Info("Detailed XML structure for debugging", debugFields)
			}

			// Check if conversion was successful
			if converted == nil {
				log.Warn("Failed to convert Discogs entity",
					"entityType", config.ElementName,
					"recordNumber", result.TotalRecords)
				result.ErroredRecords++
				continue
			}

			// Log successful parse samples
			if successfulSamples < maxSamplesToLog {
				s.logSuccessfulRecord(config.ElementName, discogsEntity, converted, result.TotalRecords, log)
				successfulSamples++
			}

			// Add to appropriate result collection
			config.SetResultFunc(result, converted)
			result.ProcessedRecords++

			// Call progress callback if provided
			if options.ProgressFunc != nil && result.ProcessedRecords%batchSize == 0 {
				options.ProgressFunc(
					result.ProcessedRecords,
					result.TotalRecords,
					result.ErroredRecords,
				)
			}

			// Log progress every N records
			if result.TotalRecords%progressInterval == 0 {
				s.logProgress(result, progressInterval, startTime, &lastProgressTime, log)
			}
		}
	}

	// Final progress and summary
	s.logFinalSummary(result, config.ElementName, startTime, log)

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
		fmt.Sprintf("%s file parsing completed", strings.ToUpper(string(entityType[0]))+entityType[1:]),
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
		maxErrorsToLog := 5
		if len(result.Errors) < maxErrorsToLog {
			maxErrorsToLog = len(result.Errors)
		}
		log.Error("Sample parsing errors",
			"sampleErrors", result.Errors[:maxErrorsToLog],
			"totalErrors", len(result.Errors))
	}
}

// NOTE: The following duplicate parsing methods (parseLabelsFile, parseArtistsFile,
// parseMastersFile, parseReleasesFile) have been replaced by the generic
// parseEntityFile method above. This eliminates ~800 lines of duplicated code.


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

	// Set profile if available
	if profile := strings.TrimSpace(discogsLabel.Profile); len(profile) > 0 {
		label.Profile = &profile
	}

	// Set website from first URL if available
	if len(discogsLabel.URLs) > 0 {
		if firstURL := strings.TrimSpace(discogsLabel.URLs[0]); len(firstURL) > 0 {
			label.Website = &firstURL
		}
	}

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

	// Convert genres
	for _, genreName := range discogsMaster.Genres {
		if genre := s.findOrCreateGenre(genreName); genre != nil {
			master.Genres = append(master.Genres, *genre)
		}
	}

	// Convert styles as genres (Discogs treats them as sub-genres)
	for _, styleName := range discogsMaster.Styles {
		if genre := s.findOrCreateGenre(styleName); genre != nil {
			master.Genres = append(master.Genres, *genre)
		}
	}

	// Convert artists
	for _, discogsArtist := range discogsMaster.Artists {
		if artist := s.convertDiscogsArtist(&discogsArtist); artist != nil {
			master.Artists = append(master.Artists, *artist)
		}
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
		Title:     title,
		DiscogsID: int64(discogsRelease.ID),
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

	// Track count
	if trackCount := len(discogsRelease.TrackList); trackCount > 0 {
		release.TrackCount = &trackCount
	}

	// Catalog number from first label
	if len(discogsRelease.Labels) > 0 && len(discogsRelease.Labels[0].CatalogNo) > 0 {
		catalogNo := strings.TrimSpace(discogsRelease.Labels[0].CatalogNo)
		if len(catalogNo) > 0 {
			release.CatalogNumber = &catalogNo
		}
	}

	// Convert Artists
	release.Artists = s.convertDiscogsArtists(discogsRelease.Artists)

	// Convert Genres (combining genres and styles)
	release.Genres = s.convertDiscogsGenres(discogsRelease.Genres, discogsRelease.Styles)

	// Convert Tracks
	release.Tracks = s.convertDiscogsTracks(discogsRelease.TrackList, release.ID)

	// Set primary image URL from first image if available
	if len(discogsRelease.Images) > 0 && discogsRelease.Images[0].URI != "" {
		imageURL := strings.TrimSpace(discogsRelease.Images[0].URI)
		if imageURL != "" {
			release.ImageURL = &imageURL
		}
	}

	// Note: Detailed images are handled separately via the polymorphic system
	// They will be stored as separate Image models linked to this release

	return release
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
	discogsRecord interface{},
	convertedRecord interface{},
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

		// Set Discogs ID if available
		if discogsArtist.ID > 0 {
			discogsID := int64(discogsArtist.ID)
			artist.DiscogsID = &discogsID
		}

		// Set biography from profile if available
		if len(discogsArtist.Profile) > 0 {
			profile := strings.TrimSpace(discogsArtist.Profile)
			if len(profile) > 0 {
				artist.Biography = &profile
			}
		}

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

// convertDiscogsTracks converts Discogs track data to our Track models
func (s *DiscogsParserService) convertDiscogsTracks(
	discogsTracks []imports.Track,
	releaseID uuid.UUID,
) []models.Track {
	if len(discogsTracks) == 0 {
		return nil
	}

	var tracks []models.Track
	for _, discogsTrack := range discogsTracks {
		if discogsTrack.Title == "" || discogsTrack.Position == "" {
			continue
		}

		title := strings.TrimSpace(discogsTrack.Title)
		position := strings.TrimSpace(discogsTrack.Position)
		if title == "" || position == "" {
			continue
		}

		track := models.Track{
			ReleaseID: releaseID,
			Position:  position,
			Title:     title,
		}

		// Parse duration if available (format: "4:45")
		if len(discogsTrack.Duration) > 0 {
			duration := strings.TrimSpace(discogsTrack.Duration)
			if len(duration) > 0 {
				if seconds := s.parseDuration(duration); seconds > 0 {
					track.Duration = &seconds
				}
			}
		}

		tracks = append(tracks, track)
	}

	return tracks
}


// parseDuration converts duration string (e.g., "4:45") to seconds
func (s *DiscogsParserService) parseDuration(duration string) int {
	parts := strings.Split(duration, ":")
	if len(parts) != 2 {
		return 0
	}

	var minutes, seconds int
	if _, err := fmt.Sscanf(parts[0], "%d", &minutes); err != nil {
		return 0
	}
	if _, err := fmt.Sscanf(parts[1], "%d", &seconds); err != nil {
		return 0
	}

	if minutes < 0 || seconds < 0 || seconds >= 60 {
		return 0
	}

	return minutes*60 + seconds
}

