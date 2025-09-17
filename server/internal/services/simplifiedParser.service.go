package services

import (
	"compress/gzip"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"waugzee/internal/imports"
)

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
		options.FileType,
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
	fileType string,
	entityChan chan<- EntityMessage,
) error {
	decoder := xml.NewDecoder(reader)

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
		}
	}

	return nil
}
