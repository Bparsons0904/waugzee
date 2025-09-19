package services

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
	"waugzee/internal/imports"
	"waugzee/internal/logger"
	"waugzee/internal/models"
)

// EntityMessage represents a single entity sent through the processing channel
type EntityMessage struct {
	Type         string // "label", "artist", "master", "release"
	RawEntity    any    // *imports.Label, *imports.Artist, *imports.Master, *imports.Release
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

// ParseFileToChannel parses a Discogs XML file and sends raw entities to the provided channel
// This extracts ALL data from XML without conversion or memory optimization
func (s *DiscogsParserService) ParseFileToChannel(
	ctx context.Context,
	options ParseOptions,
	entityChan chan<- EntityMessage,
	completionChan chan<- CompletionMessage,
) error {
	if options.FilePath == "" || options.FileType == "" {
		return fmt.Errorf("filePath and fileType are required")
	}

	// Get entity configuration
	config, exists := entityConfigs[options.FileType]
	if !exists {
		return fmt.Errorf("unsupported file type: %s", options.FileType)
	}

	// Open and decompress the gzipped XML file
	file, err := os.Open(options.FilePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	// Parse the file and send to channel
	err = s.parseEntityFileToChannel(
		ctx,
		gzipReader,
		config.ElementName,
		entityChan,
	)

	// Send completion signal
	select {
	case completionChan <- CompletionMessage{
		ProcessingID: "", // Will be set by calling service
		FileType:     options.FileType,
		Completed:    err == nil,
	}:
	case <-ctx.Done():
		return ctx.Err()
	}

	return err
}

// parseEntityFileToChannel handles parsing and sends ALL raw entity data to channel
func (s *DiscogsParserService) parseEntityFileToChannel(
	ctx context.Context,
	reader io.Reader,
	elementName string,
	entityChan chan<- EntityMessage,
) error {
	decoder := xml.NewDecoder(reader)

	// Progress tracking
	processedCount := 0
	lastLogTime := time.Now()
	const logInterval = 10 * time.Second

	// Stream through the XML file
	for {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			s.log.Error("XML parsing error", "error", err)
			continue
		}

		// Look for entity start elements
		if startElement, ok := token.(xml.StartElement); ok &&
			startElement.Name.Local == elementName {

			// Decode the element and send RAW entity to channel
			switch elementName {
			case "label":
				var label imports.Label
				if err := decoder.DecodeElement(&label, &startElement); err != nil {
					s.log.Error("Failed to decode label", "error", err)
					continue
				}
				// Send RAW label with ALL fields preserved
				select {
				case entityChan <- EntityMessage{
					Type:      "label",
					RawEntity: &label,
				}:
				case <-ctx.Done():
					return ctx.Err()
				}

			case "artist":
				var artist imports.Artist
				if err := decoder.DecodeElement(&artist, &startElement); err != nil {
					s.log.Error("Failed to decode artist", "error", err)
					continue
				}
				// Send RAW artist with ALL fields preserved
				select {
				case entityChan <- EntityMessage{
					Type:      "artist",
					RawEntity: &artist,
				}:
				case <-ctx.Done():
					return ctx.Err()
				}

			case "master":
				var master imports.Master
				if err := decoder.DecodeElement(&master, &startElement); err != nil {
					s.log.Error("Failed to decode master", "error", err)
					continue
				}
				// Send RAW master with ALL fields preserved
				select {
				case entityChan <- EntityMessage{
					Type:      "master",
					RawEntity: &master,
				}:
				case <-ctx.Done():
					return ctx.Err()
				}

			case "release":
				var release imports.Release
				if err := decoder.DecodeElement(&release, &startElement); err != nil {
					s.log.Error("Failed to decode release", "error", err)
					continue
				}
				// Send RAW release with ALL fields preserved
				select {
				case entityChan <- EntityMessage{
					Type:      "release",
					RawEntity: &release,
				}:
				case <-ctx.Done():
					return ctx.Err()
				}
			}

			// Progress tracking for all successful entity processing
			processedCount++
			if time.Since(lastLogTime) >= logInterval {
				s.log.Info("Processing progress",
					"entityType", elementName,
					"processed", processedCount)
				lastLogTime = time.Now()
			}
		}
	}

	return nil
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

func (s *DiscogsParserService) convertDiscogsLabel(discogsLabel *imports.Label) *models.Label {
	// Skip labels with invalid data
	if discogsLabel.ID == 0 {
		s.log.Warn("Dropping label due to invalid ID", "discogsID", discogsLabel.ID, "name", discogsLabel.Name, "reason", "invalid ID")
		return nil
	}

	if len(discogsLabel.Name) == 0 {
		s.log.Warn("Dropping label due to empty name", "discogsID", discogsLabel.ID, "name", discogsLabel.Name, "reason", "empty name")
		return nil
	}

	// Single trim operation with length check
	name := strings.TrimSpace(discogsLabel.Name)
	if len(name) == 0 {
		s.log.Warn("Dropping label due to empty name after trim", "discogsID", discogsLabel.ID, "name", discogsLabel.Name, "reason", "empty name after trim")
		return nil
	}

	label := &models.Label{Name: name}

	// Set DiscogsID directly from Discogs ID
	label.DiscogsID = int64(discogsLabel.ID)

	return label
}

func (s *DiscogsParserService) convertDiscogsArtist(discogsArtist *imports.Artist) *models.Artist {
	// Skip artists with invalid data
	if discogsArtist.ID == 0 {
		s.log.Warn("Dropping artist due to invalid ID", "discogsID", discogsArtist.ID, "name", discogsArtist.Name, "reason", "invalid ID")
		return nil
	}

	if len(discogsArtist.Name) == 0 {
		s.log.Warn("Dropping artist due to empty name", "discogsID", discogsArtist.ID, "name", discogsArtist.Name, "reason", "empty name")
		return nil
	}

	// Single trim operation with length check
	name := strings.TrimSpace(discogsArtist.Name)
	if len(name) == 0 {
		s.log.Warn("Dropping artist due to empty name after trim", "discogsID", discogsArtist.ID, "name", discogsArtist.Name, "reason", "empty name after trim")
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

func (s *DiscogsParserService) convertDiscogsMaster(discogsMaster *imports.Master) *models.Master {
	if discogsMaster.ID == 0 {
		s.log.Warn("Dropping master due to invalid ID", "discogsID", discogsMaster.ID, "title", discogsMaster.Title, "reason", "invalid ID")
		return nil
	}

	if len(discogsMaster.Title) == 0 {
		s.log.Warn("Dropping master due to empty title", "discogsID", discogsMaster.ID, "title", discogsMaster.Title, "reason", "empty title")
		return nil
	}

	title := strings.TrimSpace(discogsMaster.Title)
	if len(title) == 0 {
		s.log.Warn("Dropping master due to empty title after trim", "discogsID", discogsMaster.ID, "title", discogsMaster.Title, "reason", "empty title after trim")
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
	if discogsRelease.ID == 0 {
		s.log.Warn("Dropping release due to invalid ID", "discogsID", discogsRelease.ID, "title", discogsRelease.Title, "reason", "invalid ID")
		return nil
	}

	if len(discogsRelease.Title) == 0 {
		s.log.Warn("Dropping release due to empty title", "discogsID", discogsRelease.ID, "title", discogsRelease.Title, "reason", "empty title")
		return nil
	}

	title := strings.TrimSpace(discogsRelease.Title)
	if len(title) == 0 {
		s.log.Warn("Dropping release due to empty title after trim", "discogsID", discogsRelease.ID, "title", discogsRelease.Title, "reason", "empty title after trim")
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
		s.log.Debug("Dropping non-vinyl release", "discogsID", discogsRelease.ID, "title", discogsRelease.Title, "format", release.Format, "reason", "non-vinyl format")
		return nil
	}

	// Master ID - Set Discogs ID which matches the FK constraint to masters.discogs_id
	if discogsRelease.MasterID > 0 {
		masterID := int64(discogsRelease.MasterID)
		release.MasterID = &masterID
	}

	// Label ID - Set Discogs ID which matches the FK constraint to labels.discogs_id
	if len(discogsRelease.Labels) > 0 && discogsRelease.Labels[0].ID > 0 {
		labelID := int64(discogsRelease.Labels[0].ID)
		release.LabelID = &labelID
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
func (s *DiscogsParserService) generateReleaseJSONBData(
	release *models.Release,
	discogsRelease *imports.Release,
) error {
	// Generate tracks JSON
	tracks := make([]map[string]any, 0, len(discogsRelease.TrackList))
	for _, track := range discogsRelease.TrackList {
		if track.Title == "" || track.Position == "" {
			continue // Skip invalid tracks
		}

		trackData := map[string]any{
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
	artists := make([]map[string]any, 0, len(discogsRelease.Artists))
	for _, artist := range discogsRelease.Artists {
		if artist.ID <= 0 {
			continue // Skip invalid artists
		}

		artistData := map[string]any{
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
