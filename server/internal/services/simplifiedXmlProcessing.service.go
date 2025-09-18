package services

import (
	"context"
	"runtime"
	"strconv"
	"sync"
	"time"
	"waugzee/internal/imports"
	"waugzee/internal/logger"
	"waugzee/internal/models"
	"waugzee/internal/repositories"
)

// SimplifiedResult holds processing counters for streaming processing
type SimplifiedResult struct {
	// Processing counters only - no entity maps to prevent memory leaks
	TotalRecords   int
	ParsedRecords  int
	ErroredRecords int
	Errors         []string
}

// Enhanced image structure with context about parent entity
type ContextualDiscogsImage struct {
	*imports.DiscogsImage
	ImageableID   string // Parent entity ID
	ImageableType string // Parent entity type (artist, master, release, label)
}

// Buffer definitions for batch processing related entities
type ImageBuffer struct {
	Channel  chan *ContextualDiscogsImage
	Capacity int
}

type GenreBuffer struct {
	Channel  chan string
	Capacity int
}

type ArtistBuffer struct {
	Channel  chan *imports.Artist
	Capacity int
}

type LabelBuffer struct {
	Channel  chan *models.Label
	Capacity int
}

type MasterBuffer struct {
	Channel  chan *models.Master
	Capacity int
}

type ReleaseBuffer struct {
	Channel  chan *models.Release
	Capacity int
}

// Association buffer structures for many-to-many relationships (Master-level only)
type MasterArtistAssociationBuffer struct {
	Channel  chan *MasterArtistAssociation
	Capacity int
}

type MasterGenreAssociationBuffer struct {
	Channel  chan *MasterGenreAssociation
	Capacity int
}

// Association data structures for bulk operations (Master-level only)
type MasterArtistAssociation struct {
	MasterDiscogsID int64
	ArtistDiscogsID int64
}

type MasterGenreAssociation struct {
	MasterDiscogsID int64
	GenreName       string
}

// ProcessingBuffers contains all the buffered channels for related entity processing
type ProcessingBuffers struct {
	Images   *ImageBuffer
	Genres   *GenreBuffer
	Artists  *ArtistBuffer
	Labels   *LabelBuffer
	Masters  *MasterBuffer
	Releases *ReleaseBuffer
	// Association buffers (Master-level only)
	MasterArtists *MasterArtistAssociationBuffer
	MasterGenres  *MasterGenreAssociationBuffer
}

type SimplifiedXMLProcessingService struct {
	discogsDataProcessingRepo repositories.DiscogsDataProcessingRepository
	labelRepo                 repositories.LabelRepository
	artistRepo                repositories.ArtistRepository
	masterRepo                repositories.MasterRepository
	releaseRepo               repositories.ReleaseRepository
	genreRepo                 repositories.GenreRepository
	imageRepo                 repositories.ImageRepository
	parserService             *DiscogsParserService
	log                       logger.Logger
	// DB association management
	dbAssociationsEnabled bool
	// Processing counters for summary logging
	processedCounts      map[string]int
	processedCountsMutex sync.RWMutex
}

type ProcessingResult struct {
	TotalRecords     int
	ProcessedRecords int
	InsertedRecords  int
	UpdatedRecords   int
	ErroredRecords   int
	Errors           []string
}

func NewSimplifiedXMLProcessingService(
	discogsDataProcessingRepo repositories.DiscogsDataProcessingRepository,
	labelRepo repositories.LabelRepository,
	artistRepo repositories.ArtistRepository,
	masterRepo repositories.MasterRepository,
	releaseRepo repositories.ReleaseRepository,
	genreRepo repositories.GenreRepository,
	imageRepo repositories.ImageRepository,
	parserService *DiscogsParserService,
) *SimplifiedXMLProcessingService {
	return &SimplifiedXMLProcessingService{
		discogsDataProcessingRepo: discogsDataProcessingRepo,
		labelRepo:                 labelRepo,
		artistRepo:                artistRepo,
		masterRepo:                masterRepo,
		releaseRepo:               releaseRepo,
		genreRepo:                 genreRepo,
		imageRepo:                 imageRepo,
		parserService:             parserService,
		log:                       logger.New("simplifiedXMLProcessingService"),
		dbAssociationsEnabled:     true, // Default enabled
		processedCounts: map[string]int{
			"labels":   0,
			"artists":  0,
			"masters":  0,
			"releases": 0,
		},
	}
}

// createProcessingBuffers initializes buffered channels for related entity processing
func (s *SimplifiedXMLProcessingService) createProcessingBuffers() *ProcessingBuffers {
	return &ProcessingBuffers{
		Images: &ImageBuffer{
			Channel:  make(chan *ContextualDiscogsImage, 10000),
			Capacity: 10000,
		},
		Genres: &GenreBuffer{
			Channel:  make(chan string, 10000),
			Capacity: 10000,
		},
		Artists: &ArtistBuffer{
			Channel:  make(chan *imports.Artist, 10000),
			Capacity: 10000,
		},
		Labels: &LabelBuffer{
			Channel:  make(chan *models.Label, 10000),
			Capacity: 10000,
		},
		Masters: &MasterBuffer{
			Channel:  make(chan *models.Master, 10000),
			Capacity: 10000,
		},
		Releases: &ReleaseBuffer{
			Channel:  make(chan *models.Release, 10000),
			Capacity: 10000,
		},
		// Association buffers (Master-level only)
		MasterArtists: &MasterArtistAssociationBuffer{
			Channel:  make(chan *MasterArtistAssociation, 10000),
			Capacity: 10000,
		},
		MasterGenres: &MasterGenreAssociationBuffer{
			Channel:  make(chan *MasterGenreAssociation, 10000),
			Capacity: 10000,
		},
	}
}

// closeProcessingBuffers safely closes all buffer channels
func (s *SimplifiedXMLProcessingService) closeProcessingBuffers(buffers *ProcessingBuffers) {
	close(buffers.Images.Channel)
	close(buffers.Genres.Channel)
	close(buffers.Artists.Channel)
	close(buffers.Labels.Channel)
	close(buffers.Masters.Channel)
	close(buffers.Releases.Channel)
	// Close association buffers (Master-level only)
	close(buffers.MasterArtists.Channel)
	close(buffers.MasterGenres.Channel)
	s.log.Info("All processing buffer channels closed")
}

// Specialized Processor Methods for Each Entity Type
// These methods extract primary entity data and send related entities to appropriate buffers

// processLabel extracts Label data only - no related entities
func (s *SimplifiedXMLProcessingService) processLabel(
	rawLabel *imports.Label,
	processingID string,
	buffers *ProcessingBuffers,
) error {
	if rawLabel == nil || rawLabel.ID <= 0 {
		return s.log.Err("invalid label data", nil, "processingID", processingID)
	}

	// Convert to model and send to label buffer
	if convertedLabel := s.parserService.convertDiscogsLabel(rawLabel); convertedLabel != nil {
		buffers.Labels.Channel <- convertedLabel
	}

	return nil
}

// processArtist extracts Artist data + Images[] and sends Images to image buffer
func (s *SimplifiedXMLProcessingService) processArtist(
	rawArtist *imports.Artist,
	processingID string,
	buffers *ProcessingBuffers,
) error {
	if rawArtist == nil || rawArtist.ID <= 0 {
		return s.log.Err("invalid artist data", nil, "processingID", processingID)
	}

	// Extract and send Images to image buffer with context
	for i := range rawArtist.Images {
		contextualImage := &ContextualDiscogsImage{
			DiscogsImage:  &rawArtist.Images[i],
			ImageableID:   strconv.FormatInt(int64(rawArtist.ID), 10),
			ImageableType: models.ImageableTypeArtist,
		}
		buffers.Images.Channel <- contextualImage
	}

	return nil
}

// processMaster extracts Master data + Images[] + Genres[] + Artists[] and sends to appropriate buffers
func (s *SimplifiedXMLProcessingService) processMaster(
	rawMaster *imports.Master,
	processingID string,
	buffers *ProcessingBuffers,
) error {
	if rawMaster == nil || rawMaster.ID <= 0 {
		return s.log.Err("invalid master data", nil, "processingID", processingID)
	}

	// Extract and send Images to image buffer with context
	for i := range rawMaster.Images {
		contextualImage := &ContextualDiscogsImage{
			DiscogsImage:  &rawMaster.Images[i],
			ImageableID:   strconv.FormatInt(int64(rawMaster.ID), 10),
			ImageableType: models.ImageableTypeMaster,
		}
		buffers.Images.Channel <- contextualImage
	}

	// Extract and send Genres to genre buffer
	for _, genre := range rawMaster.Genres {
		if genre != "" {
			buffers.Genres.Channel <- genre
		}
	}

	// Extract and send Artists to artist buffer
	for i := range rawMaster.Artists {
		artist := &rawMaster.Artists[i]
		buffers.Artists.Channel <- artist
	}

	// Extract and send Master-Genre associations
	for _, genre := range rawMaster.Genres {
		if genre != "" {
			association := &MasterGenreAssociation{
				MasterDiscogsID: int64(rawMaster.ID),
				GenreName:       genre,
			}
			buffers.MasterGenres.Channel <- association
		}
	}

	// Extract and send Master-Artist associations
	for _, artist := range rawMaster.Artists {
		if artist.ID > 0 {
			association := &MasterArtistAssociation{
				MasterDiscogsID: int64(rawMaster.ID),
				ArtistDiscogsID: int64(artist.ID),
			}
			buffers.MasterArtists.Channel <- association
		}
	}

	// Convert to model and send to master buffer
	if convertedMaster := s.parserService.convertDiscogsMaster(rawMaster); convertedMaster != nil {
		buffers.Masters.Channel <- convertedMaster
	}

	return nil
}

// processRelease extracts Release data + Artists[] + TrackList[] + Genres[] + Images[]
// and sends all related entities to appropriate buffers
func (s *SimplifiedXMLProcessingService) processRelease(
	rawRelease *imports.Release,
	processingID string,
	buffers *ProcessingBuffers,
) error {
	if rawRelease == nil || rawRelease.ID <= 0 {
		return s.log.Err("invalid release data", nil, "processingID", processingID)
	}

	// Extract and send Artists to artist buffer
	for i := range rawRelease.Artists {
		artist := &rawRelease.Artists[i]
		buffers.Artists.Channel <- artist
	}

	// Note: Track data now stored as JSONB in Release - handled by parser convertDiscogsRelease()

	// Extract and send Genres to genre buffer
	for _, genre := range rawRelease.Genres {
		if genre != "" {
			buffers.Genres.Channel <- genre
		}
	}

	// Extract and send Images to image buffer with context
	for i := range rawRelease.Images {
		contextualImage := &ContextualDiscogsImage{
			DiscogsImage:  &rawRelease.Images[i],
			ImageableID:   strconv.FormatInt(int64(rawRelease.ID), 10),
			ImageableType: models.ImageableTypeRelease,
		}
		buffers.Images.Channel <- contextualImage
	}

	// Note: Release-level associations removed - use Master-level relationships instead

	// Convert to model and send to release buffer
	if convertedRelease := s.parserService.convertDiscogsRelease(rawRelease); convertedRelease != nil {
		buffers.Releases.Channel <- convertedRelease
	}

	return nil
}

// Batch processing methods for consuming from buffers and executing database operations

// imageKey represents a composite key for image deduplication
type imageKey struct {
	imageableID   int64
	imageableType string
	url           string
}

// processImageBuffer consumes images from the buffer, deduplicates them by composite key (imageableID + imageableType + URL), and executes batch upserts when 5000 unique items are collected
func (s *SimplifiedXMLProcessingService) processImageBuffer(
	ctx context.Context,
	buffers *ProcessingBuffers,
	wg *sync.WaitGroup,
	processingID string,
) {
	defer wg.Done()
	log := s.log.Function("processImageBuffer")
	dedupeMap := make(map[imageKey]*models.Image)
	totalProcessed := 0

	for {
		select {
		case <-ctx.Done():
			// Context cancelled, process remaining batch if any
			if len(dedupeMap) > 0 {
				s.processPendingImageBatchFromMap(ctx, dedupeMap, processingID)
			}
			return

		case contextualImage, ok := <-buffers.Images.Channel:
			if !ok {
				// Channel closed, process remaining batch
				if len(dedupeMap) > 0 {
					s.processPendingImageBatchFromMap(ctx, dedupeMap, processingID)
					totalProcessed += len(dedupeMap)
				}
				log.Info(
					"Image buffer processing completed",
					"totalProcessed",
					totalProcessed,
					"processingID",
					processingID,
				)
				return
			}

			// Convert ContextualDiscogsImage to Image model
			if modelImage := s.convertContextualDiscogsImageToModel(contextualImage); modelImage != nil {
				// Deduplicate using composite key
				key := imageKey{
					imageableID:   modelImage.ImageableID,
					imageableType: modelImage.ImageableType,
					url:           modelImage.URL,
				}
				dedupeMap[key] = modelImage

				// Process batch when we reach 5000 unique items
				if len(dedupeMap) >= 5000 {
					s.processPendingImageBatchFromMap(ctx, dedupeMap, processingID)
					totalProcessed += len(dedupeMap)
					dedupeMap = make(map[imageKey]*models.Image) // Reset map
				}
			}
		}
	}
}

// processGenreBuffer consumes genres from the buffer, deduplicates, and executes batch upserts
func (s *SimplifiedXMLProcessingService) processGenreBuffer(
	ctx context.Context,
	buffers *ProcessingBuffers,
	wg *sync.WaitGroup,
	processingID string,
) {
	defer wg.Done()
	log := s.log.Function("processGenreBuffer")

	genreSet := make(map[string]*models.Genre)
	totalReceived := 0

	for {
		select {
		case <-ctx.Done():
			// Context cancelled, process remaining genres if any
			if len(genreSet) > 0 {
				s.processPendingGenreBatch(ctx, genreSet, processingID)
			}
			return

		case genreName, ok := <-buffers.Genres.Channel:
			if !ok {
				// Channel closed, process remaining genres
				if len(genreSet) > 0 {
					s.processPendingGenreBatch(ctx, genreSet, processingID)
				}
				log.Info(
					"Genre buffer processing completed",
					"totalReceived",
					totalReceived,
					"uniqueGenres",
					len(genreSet),
					"processingID",
					processingID,
				)
				return
			}

			totalReceived++
			if genreName != "" && genreSet[genreName] == nil {
				genreSet[genreName] = &models.Genre{Name: genreName}

				// Process batch when it reaches 5000 unique genres
				if len(genreSet) >= 5000 {
					s.processPendingGenreBatch(ctx, genreSet, processingID)
					genreSet = make(map[string]*models.Genre) // Reset map
				}
			}
		}
	}
}

// processArtistBuffer consumes artists from the buffer, deduplicates them by DiscogsID, and executes batch upserts when 5000 unique items are collected
func (s *SimplifiedXMLProcessingService) processArtistBuffer(
	ctx context.Context,
	buffers *ProcessingBuffers,
	wg *sync.WaitGroup,
	processingID string,
) {
	defer wg.Done()
	log := s.log.Function("processArtistBuffer")

	// Use map for deduplication using DiscogsID as key
	dedupeMap := make(map[int64]*models.Artist)
	totalProcessed := 0

	for {
		select {
		case <-ctx.Done():
			// Context cancelled, process remaining batch if any
			if len(dedupeMap) > 0 {
				s.processPendingArtistBatchFromMap(ctx, dedupeMap, processingID)
			}
			return

		case discogsArtist, ok := <-buffers.Artists.Channel:
			if !ok {
				// Channel closed, process remaining batch
				if len(dedupeMap) > 0 {
					s.processPendingArtistBatchFromMap(ctx, dedupeMap, processingID)
					totalProcessed += len(dedupeMap)
				}
				log.Info(
					"Artist buffer processing completed",
					"totalProcessed",
					totalProcessed,
					"processingID",
					processingID,
				)
				return
			}

			// Convert Discogs Artist to Artist model
			if modelArtist := s.convertDiscogsArtistToModel(discogsArtist); modelArtist != nil {
				// Deduplicate using DiscogsID as key
				dedupeMap[modelArtist.DiscogsID] = modelArtist

				// Process batch when we reach 5000 unique items
				if len(dedupeMap) >= 5000 {
					s.processPendingArtistBatchFromMap(ctx, dedupeMap, processingID)
					totalProcessed += len(dedupeMap)
					dedupeMap = make(map[int64]*models.Artist) // Reset map
				}
			}
		}
	}
}

func (s *SimplifiedXMLProcessingService) processPendingImageBatchFromMap(
	ctx context.Context,
	dedupeMap map[imageKey]*models.Image,
	processingID string,
) {
	log := s.log.Function("processPendingImageBatchFromMap")
	if len(dedupeMap) == 0 {
		return
	}

	// Convert map to slice
	batch := make([]*models.Image, 0, len(dedupeMap))
	for _, image := range dedupeMap {
		batch = append(batch, image)
	}

	// Service has already deduplicated - pass batch directly to repository
	inserted, updated, err := s.imageRepo.UpsertBatch(ctx, batch)
	if err != nil {
		_ = log.Error(
			"Failed to upsert image batch",
			"error",
			err,
			"batchSize",
			len(batch),
			"processingID",
			processingID,
		)
		return
	}
	// Reduced logging: only log significant batches or errors to improve performance
	if len(batch) >= 10000 {
		log.Info(
			"Processed large image batch",
			"batchSize",
			len(batch),
			"inserted",
			inserted,
			"updated",
			updated,
			"processingID",
			processingID,
		)
	}
}

func (s *SimplifiedXMLProcessingService) processPendingGenreBatch(
	ctx context.Context,
	genreSet map[string]*models.Genre,
	processingID string,
) {
	log := s.log.Function("processPendingGenreBatch")
	if len(genreSet) == 0 {
		return
	}

	// Convert map to slice
	batch := make([]*models.Genre, 0, len(genreSet))
	for _, genre := range genreSet {
		batch = append(batch, genre)
	}

	inserted, updated, err := s.genreRepo.UpsertBatch(ctx, batch)
	if err != nil {
		log.Error(
			"Failed to upsert genre batch",
			"error",
			err,
			"batchSize",
			len(batch),
			"processingID",
			processingID,
		)
		return
	}
	// Reduced logging: only log significant batches or errors to improve performance
	if len(batch) >= 10000 {
		log.Info(
			"Processed large genre batch",
			"batchSize",
			len(batch),
			"inserted",
			inserted,
			"updated",
			updated,
			"processingID",
			processingID,
		)
	}
}

func (s *SimplifiedXMLProcessingService) processPendingArtistBatchFromMap(
	ctx context.Context,
	dedupeMap map[int64]*models.Artist,
	processingID string,
) {
	log := s.log.Function("processPendingArtistBatchFromMap")
	if len(dedupeMap) == 0 {
		return
	}

	// Convert map to slice
	batch := make([]*models.Artist, 0, len(dedupeMap))
	for _, artist := range dedupeMap {
		batch = append(batch, artist)
	}

	// Service has already deduplicated - pass batch directly to repository
	inserted, updated, err := s.artistRepo.UpsertBatch(ctx, batch)
	if err != nil {
		log.Error(
			"Failed to upsert artist batch",
			"error",
			err,
			"batchSize",
			len(batch),
			"processingID",
			processingID,
		)
		return
	}
	// Reduced logging: only log significant batches or errors to improve performance
	if len(batch) >= 10000 {
		log.Info(
			"Processed large artist batch",
			"batchSize",
			len(batch),
			"inserted",
			inserted,
			"updated",
			updated,
			"processingID",
			processingID,
		)
	}
}

func (s *SimplifiedXMLProcessingService) processPendingLabelBatchFromMap(
	ctx context.Context,
	dedupeMap map[int64]*models.Label,
	processingID string,
) {
	log := s.log.Function("processPendingLabelBatchFromMap")
	if len(dedupeMap) == 0 {
		return
	}

	// Convert map to slice
	batch := make([]*models.Label, 0, len(dedupeMap))
	for _, label := range dedupeMap {
		batch = append(batch, label)
	}

	// Service has already deduplicated - pass batch directly to repository
	inserted, updated, err := s.labelRepo.UpsertBatch(ctx, batch)
	if err != nil {
		log.Error(
			"Failed to upsert label batch",
			"error",
			err,
			"batchSize",
			len(batch),
			"processingID",
			processingID,
		)
		return
	}
	// Reduced logging: only log significant batches or errors to improve performance
	if len(batch) >= 10000 {
		log.Info(
			"Processed large label batch",
			"batchSize",
			len(batch),
			"inserted",
			inserted,
			"updated",
			updated,
			"processingID",
			processingID,
		)
	}
}

func (s *SimplifiedXMLProcessingService) processPendingMasterBatchFromMap(
	ctx context.Context,
	dedupeMap map[int64]*models.Master,
	processingID string,
) {
	log := s.log.Function("processPendingMasterBatchFromMap")
	if len(dedupeMap) == 0 {
		return
	}

	// Convert map to slice
	batch := make([]*models.Master, 0, len(dedupeMap))
	for _, master := range dedupeMap {
		batch = append(batch, master)
	}

	// Service has already deduplicated - pass batch directly to repository
	inserted, updated, err := s.masterRepo.UpsertBatch(ctx, batch)
	if err != nil {
		log.Error(
			"Failed to upsert master batch",
			"error",
			err,
			"batchSize",
			len(batch),
			"processingID",
			processingID,
		)
		return
	}
	// Reduced logging: only log significant batches or errors to improve performance
	if len(batch) >= 10000 {
		log.Info(
			"Processed large master batch",
			"batchSize",
			len(batch),
			"inserted",
			inserted,
			"updated",
			updated,
			"processingID",
			processingID,
		)
	}
}

func (s *SimplifiedXMLProcessingService) processPendingReleaseBatchFromMap(
	ctx context.Context,
	dedupeMap map[int64]*models.Release,
	processingID string,
) {
	log := s.log.Function("processPendingReleaseBatchFromMap")
	if len(dedupeMap) == 0 {
		return
	}

	// Convert map to slice
	batch := make([]*models.Release, 0, len(dedupeMap))
	for _, release := range dedupeMap {
		batch = append(batch, release)
	}

	// Service has already deduplicated - pass batch directly to repository
	inserted, updated, err := s.releaseRepo.UpsertBatch(ctx, batch)
	if err != nil {
		log.Error(
			"Failed to upsert release batch",
			"error",
			err,
			"batchSize",
			len(batch),
			"processingID",
			processingID,
		)
		return
	}
	// Reduced logging: only log significant batches or errors to improve performance
	if len(batch) >= 10000 {
		log.Info(
			"Processed large release batch",
			"batchSize",
			len(batch),
			"inserted",
			inserted,
			"updated",
			updated,
			"processingID",
			processingID,
		)
	}
}

// processLabelBuffer consumes labels from the buffer, deduplicates them by DiscogsID, and executes batch upserts when 5000 unique items are collected
func (s *SimplifiedXMLProcessingService) processLabelBuffer(
	ctx context.Context,
	buffers *ProcessingBuffers,
	wg *sync.WaitGroup,
	processingID string,
) {
	defer wg.Done()
	log := s.log.Function("processLabelBuffer")

	// Use map for deduplication using DiscogsID as key
	dedupeMap := make(map[int64]*models.Label)
	totalProcessed := 0

	for {
		select {
		case <-ctx.Done():
			// Context cancelled, process remaining batch if any
			if len(dedupeMap) > 0 {
				s.processPendingLabelBatchFromMap(ctx, dedupeMap, processingID)
			}
			return

		case modelLabel, ok := <-buffers.Labels.Channel:
			if !ok {
				// Channel closed, process remaining batch
				if len(dedupeMap) > 0 {
					s.processPendingLabelBatchFromMap(ctx, dedupeMap, processingID)
					totalProcessed += len(dedupeMap)
				}
				log.Info(
					"Label buffer processing completed",
					"totalProcessed",
					totalProcessed,
					"processingID",
					processingID,
				)
				return
			}

			if modelLabel != nil {
				// Deduplicate using DiscogsID as key
				dedupeMap[modelLabel.DiscogsID] = modelLabel

				// Process batch when we reach 5000 unique items
				if len(dedupeMap) >= 5000 {
					s.processPendingLabelBatchFromMap(ctx, dedupeMap, processingID)
					totalProcessed += len(dedupeMap)
					dedupeMap = make(map[int64]*models.Label) // Reset map
				}
			}
		}
	}
}

// processMasterBuffer consumes masters from the buffer, deduplicates them by DiscogsID, and executes batch upserts when 5000 unique items are collected
func (s *SimplifiedXMLProcessingService) processMasterBuffer(
	ctx context.Context,
	buffers *ProcessingBuffers,
	wg *sync.WaitGroup,
	processingID string,
) {
	defer wg.Done()
	log := s.log.Function("processMasterBuffer")

	// Use map for deduplication using DiscogsID as key
	dedupeMap := make(map[int64]*models.Master)
	totalProcessed := 0

	for {
		select {
		case <-ctx.Done():
			// Context cancelled, process remaining batch if any
			if len(dedupeMap) > 0 {
				s.processPendingMasterBatchFromMap(ctx, dedupeMap, processingID)
			}
			return

		case modelMaster, ok := <-buffers.Masters.Channel:
			if !ok {
				// Channel closed, process remaining batch
				if len(dedupeMap) > 0 {
					s.processPendingMasterBatchFromMap(ctx, dedupeMap, processingID)
					totalProcessed += len(dedupeMap)
				}
				log.Info(
					"Master buffer processing completed",
					"totalProcessed",
					totalProcessed,
					"processingID",
					processingID,
				)
				return
			}

			if modelMaster != nil {
				// Deduplicate using DiscogsID as key
				dedupeMap[modelMaster.DiscogsID] = modelMaster

				// Process batch when we reach 5000 unique items
				if len(dedupeMap) >= 5000 {
					s.processPendingMasterBatchFromMap(ctx, dedupeMap, processingID)
					totalProcessed += len(dedupeMap)
					dedupeMap = make(map[int64]*models.Master) // Reset map
				}
			}
		}
	}
}

// processReleaseBuffer consumes releases from the buffer, deduplicates them by DiscogsID, and executes batch upserts when 2000 unique items are collected
func (s *SimplifiedXMLProcessingService) processReleaseBuffer(
	ctx context.Context,
	buffers *ProcessingBuffers,
	wg *sync.WaitGroup,
	processingID string,
) {
	defer wg.Done()
	log := s.log.Function("processReleaseBuffer")

	// Use map for deduplication using DiscogsID as key
	dedupeMap := make(map[int64]*models.Release)
	totalProcessed := 0

	for {
		select {
		case <-ctx.Done():
			// Context cancelled, process remaining batch if any
			if len(dedupeMap) > 0 {
				s.processPendingReleaseBatchFromMap(ctx, dedupeMap, processingID)
			}
			return

		case modelRelease, ok := <-buffers.Releases.Channel:
			if !ok {
				// Channel closed, process remaining batch
				if len(dedupeMap) > 0 {
					s.processPendingReleaseBatchFromMap(ctx, dedupeMap, processingID)
					totalProcessed += len(dedupeMap)
				}
				log.Info(
					"Release buffer processing completed",
					"totalProcessed",
					totalProcessed,
					"processingID",
					processingID,
				)
				return
			}

			if modelRelease != nil {
				// Deduplicate using DiscogsID as key
				dedupeMap[modelRelease.DiscogsID] = modelRelease

				// Process batch when we reach 2000 unique items (smaller for releases due to JSONB data)
				if len(dedupeMap) >= 2000 {
					s.processPendingReleaseBatchFromMap(ctx, dedupeMap, processingID)
					totalProcessed += len(dedupeMap)
					dedupeMap = make(map[int64]*models.Release) // Reset map
				}
			}
		}
	}
}

// ProcessFileToMap parses the entire file using channel-based architecture to extract ALL XML data
// This approach captures complete entity data instead of minimal converted models
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

	// Reset processing counters for this session
	s.processedCountsMutex.Lock()
	s.processedCounts = map[string]int{
		"labels":   0,
		"artists":  0,
		"masters":  0,
		"releases": 0,
	}
	s.processedCountsMutex.Unlock()

	log.Info("Starting channel-based file processing with full data extraction",
		"filePath", filePath,
		"fileType", fileType,
		"startMemoryMB", startMemStats.Alloc/1024/1024,
		"startHeapMB", startMemStats.HeapAlloc/1024/1024)

	// Initialize result with counters only - no entity maps
	result := &SimplifiedResult{
		Errors: make([]string, 0),
	}

	// Create buffered channels for entity processing
	entityChan := make(chan EntityMessage, 10000)
	completionChan := make(chan CompletionMessage, 1)

	// Create processing buffers for related entities
	buffers := s.createProcessingBuffers()

	// Start buffer processing goroutines
	var wg sync.WaitGroup
	processingID := "simplified_processing" // Generate or pass actual processing ID

	wg.Add(8) // Add for each buffer processor (6 original + 2 master association processors)
	go s.processImageBuffer(ctx, buffers, &wg, processingID)
	go s.processGenreBuffer(ctx, buffers, &wg, processingID)
	go s.processArtistBuffer(ctx, buffers, &wg, processingID)
	go s.processLabelBuffer(ctx, buffers, &wg, processingID)
	go s.processMasterBuffer(ctx, buffers, &wg, processingID)
	go s.processReleaseBuffer(ctx, buffers, &wg, processingID)
	// Association buffer processors (Master-level only)
	go s.processMasterArtistAssociationBuffer(ctx, buffers, &wg, processingID)
	go s.processMasterGenreAssociationBuffer(ctx, buffers, &wg, processingID)

	// Start parser in goroutine
	parseOptions := ParseOptions{
		FilePath: filePath,
		FileType: fileType,
	}

	go func() {
		defer close(entityChan)
		if err := s.parserService.ParseFileToChannel(ctx, parseOptions, entityChan, completionChan); err != nil {
			log.Error("Channel parsing failed", "error", err, "filePath", filePath)
		}
	}()

	// Process entities from channel
	channelProcessingDone := false
	var processingStats struct {
		totalProcessed int
		totalErrors    int
	}

	for !channelProcessingDone {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()

		case entity, ok := <-entityChan:
			if !ok {
				// Channel closed, check for completion message
				continue
			}

			processingStats.totalProcessed++

			// Process entities through streaming channels - no memory accumulation
			switch entity.Type {
			case "label":
				if rawLabel, ok := entity.RawEntity.(*imports.Label); ok && rawLabel.ID > 0 {
					// Use specialized processor to extract related entities
					if err := s.processLabel(rawLabel, entity.ProcessingID, buffers); err != nil {
						log.Warn("Label processing failed", "error", err, "labelID", rawLabel.ID)
						processingStats.totalErrors++
					} else {
						result.ParsedRecords++
					}
				}

			case "artist":
				if rawArtist, ok := entity.RawEntity.(*imports.Artist); ok && rawArtist.ID > 0 {
					// Use specialized processor to extract related entities (Images)
					if err := s.processArtist(rawArtist, entity.ProcessingID, buffers); err != nil {
						log.Warn("Artist processing failed", "error", err, "artistID", rawArtist.ID)
						processingStats.totalErrors++
					} else {
						result.ParsedRecords++
					}
				}

			case "master":
				if rawMaster, ok := entity.RawEntity.(*imports.Master); ok && rawMaster.ID > 0 {
					// Use specialized processor to extract related entities (Images, Genres)
					if err := s.processMaster(rawMaster, entity.ProcessingID, buffers); err != nil {
						log.Warn("Master processing failed", "error", err, "masterID", rawMaster.ID)
						processingStats.totalErrors++
					} else {
						result.ParsedRecords++
					}
				}

			case "release":
				if rawRelease, ok := entity.RawEntity.(*imports.Release); ok && rawRelease.ID > 0 {
					// Use specialized processor to extract related entities (Artists, Tracks, Genres, Images)
					if err := s.processRelease(rawRelease, entity.ProcessingID, buffers); err != nil {
						log.Warn(
							"Release processing failed",
							"error",
							err,
							"releaseID",
							rawRelease.ID,
						)
						processingStats.totalErrors++
					} else {
						result.ParsedRecords++
					}
				}
			}

		case completion := <-completionChan:
			log.Info("File processing completion signal received",
				"fileType", completion.FileType,
				"completed", completion.Completed)
			channelProcessingDone = true
		}
	}

	// Set total records based on entities processed
	result.TotalRecords = processingStats.totalProcessed
	result.ErroredRecords = processingStats.totalErrors

	// Log buffer usage statistics before closing
	log.Info("Processing buffer usage statistics",
		"imagesInBuffer", len(buffers.Images.Channel),
		"genresInBuffer", len(buffers.Genres.Channel),
		"artistsInBuffer", len(buffers.Artists.Channel),
		"labelsInBuffer", len(buffers.Labels.Channel),
		"mastersInBuffer", len(buffers.Masters.Channel),
		"releasesInBuffer", len(buffers.Releases.Channel),
		"masterArtistsInBuffer", len(buffers.MasterArtists.Channel),
		"masterGenresInBuffer", len(buffers.MasterGenres.Channel),
		"imageBufferCapacity", buffers.Images.Capacity,
		"genreBufferCapacity", buffers.Genres.Capacity,
		"artistBufferCapacity", buffers.Artists.Capacity,
		"labelBufferCapacity", buffers.Labels.Capacity,
		"masterBufferCapacity", buffers.Masters.Capacity,
		"releaseBufferCapacity", buffers.Releases.Capacity,
		"masterArtistBufferCapacity", buffers.MasterArtists.Capacity,
		"masterGenreBufferCapacity", buffers.MasterGenres.Capacity)

	// Close buffers to signal completion and wait for all buffer processors to finish
	s.closeProcessingBuffers(buffers)
	log.Info("Waiting for buffer processors to complete")
	wg.Wait()
	log.Info("All buffer processors completed")

	// Log completion with final memory allocation
	var endMemStats runtime.MemStats
	runtime.ReadMemStats(&endMemStats)
	elapsed := time.Since(startTime)

	// Log final processing counts for all entity types
	s.processedCountsMutex.RLock()
	labelsProcessed := s.processedCounts["labels"]
	artistsProcessed := s.processedCounts["artists"]
	mastersProcessed := s.processedCounts["masters"]
	releasesProcessed := s.processedCounts["releases"]
	s.processedCountsMutex.RUnlock()

	s.log.Info("Final processing summary",
		"labelsProcessed", labelsProcessed,
		"artistsProcessed", artistsProcessed,
		"mastersProcessed", mastersProcessed,
		"releasesProcessed", releasesProcessed)

	log.Info("Completed streaming file processing - no memory accumulation",
		"fileType", fileType,
		"totalRecords", result.TotalRecords,
		"parsedRecords", result.ParsedRecords,
		"erroredRecords", result.ErroredRecords,
		"streamingMode", "enabled",
		"memoryMapsUsed", false,
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
	return s.ProcessFile(ctx, filePath, processingID, "labels")
}

func (s *SimplifiedXMLProcessingService) ProcessArtistsFile(
	ctx context.Context,
	filePath string,
	processingID string,
) (*ProcessingResult, error) {
	return s.ProcessFile(ctx, filePath, processingID, "artists")
}

func (s *SimplifiedXMLProcessingService) ProcessMastersFile(
	ctx context.Context,
	filePath string,
	processingID string,
) (*ProcessingResult, error) {
	return s.ProcessFile(ctx, filePath, processingID, "masters")
}

func (s *SimplifiedXMLProcessingService) ProcessReleasesFile(
	ctx context.Context,
	filePath string,
	processingID string,
) (*ProcessingResult, error) {
	return s.ProcessFile(ctx, filePath, processingID, "releases")
}

// ProcessingConfig defines configuration for each file type
type ProcessingConfig struct {
	GetRawMapSize       func(*SimplifiedResult) int
	GetConvertedMapSize func(*SimplifiedResult) int
	RawMapName          string
	ConvertedMapName    string
}

// Entity configurations for streaming processing
var processingConfigs = map[string]ProcessingConfig{
	"labels": {
		GetRawMapSize:       func(r *SimplifiedResult) int { return 0 }, // No maps used
		GetConvertedMapSize: func(r *SimplifiedResult) int { return 0 }, // No maps used
		RawMapName:          "streamingMode",
		ConvertedMapName:    "streamingMode",
	},
	"artists": {
		GetRawMapSize:       func(r *SimplifiedResult) int { return 0 }, // No maps used
		GetConvertedMapSize: func(r *SimplifiedResult) int { return 0 }, // No maps used
		RawMapName:          "streamingMode",
		ConvertedMapName:    "streamingMode",
	},
	"masters": {
		GetRawMapSize:       func(r *SimplifiedResult) int { return 0 }, // No maps used
		GetConvertedMapSize: func(r *SimplifiedResult) int { return 0 }, // No maps used
		RawMapName:          "streamingMode",
		ConvertedMapName:    "streamingMode",
	},
	"releases": {
		GetRawMapSize:       func(r *SimplifiedResult) int { return 0 }, // No maps used
		GetConvertedMapSize: func(r *SimplifiedResult) int { return 0 }, // No maps used
		RawMapName:          "streamingMode",
		ConvertedMapName:    "streamingMode",
	},
}

// ProcessFile is the consolidated generic method that handles all file types
func (s *SimplifiedXMLProcessingService) ProcessFile(
	ctx context.Context,
	filePath string,
	processingID string,
	fileType string,
) (*ProcessingResult, error) {
	log := s.log.Function("ProcessFile")

	log.Info("Starting file processing with simplified approach",
		"filePath", filePath,
		"processingID", processingID,
		"fileType", fileType)

	// Update processing status to "processing"
	if err := s.updateProcessingStatus(ctx, processingID, models.ProcessingStatusProcessing, nil); err != nil {
		log.Warn("failed to update processing status", "error", err, "processingID", processingID)
	}

	// Use simplified parsing approach
	simplifiedResult, err := s.ProcessFileToMap(ctx, filePath, fileType)
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
	result := s.convertToProcessingResult(simplifiedResult)

	// Create processing stats for final status update
	finalStats := s.createProcessingStats(
		fileType,
		result.TotalRecords,
		result.ProcessedRecords,
		result.ErroredRecords,
	)

	// Update final processing status to completed
	if err := s.updateProcessingStatus(ctx, processingID, models.ProcessingStatusCompleted, finalStats); err != nil {
		log.Warn("failed to update final processing status", "error", err)
	}

	// Log completion with streaming status
	log.Info("File processing completed with streaming approach",
		"fileType", fileType,
		"total", result.TotalRecords,
		"processed", result.ProcessedRecords,
		"errors", result.ErroredRecords,
		"processingMode", "streaming",
		"memoryMapsUsed", false)

	// Early return - no database operations performed
	return result, nil
}

// convertToProcessingResult converts SimplifiedResult to ProcessingResult for compatibility
func (s *SimplifiedXMLProcessingService) convertToProcessingResult(
	simplifiedResult *SimplifiedResult,
) *ProcessingResult {
	return &ProcessingResult{
		TotalRecords:     simplifiedResult.TotalRecords,
		ProcessedRecords: simplifiedResult.ParsedRecords,
		InsertedRecords:  0, // No database operations performed
		UpdatedRecords:   0, // No database operations performed
		ErroredRecords:   simplifiedResult.ErroredRecords,
		Errors:           simplifiedResult.Errors,
	}
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

// Conversion helper methods for converting Discogs import types to model types

func (s *SimplifiedXMLProcessingService) convertContextualDiscogsImageToModel(
	contextualImage *ContextualDiscogsImage,
) *models.Image {
	if contextualImage == nil || contextualImage.DiscogsImage == nil {
		return nil
	}

	discogsImage := contextualImage.DiscogsImage

	// Validate required fields before creating the image
	if discogsImage.URI == "" || contextualImage.ImageableID == "" ||
		contextualImage.ImageableType == "" {
		return nil // Skip invalid images
	}

	// Convert ImageableID from string to int64
	imageableID, err := strconv.ParseInt(contextualImage.ImageableID, 10, 64)
	if err != nil {
		return nil // Skip invalid ImageableID
	}

	image := &models.Image{
		URL:           discogsImage.URI,
		ImageType:     models.ImageTypePrimary, // Default to primary, could be enhanced later
		ImageableID:   imageableID,
		ImageableType: contextualImage.ImageableType,
	}

	// Set Discogs-specific fields
	if discogsImage.Type != "" {
		image.DiscogsType = &discogsImage.Type
		// Map Discogs type to our ImageType
		switch discogsImage.Type {
		case "primary":
			image.ImageType = models.ImageTypePrimary
		case "secondary":
			image.ImageType = models.ImageTypeSecondary
		default:
			image.ImageType = models.ImageTypeGallery
		}
	}

	if discogsImage.URI != "" {
		image.DiscogsURI = &discogsImage.URI
	}

	if discogsImage.URI150 != "" {
		image.DiscogsURI150 = &discogsImage.URI150
	}

	if discogsImage.Width > 0 {
		width := int(discogsImage.Width)
		image.Width = &width
	}

	if discogsImage.Height > 0 {
		height := int(discogsImage.Height)
		image.Height = &height
	}

	return image
}

func (s *SimplifiedXMLProcessingService) convertDiscogsArtistToModel(
	discogsArtist *imports.Artist,
) *models.Artist {
	if discogsArtist == nil || discogsArtist.ID <= 0 {
		return nil
	}

	artist := &models.Artist{
		DiscogsID: int64(discogsArtist.ID),
		Name:      discogsArtist.Name,
		IsActive:  true, // Default to active
	}

	return artist
}

// processMasterArtistAssociationBuffer processes master-artist associations in batches
func (s *SimplifiedXMLProcessingService) processMasterArtistAssociationBuffer(
	ctx context.Context,
	buffers *ProcessingBuffers,
	wg *sync.WaitGroup,
	processingID string,
) {
	defer wg.Done()
	log := s.log.Function("processMasterArtistAssociationBuffer")

	// Collect exact association pairs for batch processing
	associations := make([]*MasterArtistAssociation, 0, 5000)
	totalProcessed := 0

	for {
		select {
		case <-ctx.Done():
			// Context cancelled, process remaining associations if any
			if len(associations) > 0 {
				s.processPendingMasterArtistAssociations(ctx, associations, processingID)
			}
			return

		case association, ok := <-buffers.MasterArtists.Channel:
			if !ok {
				// Channel closed, process remaining associations
				if len(associations) > 0 {
					s.processPendingMasterArtistAssociations(ctx, associations, processingID)
					totalProcessed += len(associations)
				}
				log.Info(
					"Master-artist association buffer processing completed",
					"totalProcessed",
					totalProcessed,
					"processingID",
					processingID,
				)
				return
			}

			if association != nil {
				// Add exact association pair to batch
				associations = append(associations, association)

				// Process batch when we reach 1000 associations
				if len(associations) >= 5000 {
					s.processPendingMasterArtistAssociations(ctx, associations, processingID)
					totalProcessed += len(associations)
					associations = make([]*MasterArtistAssociation, 0, 5000) // Reset slice
				}
			}
		}
	}
}

// processMasterGenreAssociationBuffer processes master-genre associations in batches
func (s *SimplifiedXMLProcessingService) processMasterGenreAssociationBuffer(
	ctx context.Context,
	buffers *ProcessingBuffers,
	wg *sync.WaitGroup,
	processingID string,
) {
	defer wg.Done()
	log := s.log.Function("processMasterGenreAssociationBuffer")

	// Group associations by master and genre for bulk operations
	masterGenreMap := make(map[int64][]string) // masterID -> []genreNames
	totalProcessed := 0

	for {
		select {
		case <-ctx.Done():
			// Context cancelled, process remaining associations if any
			if len(masterGenreMap) > 0 {
				s.processPendingMasterGenreAssociations(ctx, masterGenreMap, processingID)
			}
			return

		case association, ok := <-buffers.MasterGenres.Channel:
			if !ok {
				// Channel closed, process remaining associations
				if len(masterGenreMap) > 0 {
					s.processPendingMasterGenreAssociations(ctx, masterGenreMap, processingID)
					totalProcessed += len(masterGenreMap)
				}
				log.Info(
					"Master-genre association buffer processing completed",
					"totalProcessed",
					totalProcessed,
					"processingID",
					processingID,
				)
				return
			}

			if association != nil {
				// Group by master for efficient processing
				if masterGenreMap[association.MasterDiscogsID] == nil {
					masterGenreMap[association.MasterDiscogsID] = make([]string, 0)
				}
				masterGenreMap[association.MasterDiscogsID] = append(
					masterGenreMap[association.MasterDiscogsID],
					association.GenreName,
				)

				// Process batch when we reach threshold
				if len(masterGenreMap) >= 5000 { // Smaller threshold for associations
					s.processPendingMasterGenreAssociations(ctx, masterGenreMap, processingID)
					totalProcessed += len(masterGenreMap)
					masterGenreMap = make(map[int64][]string) // Reset map
				}
			}
		}
	}
}

// Helper methods for processing pending association batches

func (s *SimplifiedXMLProcessingService) processPendingMasterArtistAssociations(
	ctx context.Context,
	associations []*MasterArtistAssociation,
	processingID string,
) {
	log := s.log.Function("processPendingMasterArtistAssociations")
	if len(associations) == 0 {
		return
	}

	// Convert service associations to repository associations
	repoAssociations := make([]repositories.MasterArtistAssociation, len(associations))
	for i, assoc := range associations {
		repoAssociations[i] = repositories.MasterArtistAssociation{
			MasterDiscogsID: assoc.MasterDiscogsID,
			ArtistDiscogsID: assoc.ArtistDiscogsID,
		}
	}

	// Create exact association pairs using repository method
	if err := s.masterRepo.CreateMasterArtistAssociations(ctx, repoAssociations); err != nil {
		log.Error(
			"Failed to create master-artist associations",
			"error",
			err,
			"associationCount",
			len(associations),
			"processingID",
			processingID,
		)
		return
	}

	log.Info(
		"Processed master-artist associations",
		"associationCount",
		len(associations),
		"processingID",
		processingID,
	)
}

func (s *SimplifiedXMLProcessingService) processPendingMasterGenreAssociations(
	ctx context.Context,
	masterGenreMap map[int64][]string,
	processingID string,
) {
	log := s.log.Function("processPendingMasterGenreAssociations")
	if len(masterGenreMap) == 0 {
		return
	}

	// Extract all unique master IDs and genre names
	masterIDs := make([]int64, 0, len(masterGenreMap))
	genreNameSet := make(map[string]bool)

	for masterID, genreNames := range masterGenreMap {
		masterIDs = append(masterIDs, masterID)
		for _, genreName := range genreNames {
			genreNameSet[genreName] = true
		}
	}

	genreNames := make([]string, 0, len(genreNameSet))
	for genreName := range genreNameSet {
		genreNames = append(genreNames, genreName)
	}

	// Create associations using repository method
	if err := s.masterRepo.CreateMasterGenreAssociations(ctx, masterIDs, genreNames); err != nil {
		log.Error(
			"Failed to create master-genre associations",
			"error",
			err,
			"masterCount",
			len(masterIDs),
			"genreCount",
			len(genreNames),
			"processingID",
			processingID,
		)
		return
	}

	log.Info(
		"Processed master-genre associations",
		"masterCount",
		len(masterIDs),
		"genreCount",
		len(genreNames),
		"processingID",
		processingID,
	)
}
