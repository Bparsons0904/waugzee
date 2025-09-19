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
		// Association buffers (Master-level only)
		MasterArtists: &MasterArtistAssociationBuffer{
			Channel:  make(chan *MasterArtistAssociation, BUFFER_CHANNEL_SIZE),
			Capacity: BUFFER_CHANNEL_SIZE,
		},
		MasterGenres: &MasterGenreAssociationBuffer{
			Channel:  make(chan *MasterGenreAssociation, BUFFER_CHANNEL_SIZE),
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
	// Close association buffers (Master-level only)
	close(buffers.MasterArtists.Channel)
	close(buffers.MasterGenres.Channel)
}

// StartBufferProcessors starts all buffer processing goroutines
func (bm *BufferManager) StartBufferProcessors(
	ctx context.Context,
	buffers *ProcessingBuffers,
	wg *sync.WaitGroup,
	batchCoordinator *BatchCoordinator,
) {
	wg.Add(8) // Add for each buffer processor
	go bm.processImageBuffer(ctx, buffers, wg, batchCoordinator)
	go bm.processGenreBuffer(ctx, buffers, wg, batchCoordinator)
	go bm.processArtistBuffer(ctx, buffers, wg, batchCoordinator)
	go bm.processLabelBuffer(ctx, buffers, wg, batchCoordinator)
	go bm.processMasterBuffer(ctx, buffers, wg, batchCoordinator)
	go bm.processReleaseBuffer(ctx, buffers, wg, batchCoordinator)
	// Association buffer processors (Master-level only)
	go bm.processMasterArtistAssociationBuffer(ctx, buffers, wg)
	go bm.processMasterGenreAssociationBuffer(ctx, buffers, wg)
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
			}
		}
	}
}

func (bm *BufferManager) processMasterArtistAssociationBuffer(
	ctx context.Context,
	buffers *ProcessingBuffers,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	associations := make([]*MasterArtistAssociation, 0, 5000)

	for {
		select {
		case <-ctx.Done():
			if len(associations) > 0 {
				bm.processPendingMasterArtistAssociations(ctx, associations)
			}
			return

		case association, ok := <-buffers.MasterArtists.Channel:
			if !ok {
				if len(associations) > 0 {
					bm.processPendingMasterArtistAssociations(ctx, associations)
				}
				return
			}

			if association != nil {
				associations = append(associations, association)
				if len(associations) >= 5000 {
					bm.processPendingMasterArtistAssociations(ctx, associations)
					associations = make([]*MasterArtistAssociation, 0, 5000)
				}
			}
		}
	}
}

func (bm *BufferManager) processMasterGenreAssociationBuffer(
	ctx context.Context,
	buffers *ProcessingBuffers,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	masterGenreMap := make(map[int64][]string)

	for {
		select {
		case <-ctx.Done():
			if len(masterGenreMap) > 0 {
				bm.processPendingMasterGenreAssociations(ctx, masterGenreMap)
			}
			return

		case association, ok := <-buffers.MasterGenres.Channel:
			if !ok {
				if len(masterGenreMap) > 0 {
					bm.processPendingMasterGenreAssociations(ctx, masterGenreMap)
				}
				return
			}

			if association != nil {
				if masterGenreMap[association.MasterDiscogsID] == nil {
					masterGenreMap[association.MasterDiscogsID] = make([]string, 0)
				}
				masterGenreMap[association.MasterDiscogsID] = append(
					masterGenreMap[association.MasterDiscogsID],
					association.GenreName,
				)

				if len(masterGenreMap) >= 5000 {
					bm.processPendingMasterGenreAssociations(ctx, masterGenreMap)
					masterGenreMap = make(map[int64][]string)
				}
			}
		}
	}
}

func (bm *BufferManager) processPendingMasterArtistAssociations(
	ctx context.Context,
	associations []*MasterArtistAssociation,
) {
	log := bm.log.Function("processPendingMasterArtistAssociations")
	if len(associations) == 0 {
		return
	}

	repoAssociations := make([]repositories.MasterArtistAssociation, len(associations))
	for i, assoc := range associations {
		repoAssociations[i] = repositories.MasterArtistAssociation{
			MasterDiscogsID: assoc.MasterDiscogsID,
			ArtistDiscogsID: assoc.ArtistDiscogsID,
		}
	}

	if err := bm.masterRepo.CreateMasterArtistAssociations(ctx, repoAssociations); err != nil {
		_ = log.Error("Failed to create master-artist associations", "error", err)
	}
}

func (bm *BufferManager) processPendingMasterGenreAssociations(
	ctx context.Context,
	masterGenreMap map[int64][]string,
) {
	log := bm.log.Function("processPendingMasterGenreAssociations")
	if len(masterGenreMap) == 0 {
		return
	}

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

	if err := bm.masterRepo.CreateMasterGenreAssociations(ctx, masterIDs, genreNames); err != nil {
		_ = log.Error("Failed to create master-genre associations", "error", err)
	}
}

