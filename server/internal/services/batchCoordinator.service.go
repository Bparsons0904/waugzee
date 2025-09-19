package services

import (
	"context"
	"strconv"
	"sync"
	"waugzee/internal/imports"
	"waugzee/internal/logger"
	"waugzee/internal/models"
	"waugzee/internal/repositories"
)

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
	masterBatch  map[int64]*models.Master
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
		masterBatch:  make(map[int64]*models.Master),
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
	if discogsArtist == nil || discogsArtist.ID <= 0 {
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

	bc.masterBatch[master.DiscogsID] = master

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

	batch := make([]*models.Master, 0, len(bc.masterBatch))
	for _, master := range bc.masterBatch {
		batch = append(batch, master)
	}

	err := bc.masterRepo.UpsertBatch(ctx, batch)
	if err != nil {
		return log.Err("Failed to upsert master batch", err)
	}

	bc.masterBatch = make(map[int64]*models.Master)
	return nil
}

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
	for _, release := range bc.releaseBatch {
		batch = append(batch, release)
	}

	err := bc.releaseRepo.UpsertBatch(ctx, batch)
	if err != nil {
		return log.Err("Failed to upsert release batch", err)
	}

	bc.releaseBatch = make(map[int64]*models.Release)
	return nil
}

// FlushAllBatches flushes any remaining items in all batches
func (bc *BatchCoordinator) FlushAllBatches(ctx context.Context) error {
	var err error

	if flushErr := bc.FlushImageBatch(ctx); flushErr != nil {
		err = flushErr
	}
	if flushErr := bc.FlushGenreBatch(ctx); flushErr != nil {
		err = flushErr
	}
	if flushErr := bc.FlushArtistBatch(ctx); flushErr != nil {
		err = flushErr
	}
	if flushErr := bc.FlushLabelBatch(ctx); flushErr != nil {
		err = flushErr
	}
	if flushErr := bc.FlushMasterBatch(ctx); flushErr != nil {
		err = flushErr
	}
	if flushErr := bc.FlushReleaseBatch(ctx); flushErr != nil {
		err = flushErr
	}

	return err
}
