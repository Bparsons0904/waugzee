package services

import (
	"compress/gzip"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"
	"waugzee/internal/database"
	"waugzee/internal/logger"
	"waugzee/internal/models"
	"waugzee/internal/repositories"
	"waugzee/internal/types"

	"gorm.io/gorm"
)

type DiscogsXMLParserService struct {
	log   logger.Logger
	repos repositories.Repository
	db    database.DB
}

func NewDiscogsXMLParserService(
	repos repositories.Repository,
	db database.DB,
) *DiscogsXMLParserService {
	return &DiscogsXMLParserService{
		log:   logger.New("discogsXMLParser"),
		repos: repos,
		db:    db,
	}
}

const (
	DISCOG_URL     = "https://www.discogs.com/%s/%d"
	DISCOG_API_URL = "https://api.discogs.com/%s/%d"
)

// EntityProcessorConfig holds configuration for generic entity processing
type EntityProcessorConfig[TXMLType any, TModelType any] struct {
	FilePath       string
	ElementName    string
	EntityTypeName string
	ChannelSize    int
	BatchSize      int
	ConvertFunc    func(TXMLType) *TModelType
	UpsertFunc     func(ctx context.Context, db *gorm.DB, entities []*TModelType) error
}

// ProcessXMLEntities is a generic function that processes XML entities with configurable conversion and batch operations
func ProcessXMLEntities[TXMLType any, TModelType any](
	ctx context.Context,
	config EntityProcessorConfig[TXMLType, TModelType],
	db database.DB,
	log logger.Logger,
) error {
	processingLog := log.Function("ProcessXMLEntities").With("entityType", config.EntityTypeName)

	// Create channel for streaming XML entities
	xmlChan := make(chan TXMLType, config.ChannelSize)

	// Start XML parsing in goroutine
	go func() {
		defer close(xmlChan)
		err := ParseXMLGeneric(
			ctx,
			config.FilePath,
			config.ElementName,
			xmlChan,
			0, // No limit = 0
			// 50_000, // No limit = 0
			processingLog,
		)
		if err != nil {
			processingLog.Error("Failed to parse XML", "error", err)
		}
		processingLog.Info("XML parsing goroutine completed")
	}()

	// Process entities in batches
	processedCount := 0
	var entities []*TModelType

	for xmlEntity := range xmlChan {
		processedCount++

		// Convert XML entity to model using provided function
		modelEntity := config.ConvertFunc(xmlEntity)
		entities = append(entities, modelEntity)

		// Process batch when it reaches the configured size
		if len(entities) >= config.BatchSize {
			if err := config.UpsertFunc(ctx, db.SQLWithContext(ctx), entities); err != nil {
				processingLog.Error(
					"Failed to upsert batch",
					"error",
					err,
					"batchSize",
					len(entities),
				)
			} else {
				processingLog.Info("Processed batch", "batchSize", len(entities), "totalProcessed", processedCount)
			}
			entities = []*TModelType{}
		}
	}

	// Process any remaining entities in final batch
	if len(entities) > 0 {
		if err := config.UpsertFunc(ctx, db.SQLWithContext(ctx), entities); err != nil {
			processingLog.Error(
				"Failed to upsert final batch",
				"error",
				err,
				"batchSize",
				len(entities),
			)
		} else {
			processingLog.Info("Processed final batch", "batchSize", len(entities), "totalProcessed", processedCount)
		}
	}

	processingLog.Info(
		"Entity processing completed",
		"entityType",
		config.EntityTypeName,
		"totalProcessed",
		processedCount,
	)
	return nil
}

// convertXMLLabelToModel converts XML Label to database Label model
func (s *DiscogsXMLParserService) convertXMLLabelToModel(xmlLabel types.Label) *models.Label {
	resourceURL := fmt.Sprintf(DISCOG_API_URL, "labels", xmlLabel.ID)
	uri := fmt.Sprintf(DISCOG_URL, "labels", xmlLabel.ID)

	return &models.Label{
		BaseDiscogModel: models.BaseDiscogModel{
			ID: xmlLabel.ID,
		},
		Profile:     &xmlLabel.Profile,
		Name:        xmlLabel.Name,
		ResourceURL: resourceURL,
		URI:         uri,
	}
}

// convertXMLArtistToModel converts XML Artist to database Artist model
func (s *DiscogsXMLParserService) convertXMLArtistToModel(xmlArtist types.Artist) *models.Artist {
	resourceURL := fmt.Sprintf(DISCOG_API_URL, "artists", xmlArtist.ID)
	uri := fmt.Sprintf(DISCOG_URL, "artists", xmlArtist.ID)
	releasesURL := uri + "/releases"

	return &models.Artist{
		BaseDiscogModel: models.BaseDiscogModel{
			ID: xmlArtist.ID,
		},
		Name:        xmlArtist.Name,
		Profile:     xmlArtist.Profile,
		ResourceURL: resourceURL,
		ReleasesURL: releasesURL,
		Uri:         uri,
	}
}

// convertXMLMasterToModel converts XML Master to database Master model
func (s *DiscogsXMLParserService) convertXMLMasterToModel(xmlMaster types.Master) *models.Master {
	resourceURL := fmt.Sprintf(DISCOG_API_URL, "masters", xmlMaster.ID)
	uri := fmt.Sprintf(DISCOG_URL, "master", xmlMaster.ID)

	var mainReleaseID *int64
	var mainReleaseResourceURL *string
	if xmlMaster.MainRelease > 0 {
		id := int64(xmlMaster.MainRelease)
		mainReleaseID = &id
		url := fmt.Sprintf(DISCOG_API_URL, "releases", xmlMaster.MainRelease)
		mainReleaseResourceURL = &url
	}

	var year *int
	if xmlMaster.Year > 0 {
		year = &xmlMaster.Year
	}

	return &models.Master{
		BaseDiscogModel: models.BaseDiscogModel{
			ID: xmlMaster.ID,
		},
		Title:                  xmlMaster.Title,
		Year:                   year,
		MainReleaseID:          mainReleaseID,
		MainReleaseResourceURL: mainReleaseResourceURL,
		Uri:                    uri,
		ResourceURL:            resourceURL,
	}
}

// convertXMLReleaseToModel converts XML Release to database Release model
func (s *DiscogsXMLParserService) convertXMLReleaseToModel(
	xmlRelease types.Release,
) *models.Release {
	resourceURL := fmt.Sprintf(DISCOG_API_URL, "releases", xmlRelease.ID)
	uri := fmt.Sprintf(DISCOG_URL, "release", xmlRelease.ID)

	var year *int
	if xmlRelease.Released != "" {
		// Try to parse year from released string (could be YYYY or YYYY-MM-DD)
		if len(xmlRelease.Released) >= 4 {
			if parsedYear, err := strconv.Atoi(xmlRelease.Released[:4]); err == nil &&
				parsedYear > 0 {
				year = &parsedYear
			}
		}
	}

	var country *string
	if xmlRelease.Country != "" {
		country = &xmlRelease.Country
	}

	var notes *string
	if xmlRelease.Notes != "" {
		notes = &xmlRelease.Notes
	}

	// Determine format based on format information
	format := models.FormatVinyl // Default to vinyl
	// TODO: Parse actual format from xmlRelease.Formats when needed

	release := &models.Release{
		BaseDiscogModel: models.BaseDiscogModel{
			ID: xmlRelease.ID,
		},
		Title: xmlRelease.Title,
		// MasterID:    masterID,
		Year:        year,
		Country:     country,
		Format:      format,
		Notes:       notes,
		ResourceURL: &resourceURL,
		URI:         &uri,
	}

	if xmlRelease.MasterID > 0 {
		release.MasterID = &xmlRelease.MasterID
	}

	return release
}

// ParseXMLFiles processes Discogs XML data files
func (s *DiscogsXMLParserService) ParseXMLFiles(ctx context.Context) error {
	log := s.log.Function("ParseXMLFiles")

	now := time.Now().UTC()
	yearMonth := now.Format("2006-01")
	log.Info("Starting XML parsing process", "yearMonth", yearMonth)

	downloadDir := fmt.Sprintf("%s/%s", DiscogsDataDir, yearMonth)
	if err := ensureDirectory(downloadDir, log); err != nil {
		return log.Err("failed to create download directory", err, "directory", downloadDir)
	}

	// Process labels using the abstracted entity processor
	labelsFilePath := filepath.Join(downloadDir, "labels.xml.gz")
	labelsConfig := EntityProcessorConfig[types.Label, models.Label]{
		FilePath:       labelsFilePath,
		ElementName:    "label",
		EntityTypeName: "labels",
		ChannelSize:    5000,
		BatchSize:      5000,
		ConvertFunc:    s.convertXMLLabelToModel,
		UpsertFunc:     s.repos.Label.UpsertBatch,
	}

	if err := ProcessXMLEntities(ctx, labelsConfig, s.db, log); err != nil {
		return log.Err("failed to process labels", err)
	}

	// Process artists using the abstracted entity processor
	artistsFilePath := filepath.Join(downloadDir, "artists.xml.gz")
	artistsConfig := EntityProcessorConfig[types.Artist, models.Artist]{
		FilePath:       artistsFilePath,
		ElementName:    "artist",
		EntityTypeName: "artists",
		ChannelSize:    5000,
		BatchSize:      2500,
		ConvertFunc:    s.convertXMLArtistToModel,
		UpsertFunc:     s.repos.Artist.UpsertBatch,
	}

	if err := ProcessXMLEntities(ctx, artistsConfig, s.db, log); err != nil {
		return log.Err("failed to process artists", err)
	}

	// Process masters using the abstracted entity processor
	mastersFilePath := filepath.Join(downloadDir, "masters.xml.gz")
	mastersConfig := EntityProcessorConfig[types.Master, models.Master]{
		FilePath:       mastersFilePath,
		ElementName:    "master",
		EntityTypeName: "masters",
		ChannelSize:    5000,
		BatchSize:      5000,
		ConvertFunc:    s.convertXMLMasterToModel,
		UpsertFunc:     s.repos.Master.UpsertBatch,
	}

	if err := ProcessXMLEntities(ctx, mastersConfig, s.db, log); err != nil {
		return log.Err("failed to process masters", err)
	}

	// Process releases using the abstracted entity processor
	releasesFilePath := filepath.Join(downloadDir, "releases.xml.gz")
	releasesConfig := EntityProcessorConfig[types.Release, models.Release]{
		FilePath:       releasesFilePath,
		ElementName:    "release",
		EntityTypeName: "releases",
		ChannelSize:    5000,
		BatchSize:      2500, // Smaller batch size for releases due to more complex data
		ConvertFunc:    s.convertXMLReleaseToModel,
		UpsertFunc:     s.repos.Release.UpsertBatch,
	}

	if err := ProcessXMLEntities(ctx, releasesConfig, s.db, log); err != nil {
		return log.Err("failed to process releases", err)
	}

	// Example 3: Parse releases using generics
	/*
		releasesChan := make(chan types.Release, 100)
		go func() {
			defer close(releasesChan)
			err := ParseXMLGeneric(ctx, "/path/to/discogs_releases.xml.gz", "release", releasesChan, 1000, log)
			if err != nil {
				log.Error("Failed to parse releases", "error", err)
			}
		}()

		for release := range releasesChan {
			log.Info("Parsed release", "id", release.ID, "title", release.Title)
		}
	*/

	// Example 4: Parse masters using generics
	/*
		mastersChan := make(chan types.Master, 100)
		go func() {
			defer close(mastersChan)
			err := ParseXMLGeneric(ctx, "/path/to/discogs_masters.xml.gz", "master", mastersChan, 1000, log)
			if err != nil {
				log.Error("Failed to parse masters", "error", err)
			}
		}()

		for master := range mastersChan {
			log.Info("Parsed master", "id", master.ID, "title", master.Title)
		}
	*/

	log.Info("XML parsing completed successfully", "yearMonth", yearMonth)
	return nil
}

// isGzipFile checks if the file path indicates a gzip compressed file
func isGzipFile(filePath string) bool {
	return len(filePath) > 3 && filePath[len(filePath)-3:] == ".gz"
}

// ParseXMLGeneric is a generic function that can parse any XML entity type
func ParseXMLGeneric[T any](
	ctx context.Context,
	filePath string,
	elementName string,
	resultChan chan<- T,
	maxEntities int,
	log logger.Logger,
) error {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	// Handle gzip files
	var reader io.Reader = file
	if isGzipFile(filePath) {
		gzipReader, err := gzip.NewReader(file)
		if err != nil {
			return fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gzipReader.Close()
		reader = gzipReader
	}

	// Parse the XML stream
	decoder := xml.NewDecoder(reader)
	entityCount := 0
	errorCount := 0

	log.Info("Starting generic XML parsing", "elementName", elementName, "maxEntities", maxEntities)

	for {
		select {
		case <-ctx.Done():
			log.Info("Parsing cancelled", "entitiesParsed", entityCount, "errors", errorCount)
			return ctx.Err()
		default:
		}

		token, err := decoder.Token()
		if err == io.EOF {
			log.Info("Reached end of XML file", "entitiesParsed", entityCount, "errors", errorCount)
			break
		}
		if err != nil {
			log.Error("XML token error", "error", err)
			errorCount++
			continue
		}

		// Check if this is our target element
		if startElement, ok := token.(xml.StartElement); ok &&
			startElement.Name.Local == elementName {
			var entity T
			if err := decoder.DecodeElement(&entity, &startElement); err != nil {
				log.Error("Failed to decode entity", "elementName", elementName, "error", err)
				errorCount++
				continue
			}

			// Send the parsed entity to the channel (blocking send with context check)
			select {
			case resultChan <- entity:
				entityCount++
				if entityCount%100_000 == 0 {
					log.Info(
						"Parsing progress",
						"entitiesParsed",
						entityCount,
						"errors",
						errorCount,
					)
				}
			case <-ctx.Done():
				log.Info("Context cancelled while sending result", "entitiesParsed", entityCount)
				return ctx.Err()
			}

			// Check max entities limit (useful for testing large files)
			if maxEntities > 0 && entityCount >= maxEntities {
				log.Info(
					"Reached max entities limit",
					"entitiesParsed",
					entityCount,
					"maxEntities",
					maxEntities,
				)
				break
			}
		}
	}

	log.Info("Generic XML parsing completed", "entitiesParsed", entityCount, "errors", errorCount)
	return nil
}
