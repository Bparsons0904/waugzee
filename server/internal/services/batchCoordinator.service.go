package services

import (
	"context"
	"strconv"
	"sync"
	"time"
	"waugzee/internal/imports"
	"waugzee/internal/logger"
	"waugzee/internal/models"
	"waugzee/internal/repositories"
)

// MasterWithAssociations bundles a master with its associations for atomic processing
type MasterWithAssociations struct {
	Master           *models.Master
	ArtistAssocs     []repositories.MasterArtistAssociation
	GenreAssocs      map[string]bool // Use map for deduplication
}

type BatchCoordinator struct {
	labelRepo   repositories.LabelRepository
	artistRepo  repositories.ArtistRepository
	masterRepo  repositories.MasterRepository
	releaseRepo repositories.ReleaseRepository
	genreRepo   repositories.GenreRepository
	imageRepo   repositories.ImageRepository
	log         logger.Logger

	// Batch storage maps for deduplication with mutexes for thread safety
	imageBatch   map[imageKey]*models.Image
	imageMutex   sync.RWMutex
	genreBatch   map[string]*models.Genre
	genreMutex   sync.RWMutex
	artistBatch  map[int64]*models.Artist
	artistMutex  sync.RWMutex
	labelBatch   map[int64]*models.Label
	labelMutex   sync.RWMutex
	masterBatch  map[int64]*MasterWithAssociations
	masterMutex  sync.RWMutex
	releaseBatch map[int64]*models.Release
	releaseMutex sync.RWMutex
}

// imageKey represents a composite key for image deduplication
type imageKey struct {
	imageableID   int64
	imageableType string
	url           string
}

func NewBatchCoordinator(
	labelRepo repositories.LabelRepository,
	artistRepo repositories.ArtistRepository,
	masterRepo repositories.MasterRepository,
	releaseRepo repositories.ReleaseRepository,
	genreRepo repositories.GenreRepository,
	imageRepo repositories.ImageRepository,
) *BatchCoordinator {
	return &BatchCoordinator{
		labelRepo:   labelRepo,
		artistRepo:  artistRepo,
		masterRepo:  masterRepo,
		releaseRepo: releaseRepo,
		genreRepo:   genreRepo,
		imageRepo:   imageRepo,
		log:         logger.New("batchCoordinator"),

		imageBatch:   make(map[imageKey]*models.Image),
		genreBatch:   make(map[string]*models.Genre),
		artistBatch:  make(map[int64]*models.Artist),
		labelBatch:   make(map[int64]*models.Label),
		masterBatch:  make(map[int64]*MasterWithAssociations),
		releaseBatch: make(map[int64]*models.Release),
	}
}

// Entity conversion methods

func (bc *BatchCoordinator) ConvertContextualDiscogsImageToModel(
	contextualImage *ContextualDiscogsImage,
) *models.Image {
	if contextualImage == nil || contextualImage.DiscogsImage == nil {
		return nil
	}

	discogsImage := contextualImage.DiscogsImage

	if discogsImage.URI == "" || contextualImage.ImageableID == "" ||
		contextualImage.ImageableType == "" {
		return nil
	}

	imageableID, err := strconv.ParseInt(contextualImage.ImageableID, 10, 64)
	if err != nil {
		return nil
	}

	image := &models.Image{
		URL:           discogsImage.URI,
		ImageType:     models.ImageTypePrimary,
		ImageableID:   imageableID,
		ImageableType: contextualImage.ImageableType,
	}

	if discogsImage.Type != "" {
		image.DiscogsType = &discogsImage.Type
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

func (bc *BatchCoordinator) ConvertDiscogsArtistToModel(
	discogsArtist *imports.Artist,
) *models.Artist {
	if discogsArtist == nil {
		bc.log.Warn("Dropping artist due to nil input", "reason", "discogsArtist is nil")
		return nil
	}

	if discogsArtist.ID <= 0 {
		bc.log.Warn("Dropping artist due to invalid ID", "discogsID", discogsArtist.ID, "name", discogsArtist.Name, "reason", "invalid ID")
		return nil
	}

	if len(discogsArtist.Name) == 0 {
		bc.log.Warn("Dropping artist due to empty name", "discogsID", discogsArtist.ID, "name", discogsArtist.Name, "reason", "empty name")
		return nil
	}

	artist := &models.Artist{
		DiscogsID: int64(discogsArtist.ID),
		Name:      discogsArtist.Name,
		IsActive:  true,
	}

	return artist
}

// Batch management methods

func (bc *BatchCoordinator) AddImageToBatch(
	ctx context.Context,
	image *models.Image,
) error {
	bc.imageMutex.Lock()
	defer bc.imageMutex.Unlock()

	key := imageKey{
		imageableID:   image.ImageableID,
		imageableType: image.ImageableType,
		url:           image.URL,
	}
	bc.imageBatch[key] = image

	if len(bc.imageBatch) >= 5000 {
		return bc.flushImageBatchInternal(ctx)
	}
	return nil
}

func (bc *BatchCoordinator) AddGenreToBatch(
	ctx context.Context,
	genreName string,
) error {
	bc.genreMutex.Lock()
	defer bc.genreMutex.Unlock()

	if genreName != "" && bc.genreBatch[genreName] == nil {
		bc.genreBatch[genreName] = &models.Genre{Name: genreName}

		if len(bc.genreBatch) >= 5000 {
			return bc.flushGenreBatchInternal(ctx)
		}
	}
	return nil
}

func (bc *BatchCoordinator) AddArtistToBatch(
	ctx context.Context,
	artist *models.Artist,
) error {
	bc.artistMutex.Lock()
	defer bc.artistMutex.Unlock()

	bc.artistBatch[artist.DiscogsID] = artist

	if len(bc.artistBatch) >= 5000 {
		return bc.flushArtistBatchInternal(ctx)
	}
	return nil
}

func (bc *BatchCoordinator) AddLabelToBatch(
	ctx context.Context,
	label *models.Label,
) error {
	bc.labelMutex.Lock()
	defer bc.labelMutex.Unlock()

	bc.labelBatch[label.DiscogsID] = label

	if len(bc.labelBatch) >= 5000 {
		return bc.flushLabelBatchInternal(ctx)
	}
	return nil
}

func (bc *BatchCoordinator) AddMasterToBatch(
	ctx context.Context,
	master *models.Master,
) error {
	bc.masterMutex.Lock()
	defer bc.masterMutex.Unlock()

	// Create or get existing MasterWithAssociations
	masterWithAssocs, exists := bc.masterBatch[master.DiscogsID]
	if !exists {
		masterWithAssocs = &MasterWithAssociations{
			Master:       master,
			ArtistAssocs: make([]repositories.MasterArtistAssociation, 0),
			GenreAssocs:  make(map[string]bool),
		}
		bc.masterBatch[master.DiscogsID] = masterWithAssocs
	} else {
		// Update the master in case it was modified
		masterWithAssocs.Master = master
	}

	if len(bc.masterBatch) >= 5000 {
		return bc.flushMasterBatchInternal(ctx)
	}
	return nil
}

func (bc *BatchCoordinator) AddReleaseToBatch(
	ctx context.Context,
	release *models.Release,
) error {
	bc.releaseMutex.Lock()
	defer bc.releaseMutex.Unlock()

	bc.releaseBatch[release.DiscogsID] = release

	if len(bc.releaseBatch) >= 2000 {
		return bc.flushReleaseBatchInternal(ctx)
	}
	return nil
}

func (bc *BatchCoordinator) AddMasterArtistAssociation(
	masterDiscogsID int64,
	artistDiscogsID int64,
) {
	bc.masterMutex.Lock()
	defer bc.masterMutex.Unlock()

	// Get or create the master with associations
	masterWithAssocs, exists := bc.masterBatch[masterDiscogsID]
	if !exists {
		// Master doesn't exist yet, create placeholder
		masterWithAssocs = &MasterWithAssociations{
			Master:       nil, // Will be set when master is added
			ArtistAssocs: make([]repositories.MasterArtistAssociation, 0),
			GenreAssocs:  make(map[string]bool),
		}
		bc.masterBatch[masterDiscogsID] = masterWithAssocs
	}

	// Add the association
	masterWithAssocs.ArtistAssocs = append(masterWithAssocs.ArtistAssocs, repositories.MasterArtistAssociation{
		MasterDiscogsID: masterDiscogsID,
		ArtistDiscogsID: artistDiscogsID,
	})
}

func (bc *BatchCoordinator) AddMasterGenreAssociation(
	masterDiscogsID int64,
	genreName string,
) {
	bc.masterMutex.Lock()
	defer bc.masterMutex.Unlock()

	// Get or create the master with associations
	masterWithAssocs, exists := bc.masterBatch[masterDiscogsID]
	if !exists {
		// Master doesn't exist yet, create placeholder
		masterWithAssocs = &MasterWithAssociations{
			Master:       nil, // Will be set when master is added
			ArtistAssocs: make([]repositories.MasterArtistAssociation, 0),
			GenreAssocs:  make(map[string]bool),
		}
		bc.masterBatch[masterDiscogsID] = masterWithAssocs
	}

	// Add the genre (map automatically deduplicates)
	masterWithAssocs.GenreAssocs[genreName] = true
}

// Batch flushing methods

func (bc *BatchCoordinator) FlushImageBatch(ctx context.Context) error {
	bc.imageMutex.Lock()
	defer bc.imageMutex.Unlock()
	return bc.flushImageBatchInternal(ctx)
}

func (bc *BatchCoordinator) flushImageBatchInternal(
	ctx context.Context,
) error {
	log := bc.log.Function("flushImageBatchInternal")
	if len(bc.imageBatch) == 0 {
		return nil
	}

	batch := make([]*models.Image, 0, len(bc.imageBatch))
	for _, image := range bc.imageBatch {
		batch = append(batch, image)
	}

	_, _, err := bc.imageRepo.UpsertBatch(ctx, batch)
	if err != nil {
		return log.Err("Failed to upsert image batch", err)
	}

	bc.imageBatch = make(map[imageKey]*models.Image)
	return nil
}

func (bc *BatchCoordinator) FlushGenreBatch(ctx context.Context) error {
	bc.genreMutex.Lock()
	defer bc.genreMutex.Unlock()
	return bc.flushGenreBatchInternal(ctx)
}

func (bc *BatchCoordinator) flushGenreBatchInternal(
	ctx context.Context,
) error {
	log := bc.log.Function("flushGenreBatchInternal")
	if len(bc.genreBatch) == 0 {
		return nil
	}

	batch := make([]*models.Genre, 0, len(bc.genreBatch))
	for _, genre := range bc.genreBatch {
		batch = append(batch, genre)
	}

	err := bc.genreRepo.UpsertBatch(ctx, batch)
	if err != nil {
		return log.Err("Failed to upsert genre batch", err)
	}

	bc.genreBatch = make(map[string]*models.Genre)
	return nil
}

func (bc *BatchCoordinator) FlushArtistBatch(ctx context.Context) error {
	bc.artistMutex.Lock()
	defer bc.artistMutex.Unlock()
	return bc.flushArtistBatchInternal(ctx)
}

func (bc *BatchCoordinator) flushArtistBatchInternal(
	ctx context.Context,
) error {
	log := bc.log.Function("flushArtistBatchInternal")
	if len(bc.artistBatch) == 0 {
		return nil
	}

	batch := make([]*models.Artist, 0, len(bc.artistBatch))
	for _, artist := range bc.artistBatch {
		batch = append(batch, artist)
	}

	err := bc.artistRepo.UpsertBatch(ctx, batch)
	if err != nil {
		return log.Err("Failed to upsert artist batch", err)
	}

	bc.artistBatch = make(map[int64]*models.Artist)
	return nil
}

func (bc *BatchCoordinator) FlushLabelBatch(ctx context.Context) error {
	bc.labelMutex.Lock()
	defer bc.labelMutex.Unlock()
	return bc.flushLabelBatchInternal(ctx)
}

func (bc *BatchCoordinator) flushLabelBatchInternal(
	ctx context.Context,
) error {
	log := bc.log.Function("flushLabelBatchInternal")
	if len(bc.labelBatch) == 0 {
		return nil
	}

	batch := make([]*models.Label, 0, len(bc.labelBatch))
	for _, label := range bc.labelBatch {
		batch = append(batch, label)
	}

	err := bc.labelRepo.UpsertBatch(ctx, batch)
	if err != nil {
		return log.Err("Failed to upsert label batch", err)
	}

	bc.labelBatch = make(map[int64]*models.Label)
	return nil
}

func (bc *BatchCoordinator) FlushMasterBatch(ctx context.Context) error {
	bc.masterMutex.Lock()
	defer bc.masterMutex.Unlock()
	return bc.flushMasterBatchInternal(ctx)
}

func (bc *BatchCoordinator) flushMasterBatchInternal(
	ctx context.Context,
) error {
	log := bc.log.Function("flushMasterBatchInternal")
	if len(bc.masterBatch) == 0 {
		return nil
	}

	// Extract masters for batch processing
	batch := make([]*models.Master, 0, len(bc.masterBatch))
	for _, masterWithAssocs := range bc.masterBatch {
		if masterWithAssocs.Master != nil {
			batch = append(batch, masterWithAssocs.Master)
		}
	}

	// Insert all masters first
	err := bc.masterRepo.UpsertBatch(ctx, batch)
	if err != nil {
		return log.Err("Failed to upsert master batch", err)
	}

	// Ensure all artists are flushed before processing associations
	if err := bc.FlushArtistBatch(ctx); err != nil {
		log.Er("Failed to flush artist batch before associations", err)
	}

	// Small delay to ensure database consistency
	time.Sleep(50 * time.Millisecond)

	// Process each master's associations immediately after masters and artists are inserted
	for _, masterWithAssocs := range bc.masterBatch {
		if masterWithAssocs.Master != nil {
			if err := bc.flushSingleMasterAssociations(ctx, masterWithAssocs); err != nil {
				log.Er("Failed to flush associations for master", err, "masterID", masterWithAssocs.Master.DiscogsID)
				// Continue processing other masters even if one fails
			}
		}
	}

	bc.masterBatch = make(map[int64]*MasterWithAssociations)
	return nil
}

func (bc *BatchCoordinator) flushSingleMasterAssociations(
	ctx context.Context,
	masterWithAssocs *MasterWithAssociations,
) error {
	log := bc.log.Function("flushSingleMasterAssociations")

	// Process master-artist associations
	if len(masterWithAssocs.ArtistAssocs) > 0 {
		log.Info("Processing master-artist associations", 
			"masterID", masterWithAssocs.Master.DiscogsID,
			"count", len(masterWithAssocs.ArtistAssocs))
		if err := bc.masterRepo.CreateMasterArtistAssociations(ctx, masterWithAssocs.ArtistAssocs); err != nil {
			return log.Err("Failed to create master-artist associations", err)
		}
	}

	// Process master-genre associations
	if len(masterWithAssocs.GenreAssocs) > 0 {
		log.Info("Processing master-genre associations", 
			"masterID", masterWithAssocs.Master.DiscogsID,
			"genreCount", len(masterWithAssocs.GenreAssocs))
		
		masterIDs := []int64{masterWithAssocs.Master.DiscogsID}
		genreNames := make([]string, 0, len(masterWithAssocs.GenreAssocs))
		for genreName := range masterWithAssocs.GenreAssocs {
			genreNames = append(genreNames, genreName)
		}

		if err := bc.masterRepo.CreateMasterGenreAssociations(ctx, masterIDs, genreNames); err != nil {
			return log.Err("Failed to create master-genre associations", err)
		}
	}

	return nil
}

// Note: Old flushMasterAssociations method removed - now using flushSingleMasterAssociations

func (bc *BatchCoordinator) FlushReleaseBatch(ctx context.Context) error {
	bc.releaseMutex.Lock()
	defer bc.releaseMutex.Unlock()
	return bc.flushReleaseBatchInternal(ctx)
}

func (bc *BatchCoordinator) flushReleaseBatchInternal(
	ctx context.Context,
) error {
	log := bc.log.Function("flushReleaseBatchInternal")
	if len(bc.releaseBatch) == 0 {
		return nil
	}

	batch := make([]*models.Release, 0, len(bc.releaseBatch))
	masterIDCount := 0
	labelIDCount := 0
	sampleMasterIDs := make([]int64, 0, 5)
	sampleLabelIDs := make([]int64, 0, 5)

	for _, release := range bc.releaseBatch {
		batch = append(batch, release)

		// Count and sample MasterIDs for debugging
		if release.MasterID != nil {
			masterIDCount++
			if len(sampleMasterIDs) < 5 {
				sampleMasterIDs = append(sampleMasterIDs, *release.MasterID)
			}
		}

		// Count and sample LabelIDs for debugging
		if release.LabelID != nil {
			labelIDCount++
			if len(sampleLabelIDs) < 5 {
				sampleLabelIDs = append(sampleLabelIDs, *release.LabelID)
			}
		}
	}

	err := bc.releaseRepo.UpsertBatch(ctx, batch)
	if err != nil {
		return log.Err("Failed to upsert release batch", err)
	}

	bc.releaseBatch = make(map[int64]*models.Release)
	return nil
}

// FlushAllBatches flushes any remaining items in all batches
// CRITICAL: Sequential processing in strict dependency order to avoid foreign key violations
// Order: Labels → Artists → Masters → Releases
func (bc *BatchCoordinator) FlushAllBatches(ctx context.Context) error {
	log := bc.log.Function("FlushAllBatches")

	// Step 1: Flush Labels first (no dependencies)
	log.Info("Step 1: Flushing labels")
	if err := bc.FlushLabelBatch(ctx); err != nil {
		log.Er("Failed to flush label batch", err)
		return err
	}

	// Step 2: Flush Artists (no dependencies)
	log.Info("Step 2: Flushing artists")
	if err := bc.FlushArtistBatch(ctx); err != nil {
		log.Er("Failed to flush artist batch", err)
		return err
	}

	// Step 3: Flush Masters (depends on Artists for associations)
	log.Info("Step 3: Flushing masters")
	if err := bc.FlushMasterBatch(ctx); err != nil {
		log.Er("Failed to flush master batch", err)
		return err
	}

	// Step 4: Flush Releases (depends on Masters and Labels)
	log.Info("Step 4: Flushing releases")
	if err := bc.FlushReleaseBatch(ctx); err != nil {
		log.Er("Failed to flush release batch", err)
		return err
	}

	// Step 5: Flush remaining independent entities
	log.Info("Step 5: Flushing remaining entities")
	if err := bc.FlushGenreBatch(ctx); err != nil {
		log.Er("Failed to flush genre batch", err)
		return err
	}
	if err := bc.FlushImageBatch(ctx); err != nil {
		log.Er("Failed to flush image batch", err)
		return err
	}

	log.Info("All batches flushed successfully")
	return nil
}
