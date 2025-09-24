package services

import (
	"compress/gzip"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
	"waugzee/internal/database"
	"waugzee/internal/logger"
	. "waugzee/internal/models"
	"waugzee/internal/repositories"
	"waugzee/internal/types"
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

// func (s *DiscogsXMLParserService) HandleLabels(labelChannel chan<- types.Label) string {
//
// }

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

	// Example usage of the new generic parser
	// In production, these would be actual file paths
	log.Info("Example usage of generic XML parser")

	// Example 1: Parse artists using generics
	/*
		artistsChan := make(chan types.Artist, 100)
		go func() {
			defer close(artistsChan)
			err := ParseXMLGeneric(ctx, "/path/to/discogs_artists.xml.gz", "artist", artistsChan, 1000, log)
			if err != nil {
				log.Error("Failed to parse artists", "error", err)
			}
		}()

		for artist := range artistsChan {
			log.Info("Parsed artist", "id", artist.ID, "name", artist.Name)
		}
	*/

	// Example 2: Parse labels using generics
	// Use unbuffered channel to ensure no records are lost due to channel blocking
	labelsChan := make(chan types.Label)

	// Start the parser in a goroutine
	go func() {
		defer close(labelsChan)
		targetFile := filepath.Join(downloadDir, fmt.Sprintf("%s.xml.gz", "labels"))
		err := ParseXMLGeneric(
			ctx,
			targetFile,
			"label",
			labelsChan,
			10000, // Remove limit to process all records
			log,
		)
		if err != nil {
			log.Error("Failed to parse labels", "error", err)
		}
		log.Info("Label parsing goroutine completed")
	}()

	// Process labels with smaller batch size for more frequent database saves
	processedCount := 0

	var labels []Label
	for xmlLabel := range labelsChan {
		processedCount++
		if processedCount%1000 == 0 {
			log.Info("Processing labels", "processed", processedCount)
		}

		label := Label{
			ID:      xmlLabel.ID,
			Profile: &xmlLabel.Profile,
			Name:    xmlLabel.Name,
			// URI:        xmlLabel.URLs,
		}
		labels = append(labels, label)

		// Use smaller batch size to prevent channel blocking
		if len(labels) >= 1000 {
			log.Info(
				"Saving labels batch",
				"count",
				len(labels),
				"totalProcessed",
				processedCount,
				"firstLabelID",
				labels[0].ID,
				"lastLabelID",
				labels[len(labels)-1].ID,
			)
			if err := s.repos.Label.UpsertBatch(ctx, s.db.SQLWithContext(ctx), labels); err != nil {
				log.Error("Failed to upsert labels", "error", err)
			} else {
				log.Info("Successfully saved labels batch", "count", len(labels), "firstLabelID", labels[0].ID, "lastLabelID", labels[len(labels)-1].ID)
			}
			labels = []Label{}
		}
	}

	// Save any remaining labels in the final batch
	if len(labels) > 0 {
		log.Info(
			"Saving final labels batch",
			"count",
			len(labels),
			"totalProcessed",
			processedCount,
			"firstLabelID",
			labels[0].ID,
			"lastLabelID",
			labels[len(labels)-1].ID,
		)
		if err := s.repos.Label.UpsertBatch(ctx, s.db.SQLWithContext(ctx), labels); err != nil {
			log.Error("Failed to upsert final labels batch", "error", err)
		} else {
			log.Info("Successfully saved final labels batch", "count", len(labels), "firstLabelID", labels[0].ID, "lastLabelID", labels[len(labels)-1].ID)
		}
	}

	log.Info("Label processing completed", "totalProcessed", processedCount)

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
				if entityCount%1000 == 0 {
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
