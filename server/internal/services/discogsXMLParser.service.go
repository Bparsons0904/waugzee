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
type EntityProcessorConfig[XMLType any, TModelType any] struct {
	FilePath       string
	ElementName    string
	EntityTypeName string
	ChannelSize    int
	BatchSize      int
	ConvertFunc    func(XMLType) *TModelType
	UpsertFunc     func(ctx context.Context, db *gorm.DB, entities []*TModelType) error
}

// ProcessXMLEntities is a generic function that processes XML entities with configurable conversion and batch operations
func ProcessXMLEntities[XMLType any, TModelType any](
	ctx context.Context,
	config EntityProcessorConfig[XMLType, TModelType],
	db database.DB,
	log logger.Logger,
) error {
	processingLog := log.Function("ProcessXMLEntities").With("entityType", config.EntityTypeName)

	// Create channel for streaming XML entities
	xmlChan := make(chan XMLType, config.ChannelSize)

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

// convertReleaseToArtistAssociations extracts artist associations from a release
func (s *DiscogsXMLParserService) convertReleaseToArtistAssociations(
	xmlRelease types.Release,
) *[]repositories.ReleaseArtistAssociation {
	var associations []repositories.ReleaseArtistAssociation

	for _, artist := range xmlRelease.Artists {
		if artist.ID > 0 {
			associations = append(associations, repositories.ReleaseArtistAssociation{
				ReleaseID: xmlRelease.ID,
				ArtistID:  artist.ID,
			})
		}
	}

	return &associations
}

// convertReleaseToLabelAssociations extracts label associations from a release
func (s *DiscogsXMLParserService) convertReleaseToLabelAssociations(
	xmlRelease types.Release,
) *[]repositories.ReleaseLabelAssociation {
	var associations []repositories.ReleaseLabelAssociation

	for _, label := range xmlRelease.Labels {
		if label.ID > 0 {
			associations = append(associations, repositories.ReleaseLabelAssociation{
				ReleaseID: xmlRelease.ID,
				LabelID:   label.ID,
			})
		}
	}

	return &associations
}

// convertMasterToArtistAssociations extracts artist associations from a master
func (s *DiscogsXMLParserService) convertMasterToArtistAssociations(
	xmlMaster types.Master,
) *[]repositories.MasterArtistAssociation {
	var associations []repositories.MasterArtistAssociation

	for _, artist := range xmlMaster.Artists {
		if artist.ID > 0 {
			associations = append(associations, repositories.MasterArtistAssociation{
				MasterID: xmlMaster.ID,
				ArtistID: artist.ID,
			})
		}
	}

	return &associations
}

// convertMasterToGenreAssociations extracts genre associations from a master using the genre manager
func (s *DiscogsXMLParserService) convertMasterToGenreAssociations(
	xmlMaster types.Master,
	genreManager *GenreStyleManager,
) *[]repositories.MasterGenreAssociation {
	var associations []repositories.MasterGenreAssociation

	genreIDs := genreManager.GetGenreIDsByNames(xmlMaster.Genres, xmlMaster.Styles)
	for _, genreID := range genreIDs {
		associations = append(associations, repositories.MasterGenreAssociation{
			MasterID: xmlMaster.ID,
			GenreID:  genreID,
		})
	}

	return &associations
}

// convertReleaseToGenreAssociations extracts genre associations from a release using the genre manager
func (s *DiscogsXMLParserService) convertReleaseToGenreAssociations(
	xmlRelease types.Release,
	genreManager *GenreStyleManager,
) *[]repositories.ReleaseGenreAssociation {
	var associations []repositories.ReleaseGenreAssociation

	genreIDs := genreManager.GetGenreIDsByNames(xmlRelease.Genres, xmlRelease.Styles)
	for _, genreID := range genreIDs {
		associations = append(associations, repositories.ReleaseGenreAssociation{
			ReleaseID: xmlRelease.ID,
			GenreID:   genreID,
		})
	}

	return &associations
}

// ParseXMLFiles processes Discogs XML data files
func (s *DiscogsXMLParserService) ParseXMLFiles(ctx context.Context) error {
	log := s.log.Function("ParseXMLFiles")

	now := time.Now().UTC()
	yearMonth := now.Format("2006-01")
	log.Info("Starting XML parsing process", "yearMonth", yearMonth)

	// Get or create processing record
	processing, err := s.getOrCreateProcessingRecord(ctx, yearMonth)
	if err != nil {
		return log.Err("failed to get or create processing record", err)
	}

	// Validate processing status
	if processing.Status != models.ProcessingStatusReadyForProcessing && processing.Status != models.ProcessingStatusProcessing {
		return log.Err("processing not ready", nil, "status", processing.Status, "yearMonth", yearMonth)
	}

	// Update status to processing if not already
	if processing.Status != models.ProcessingStatusProcessing {
		processing.Status = models.ProcessingStatusProcessing
		processing.StartedAt = &now
		if err := s.repos.DiscogsDataProcessing.Update(ctx, processing); err != nil {
			return log.Err("failed to update processing status", err)
		}
	}

	downloadDir := fmt.Sprintf("%s/%s", DiscogsDataDir, yearMonth)
	if err := ensureDirectory(downloadDir, log); err != nil {
		return log.Err("failed to create download directory", err, "directory", downloadDir)
	}

	// Process labels using the abstracted entity processor
	labelsFilePath := filepath.Join(downloadDir, "labels.xml.gz")
	err = s.executeProcessingStep(ctx, processing, models.StepLabelsProcessing, "Labels Processing", func() error {
		labelsConfig := EntityProcessorConfig[types.Label, models.Label]{
			FilePath:       labelsFilePath,
			ElementName:    "label",
			EntityTypeName: "labels",
			ChannelSize:    5000,
			BatchSize:      5000,
			ConvertFunc:    s.convertXMLLabelToModel,
			UpsertFunc:     s.repos.Label.UpsertBatch,
		}
		return ProcessXMLEntities(ctx, labelsConfig, s.db, log)
	})
	if err != nil {
		return err
	}

	// Process artists using the abstracted entity processor
	artistsFilePath := filepath.Join(downloadDir, "artists.xml.gz")
	err = s.executeProcessingStep(ctx, processing, models.StepArtistsProcessing, "Artists Processing", func() error {
		artistsConfig := EntityProcessorConfig[types.Artist, models.Artist]{
			FilePath:       artistsFilePath,
			ElementName:    "artist",
			EntityTypeName: "artists",
			ChannelSize:    5000,
			BatchSize:      2500,
			ConvertFunc:    s.convertXMLArtistToModel,
			UpsertFunc:     s.repos.Artist.UpsertBatch,
		}
		return ProcessXMLEntities(ctx, artistsConfig, s.db, log)
	})
	if err != nil {
		return err
	}

	// Process masters using the abstracted entity processor
	mastersFilePath := filepath.Join(downloadDir, "masters.xml.gz")
	err = s.executeProcessingStep(ctx, processing, models.StepMastersProcessing, "Masters Processing", func() error {
		mastersConfig := EntityProcessorConfig[types.Master, models.Master]{
			FilePath:       mastersFilePath,
			ElementName:    "master",
			EntityTypeName: "masters",
			ChannelSize:    5000,
			BatchSize:      5000,
			ConvertFunc:    s.convertXMLMasterToModel,
			UpsertFunc:     s.repos.Master.UpsertBatch,
		}
		return ProcessXMLEntities(ctx, mastersConfig, s.db, log)
	})
	if err != nil {
		return err
	}

	// Process releases using the abstracted entity processor
	releasesFilePath := filepath.Join(downloadDir, "releases.xml.gz")
	err = s.executeProcessingStep(ctx, processing, models.StepReleasesProcessing, "Releases Processing", func() error {
		releasesConfig := EntityProcessorConfig[types.Release, models.Release]{
			FilePath:       releasesFilePath,
			ElementName:    "release",
			EntityTypeName: "releases",
			ChannelSize:    5000,
			BatchSize:      2500, // Smaller batch size for releases due to more complex data
			ConvertFunc:    s.convertXMLReleaseToModel,
			UpsertFunc:     s.repos.Release.UpsertBatch,
		}
		return ProcessXMLEntities(ctx, releasesConfig, s.db, log)
	})
	if err != nil {
		return err
	}

	// Process genre/style data using entity-by-entity approach
	log.Info("Starting genre/style processing")

	// Initialize genre manager for masters processing
	genreManager := NewGenreStyleManager(s.repos.Genre)

	// === MASTERS GENRE PROCESSING ===
	err = s.executeProcessingStep(ctx, processing, models.StepMasterGenresCollection, "Master Genres Collection", func() error {
		return s.collectGenresFromXML(ctx, mastersFilePath, "master", genreManager, log)
	})
	if err != nil {
		return err
	}

	err = s.executeProcessingStep(ctx, processing, models.StepMasterGenresUpsert, "Master Genres Upsert", func() error {
		return genreManager.BatchUpsertMissingGenres(ctx, s.db.SQLWithContext(ctx))
	})
	if err != nil {
		return err
	}

	err = s.executeProcessingStep(ctx, processing, models.StepMasterGenreAssociations, "Master Genre Associations", func() error {
		masterGenreConfig := EntityProcessorConfig[types.Master, []repositories.MasterGenreAssociation]{
			FilePath:       mastersFilePath,
			ElementName:    "master",
			EntityTypeName: "master-genre-associations",
			ChannelSize:    5000,
			BatchSize:      5000,
			ConvertFunc: func(xmlMaster types.Master) *[]repositories.MasterGenreAssociation {
				return s.convertMasterToGenreAssociations(xmlMaster, genreManager)
			},
			UpsertFunc: s.repos.Master.UpsertMasterGenreAssociationsBatch,
		}
		return ProcessXMLEntities(ctx, masterGenreConfig, s.db, log)
	})
	if err != nil {
		return err
	}

	// === RELEASES GENRE PROCESSING ===
	err = s.executeProcessingStep(ctx, processing, models.StepReleaseGenresCollection, "Release Genres Collection", func() error {
		genreManager.Reset()
		return s.collectGenresFromXML(ctx, releasesFilePath, "release", genreManager, log)
	})
	if err != nil {
		return err
	}

	err = s.executeProcessingStep(ctx, processing, models.StepReleaseGenresUpsert, "Release Genres Upsert", func() error {
		return genreManager.BatchUpsertMissingGenres(ctx, s.db.SQLWithContext(ctx))
	})
	if err != nil {
		return err
	}

	err = s.executeProcessingStep(ctx, processing, models.StepReleaseGenreAssociations, "Release Genre Associations", func() error {
		releaseGenreConfig := EntityProcessorConfig[types.Release, []repositories.ReleaseGenreAssociation]{
			FilePath:       releasesFilePath,
			ElementName:    "release",
			EntityTypeName: "release-genre-associations",
			ChannelSize:    5000,
			BatchSize:      5000,
			ConvertFunc: func(xmlRelease types.Release) *[]repositories.ReleaseGenreAssociation {
				return s.convertReleaseToGenreAssociations(xmlRelease, genreManager)
			},
			UpsertFunc: s.repos.Release.UpsertReleaseGenreAssociationsBatch,
		}
		return ProcessXMLEntities(ctx, releaseGenreConfig, s.db, log)
	})
	if err != nil {
		return err
	}

	// === OTHER ASSOCIATIONS ===
	log.Info("Processing other associations")

	err = s.executeProcessingStep(ctx, processing, models.StepReleaseLabelAssociations, "Release Label Associations", func() error {
		releaseLabelConfig := EntityProcessorConfig[types.Release, []repositories.ReleaseLabelAssociation]{
			FilePath:       releasesFilePath,
			ElementName:    "release",
			EntityTypeName: "release-label-associations",
			ChannelSize:    5000,
			BatchSize:      5000,
			ConvertFunc:    s.convertReleaseToLabelAssociations,
			UpsertFunc:     s.repos.Release.UpsertReleaseLabelAssociationsBatch,
		}
		return ProcessXMLEntities(ctx, releaseLabelConfig, s.db, log)
	})
	if err != nil {
		return err
	}

	err = s.executeProcessingStep(ctx, processing, models.StepMasterArtistAssociations, "Master Artist Associations", func() error {
		masterArtistConfig := EntityProcessorConfig[types.Master, []repositories.MasterArtistAssociation]{
			FilePath:       mastersFilePath,
			ElementName:    "master",
			EntityTypeName: "master-artist-associations",
			ChannelSize:    5000,
			BatchSize:      5000,
			ConvertFunc:    s.convertMasterToArtistAssociations,
			UpsertFunc:     s.repos.Master.UpsertMasterArtistAssociationsBatch,
		}
		return ProcessXMLEntities(ctx, masterArtistConfig, s.db, log)
	})
	if err != nil {
		return err
	}

	err = s.executeProcessingStep(ctx, processing, models.StepReleaseArtistAssociations, "Release Artist Associations", func() error {
		releaseArtistConfig := EntityProcessorConfig[types.Release, []repositories.ReleaseArtistAssociation]{
			FilePath:       releasesFilePath,
			ElementName:    "release",
			EntityTypeName: "release-artist-associations",
			ChannelSize:    5000,
			BatchSize:      5000,
			ConvertFunc:    s.convertReleaseToArtistAssociations,
			UpsertFunc:     s.repos.Release.UpsertReleaseArtistAssociationsBatch,
		}
		return ProcessXMLEntities(ctx, releaseArtistConfig, s.db, log)
	})
	if err != nil {
		return err
	}

	// Mark processing as completed
	processing.Status = models.ProcessingStatusCompleted
	completedAt := time.Now().UTC()
	processing.ProcessingCompletedAt = &completedAt

	if err := s.repos.DiscogsDataProcessing.Update(ctx, processing); err != nil {
		return log.Err("failed to update final processing status", err)
	}

	log.Info("XML parsing completed successfully", "yearMonth", yearMonth, "allStepsCompleted", processing.AllStepsCompleted())
	return nil
}

// getOrCreateProcessingRecord gets the processing record for the given year month or creates it if not found
func (s *DiscogsXMLParserService) getOrCreateProcessingRecord(ctx context.Context, yearMonth string) (*models.DiscogsDataProcessing, error) {
	log := s.log.Function("getOrCreateProcessingRecord")

	// Try to get existing record
	processing, err := s.repos.DiscogsDataProcessing.GetByYearMonth(ctx, yearMonth)
	if err == nil {
		return processing, nil
	}

	// If not found, try to get the latest processing record
	processing, err = s.repos.DiscogsDataProcessing.GetLatestProcessing(ctx)
	if err != nil {
		return nil, log.Err("failed to get latest processing record", err)
	}

	if processing == nil {
		return nil, log.Err("no processing record found with ready_for_processing or processing status", nil)
	}

	return processing, nil
}

// executeProcessingStep executes a processing step if not already completed
func (s *DiscogsXMLParserService) executeProcessingStep(
	ctx context.Context,
	processing *models.DiscogsDataProcessing,
	step models.ProcessingStep,
	stepName string,
	stepFunc func() error,
) error {
	log := s.log.Function("executeProcessingStep").With("step", step, "stepName", stepName)

	// Check if step is already completed
	if processing.IsStepCompleted(step) {
		log.Info("Step already completed, skipping", "step", step)
		return nil
	}

	log.Info("Starting processing step", "step", step)
	startTime := time.Now()

	// Execute the step
	err := stepFunc()

	// Calculate duration
	duration := time.Since(startTime)
	durationStr := duration.String()

	// Update step status based on result
	if err != nil {
		processing.MarkStepFailed(step, err.Error())
		if updateErr := s.repos.DiscogsDataProcessing.Update(ctx, processing); updateErr != nil {
			log.Error("Failed to update step status after failure", "error", updateErr)
		}
		return log.Err("processing step failed", err, "step", step, "duration", durationStr)
	}

	// Mark step as completed
	processing.MarkStepCompleted(step, nil, &durationStr)
	if err := s.repos.DiscogsDataProcessing.Update(ctx, processing); err != nil {
		return log.Err("failed to update step completion status", err, "step", step)
	}

	log.Info("Processing step completed", "step", step, "duration", durationStr)
	return nil
}

// collectGenresFromXML performs the first pass to collect all unique genre/style names
func (s *DiscogsXMLParserService) collectGenresFromXML(
	ctx context.Context,
	filePath string,
	elementName string,
	genreManager *GenreStyleManager,
	log logger.Logger,
) error {
	log = log.Function("collectGenresFromXML").With("elementName", elementName)

	switch elementName {
	case "master":
		// Create channel for streaming Master XML entities
		masterChan := make(chan types.Master, 5000)

		// Start XML parsing in goroutine
		go func() {
			defer close(masterChan)
			err := ParseXMLGeneric(ctx, filePath, elementName, masterChan, 0, log)
			if err != nil {
				log.Error("Failed to parse XML for genre collection", "error", err)
			}
		}()

		// Collect genres from masters
		for xmlMaster := range masterChan {
			genreManager.CollectNames(xmlMaster.Genres, xmlMaster.Styles)
		}

	case "release":
		// Create channel for streaming Release XML entities
		releaseChan := make(chan types.Release, 5000)

		// Start XML parsing in goroutine
		go func() {
			defer close(releaseChan)
			err := ParseXMLGeneric(ctx, filePath, elementName, releaseChan, 0, log)
			if err != nil {
				log.Error("Failed to parse XML for genre collection", "error", err)
			}
		}()

		// Collect genres from releases
		for xmlRelease := range releaseChan {
			genreManager.CollectNames(xmlRelease.Genres, xmlRelease.Styles)
		}

	default:
		return log.Err(
			"unsupported element name for genre collection",
			nil,
			"elementName",
			elementName,
		)
	}

	stats := genreManager.GetStats()
	log.Info("Genre collection completed", "stats", stats)
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
	log = log.Function("ParseXMLGeneric")
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
