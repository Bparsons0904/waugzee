package services

import (
	"compress/gzip"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"
	"waugzee/internal/imports"
	"waugzee/internal/logger"
	"waugzee/internal/models"
	"waugzee/internal/repositories"
)

const (
	XML_BATCH_SIZE = 2000
	PROGRESS_REPORT_INTERVAL = 50000
)

type XMLProcessingService struct {
	labelRepo                 repositories.LabelRepository
	artistRepo                repositories.ArtistRepository
	masterRepo                repositories.MasterRepository
	releaseRepo               repositories.ReleaseRepository
	discogsDataProcessingRepo repositories.DiscogsDataProcessingRepository
	transactionService        *TransactionService
	log                       logger.Logger
}

func NewXMLProcessingService(
	labelRepo repositories.LabelRepository,
	artistRepo repositories.ArtistRepository,
	masterRepo repositories.MasterRepository,
	releaseRepo repositories.ReleaseRepository,
	discogsDataProcessingRepo repositories.DiscogsDataProcessingRepository,
	transactionService *TransactionService,
) *XMLProcessingService {
	return &XMLProcessingService{
		labelRepo:                 labelRepo,
		artistRepo:                artistRepo,
		masterRepo:                masterRepo,
		releaseRepo:               releaseRepo,
		discogsDataProcessingRepo: discogsDataProcessingRepo,
		transactionService:        transactionService,
		log:                       logger.New("xmlProcessingService"),
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

func (s *XMLProcessingService) ProcessLabelsFile(ctx context.Context, filePath string, processingID string) (*ProcessingResult, error) {
	log := s.log.Function("ProcessLabelsFile")

	log.Info("Starting labels file processing", "filePath", filePath, "processingID", processingID)

	// Open and decompress the gzipped XML file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, log.Err("failed to open labels file", err, "filePath", filePath)
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

	var labelBatch []*models.Label
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

		// Look for label start elements
		if startElement, ok := token.(xml.StartElement); ok && startElement.Name.Local == "label" {
			var discogsLabel imports.Label
			if err := decoder.DecodeElement(&discogsLabel, &startElement); err != nil {
				errorMsg := fmt.Sprintf("Failed to decode label element: %v", err)
				result.Errors = append(result.Errors, errorMsg)
				result.ErroredRecords++
				log.Warn("Failed to decode label", "error", err)
				continue
			}

			// Convert Discogs label to our label model
			label := s.convertDiscogsLabel(&discogsLabel)
			if label == nil {
				result.ErroredRecords++
				continue
			}

			labelBatch = append(labelBatch, label)
			recordCount++
			result.TotalRecords++

			// Process batch when it reaches the limit
			if len(labelBatch) >= XML_BATCH_SIZE {
				if err := s.processBatch(ctx, labelBatch, result); err != nil {
					log.Err("failed to process label batch", err, "batchSize", len(labelBatch))
					return result, err
				}
				labelBatch = labelBatch[:0] // Reset batch
			}

			// Report progress every PROGRESS_REPORT_INTERVAL records
			if recordCount%PROGRESS_REPORT_INTERVAL == 0 {
				stats := &models.ProcessingStats{
					TotalRecords:    result.TotalRecords,
					LabelsProcessed: result.ProcessedRecords,
					FailedRecords:   result.ErroredRecords,
				}
				if err := s.updateProcessingStats(ctx, processingID, stats); err != nil {
					log.Warn("failed to update processing stats", "error", err, "recordCount", recordCount)
				}
				log.Info("Processing progress", "processed", recordCount, "inserted", result.InsertedRecords, "updated", result.UpdatedRecords, "errors", result.ErroredRecords)
			}
		}
	}

	// Process remaining batch
	if len(labelBatch) > 0 {
		if err := s.processBatch(ctx, labelBatch, result); err != nil {
			log.Err("failed to process final label batch", err, "batchSize", len(labelBatch))
			return result, err
		}
	}

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

func (s *XMLProcessingService) processBatch(ctx context.Context, labels []*models.Label, result *ProcessingResult) error {
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

func (s *XMLProcessingService) updateProcessingStatus(ctx context.Context, processingID string, status models.ProcessingStatus, stats *models.ProcessingStats) error {
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

func (s *XMLProcessingService) updateProcessingStats(ctx context.Context, processingID string, stats *models.ProcessingStats) error {
	return s.updateProcessingStatus(ctx, processingID, models.ProcessingStatusProcessing, stats)
}

func (s *XMLProcessingService) ProcessArtistsFile(ctx context.Context, filePath string, processingID string) (*ProcessingResult, error) {
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
			artist := s.convertDiscogsArtist(&discogsArtist)
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
					log.Warn("failed to update processing stats", "error", err, "recordCount", recordCount)
				}
				log.Info("Processing progress", "processed", recordCount, "inserted", result.InsertedRecords, "updated", result.UpdatedRecords, "errors", result.ErroredRecords)
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

func (s *XMLProcessingService) processArtistBatch(ctx context.Context, artists []*models.Artist, result *ProcessingResult) error {
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

func (s *XMLProcessingService) convertDiscogsArtist(discogsArtist *imports.Artist) *models.Artist {
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

func (s *XMLProcessingService) ProcessMastersFile(ctx context.Context, filePath string, processingID string) (*ProcessingResult, error) {
	log := s.log.Function("ProcessMastersFile")

	log.Info("Starting masters file processing", "filePath", filePath, "processingID", processingID)

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
			master := s.convertDiscogsMaster(&discogsMaster)
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
					log.Warn("failed to update processing stats", "error", err, "recordCount", recordCount)
				}
				log.Info("Processing progress", "processed", recordCount, "inserted", result.InsertedRecords, "updated", result.UpdatedRecords, "errors", result.ErroredRecords)
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

func (s *XMLProcessingService) processMasterBatch(ctx context.Context, masters []*models.Master, result *ProcessingResult) error {
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

func (s *XMLProcessingService) convertDiscogsMaster(discogsMaster *imports.Master) *models.Master {
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

func (s *XMLProcessingService) ProcessReleasesFile(ctx context.Context, filePath string, processingID string) (*ProcessingResult, error) {
	log := s.log.Function("ProcessReleasesFile")

	log.Info("Starting releases file processing", "filePath", filePath, "processingID", processingID)

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
		if startElement, ok := token.(xml.StartElement); ok && startElement.Name.Local == "release" {
			var discogsRelease imports.Release
			if err := decoder.DecodeElement(&discogsRelease, &startElement); err != nil {
				errorMsg := fmt.Sprintf("Failed to decode release element: %v", err)
				result.Errors = append(result.Errors, errorMsg)
				result.ErroredRecords++
				log.Warn("Failed to decode release", "error", err)
				continue
			}

			// Convert Discogs release to our release model
			release := s.convertDiscogsRelease(&discogsRelease)
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
					log.Warn("failed to update processing stats", "error", err, "recordCount", recordCount)
				}
				log.Info("Processing progress", "processed", recordCount, "inserted", result.InsertedRecords, "updated", result.UpdatedRecords, "errors", result.ErroredRecords)
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

func (s *XMLProcessingService) processReleaseBatch(ctx context.Context, releases []*models.Release, result *ProcessingResult) error {
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

func (s *XMLProcessingService) convertDiscogsRelease(discogsRelease *imports.Release) *models.Release {
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