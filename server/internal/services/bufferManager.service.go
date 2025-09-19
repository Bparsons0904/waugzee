package services

import (
	"context"
	"sync"
	"waugzee/internal/imports"
	"waugzee/internal/logger"
	"waugzee/internal/models"
	"waugzee/internal/repositories"
)

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

// Note: Association processing now handled directly in batch coordinator

// ProcessingBuffers contains all the buffered channels for related entity processing
type ProcessingBuffers struct {
	Images   *ImageBuffer
	Genres   *GenreBuffer
	Artists  *ArtistBuffer
	Labels   *LabelBuffer
	Masters  *MasterBuffer
	Releases *ReleaseBuffer
	// Note: Association processing now handled directly in batch coordinator
}

type BufferManager struct {
	labelRepo   repositories.LabelRepository
	artistRepo  repositories.ArtistRepository
	masterRepo  repositories.MasterRepository
	releaseRepo repositories.ReleaseRepository
	genreRepo   repositories.GenreRepository
	imageRepo   repositories.ImageRepository
	log         logger.Logger
}

func NewBufferManager(
	labelRepo repositories.LabelRepository,
	artistRepo repositories.ArtistRepository,
	masterRepo repositories.MasterRepository,
	releaseRepo repositories.ReleaseRepository,
	genreRepo repositories.GenreRepository,
	imageRepo repositories.ImageRepository,
) *BufferManager {
	return &BufferManager{
		labelRepo:   labelRepo,
		artistRepo:  artistRepo,
		masterRepo:  masterRepo,
		releaseRepo: releaseRepo,
		genreRepo:   genreRepo,
		imageRepo:   imageRepo,
		log:         logger.New("bufferManager"),
	}
}

const (
	BUFFER_CHANNEL_SIZE = 10_000
)

// CreateProcessingBuffers initializes buffered channels for related entity processing
func (bm *BufferManager) CreateProcessingBuffers() *ProcessingBuffers {
	return &ProcessingBuffers{
		Images: &ImageBuffer{
			Channel:  make(chan *ContextualDiscogsImage, BUFFER_CHANNEL_SIZE),
			Capacity: BUFFER_CHANNEL_SIZE,
		},
		Genres: &GenreBuffer{
			Channel:  make(chan string, BUFFER_CHANNEL_SIZE),
			Capacity: BUFFER_CHANNEL_SIZE,
		},
		Artists: &ArtistBuffer{
			Channel:  make(chan *imports.Artist, BUFFER_CHANNEL_SIZE),
			Capacity: BUFFER_CHANNEL_SIZE,
		},
		Labels: &LabelBuffer{
			Channel:  make(chan *models.Label, BUFFER_CHANNEL_SIZE),
			Capacity: BUFFER_CHANNEL_SIZE,
		},
		Masters: &MasterBuffer{
			Channel:  make(chan *models.Master, BUFFER_CHANNEL_SIZE),
			Capacity: BUFFER_CHANNEL_SIZE,
		},
		Releases: &ReleaseBuffer{
			Channel:  make(chan *models.Release, BUFFER_CHANNEL_SIZE),
			Capacity: BUFFER_CHANNEL_SIZE,
		},
	}
}

// CloseProcessingBuffers safely closes all buffer channels
func (bm *BufferManager) CloseProcessingBuffers(buffers *ProcessingBuffers) {
	close(buffers.Images.Channel)
	close(buffers.Genres.Channel)
	close(buffers.Artists.Channel)
	close(buffers.Labels.Channel)
	close(buffers.Masters.Channel)
	close(buffers.Releases.Channel)
}

// StartBufferProcessors starts all buffer processing goroutines
func (bm *BufferManager) StartBufferProcessors(
	ctx context.Context,
	buffers *ProcessingBuffers,
	wg *sync.WaitGroup,
	batchCoordinator *BatchCoordinator,
) {
	wg.Add(6) // Only entity processors, no association processors
	go bm.processImageBuffer(ctx, buffers, wg, batchCoordinator)
	go bm.processGenreBuffer(ctx, buffers, wg, batchCoordinator)
	go bm.processArtistBuffer(ctx, buffers, wg, batchCoordinator)
	go bm.processLabelBuffer(ctx, buffers, wg, batchCoordinator)
	go bm.processMasterBuffer(ctx, buffers, wg, batchCoordinator)
	go bm.processReleaseBuffer(ctx, buffers, wg, batchCoordinator)
	// DISABLED: Association buffer processors (will be handled after entity flushing)
	// go bm.processMasterArtistAssociationBuffer(ctx, buffers, wg)
	// go bm.processMasterGenreAssociationBuffer(ctx, buffers, wg)
}

// Buffer processor methods - these handle the consumption from channels and batching

func (bm *BufferManager) processImageBuffer(
	ctx context.Context,
	buffers *ProcessingBuffers,
	wg *sync.WaitGroup,
	batchCoordinator *BatchCoordinator,
) {
	defer wg.Done()
	log := bm.log.Function("processImageBuffer")

	for {
		select {
		case <-ctx.Done():
			return
		case contextualImage, ok := <-buffers.Images.Channel:
			if !ok {
				return
			}
			if modelImage := batchCoordinator.ConvertContextualDiscogsImageToModel(contextualImage); modelImage != nil {
				if err := batchCoordinator.AddImageToBatch(ctx, modelImage); err != nil {
					log.Error("Failed to add image to batch", "error", err)
				}
			} else {
				log.Warn("Image conversion failed in buffer", "imageableID", contextualImage.ImageableID, "imageableType", contextualImage.ImageableType, "reason", "ConvertContextualDiscogsImageToModel returned nil")
			}
		}
	}
}

func (bm *BufferManager) processGenreBuffer(
	ctx context.Context,
	buffers *ProcessingBuffers,
	wg *sync.WaitGroup,
	batchCoordinator *BatchCoordinator,
) {
	defer wg.Done()
	log := bm.log.Function("processGenreBuffer")

	for {
		select {
		case <-ctx.Done():
			return
		case genreName, ok := <-buffers.Genres.Channel:
			if !ok {
				return
			}
			if genreName != "" {
				if err := batchCoordinator.AddGenreToBatch(ctx, genreName); err != nil {
					log.Error("Failed to add genre to batch", "error", err)
				}
			}
		}
	}
}

func (bm *BufferManager) processArtistBuffer(
	ctx context.Context,
	buffers *ProcessingBuffers,
	wg *sync.WaitGroup,
	batchCoordinator *BatchCoordinator,
) {
	defer wg.Done()
	log := bm.log.Function("processArtistBuffer")

	for {
		select {
		case <-ctx.Done():
			return
		case discogsArtist, ok := <-buffers.Artists.Channel:
			if !ok {
				return
			}
			if modelArtist := batchCoordinator.ConvertDiscogsArtistToModel(discogsArtist); modelArtist != nil {
				if err := batchCoordinator.AddArtistToBatch(ctx, modelArtist); err != nil {
					log.Error("Failed to add artist to batch", "error", err)
				}
			} else {
				log.Warn("Artist conversion failed in buffer", "discogsID", discogsArtist.ID, "name", discogsArtist.Name, "reason", "ConvertDiscogsArtistToModel returned nil")
			}
		}
	}
}

func (bm *BufferManager) processLabelBuffer(
	ctx context.Context,
	buffers *ProcessingBuffers,
	wg *sync.WaitGroup,
	batchCoordinator *BatchCoordinator,
) {
	defer wg.Done()
	log := bm.log.Function("processLabelBuffer")

	for {
		select {
		case <-ctx.Done():
			return
		case modelLabel, ok := <-buffers.Labels.Channel:
			if !ok {
				return
			}
			if modelLabel != nil {
				if err := batchCoordinator.AddLabelToBatch(ctx, modelLabel); err != nil {
					log.Error("Failed to add label to batch", "error", err)
				}
			} else {
				log.Warn("Nil label received in buffer", "reason", "label is nil")
			}
		}
	}
}

func (bm *BufferManager) processMasterBuffer(
	ctx context.Context,
	buffers *ProcessingBuffers,
	wg *sync.WaitGroup,
	batchCoordinator *BatchCoordinator,
) {
	defer wg.Done()
	log := bm.log.Function("processMasterBuffer")

	for {
		select {
		case <-ctx.Done():
			return
		case modelMaster, ok := <-buffers.Masters.Channel:
			if !ok {
				return
			}
			if modelMaster != nil {
				if err := batchCoordinator.AddMasterToBatch(ctx, modelMaster); err != nil {
					log.Error("Failed to add master to batch", "error", err)
				}
			} else {
				log.Warn("Nil master received in buffer", "reason", "master is nil")
			}
		}
	}
}

func (bm *BufferManager) processReleaseBuffer(
	ctx context.Context,
	buffers *ProcessingBuffers,
	wg *sync.WaitGroup,
	batchCoordinator *BatchCoordinator,
) {
	defer wg.Done()
	log := bm.log.Function("processReleaseBuffer")

	for {
		select {
		case <-ctx.Done():
			return
		case modelRelease, ok := <-buffers.Releases.Channel:
			if !ok {
				return
			}
			if modelRelease != nil {
				if err := batchCoordinator.AddReleaseToBatch(ctx, modelRelease); err != nil {
					log.Error("Failed to add release to batch", "error", err)
				}
			} else {
				log.Warn("Nil release received in buffer", "reason", "release is nil")
			}
		}
	}
}

// Note: All association processing removed - now handled directly in batch coordinator

