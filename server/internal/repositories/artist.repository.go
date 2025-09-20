package repositories

import (
	"context"
	"waugzee/internal/database"
	"waugzee/internal/logger"
	. "waugzee/internal/models"
	"waugzee/internal/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ArtistRepository interface {
	GetByID(ctx context.Context, id string) (*Artist, error)
	GetByDiscogsID(ctx context.Context, discogsID int64) (*Artist, error)
	Create(ctx context.Context, artist *Artist) (*Artist, error)
	Update(ctx context.Context, artist *Artist) error
	Delete(ctx context.Context, id string) error
	UpsertBatch(ctx context.Context, artists []*Artist) error
	GetBatchByDiscogsIDs(ctx context.Context, discogsIDs []int64) (map[int64]*Artist, error)
	GetHashesByDiscogsIDs(ctx context.Context, discogsIDs []int64) (map[int64]string, error)
	InsertBatch(ctx context.Context, artists []*Artist) error
	UpdateBatch(ctx context.Context, artists []*Artist) error
	FindOrCreateByDiscogsID(ctx context.Context, discogsID int64, name string) (*Artist, error)
}

type artistRepository struct {
	db  database.DB
	log logger.Logger
}

func NewArtistRepository(db database.DB) ArtistRepository {
	return &artistRepository{
		db:  db,
		log: logger.New("artistRepository"),
	}
}

func (r *artistRepository) GetByID(ctx context.Context, id string) (*Artist, error) {
	log := r.log.Function("GetByID")

	artistID, err := uuid.Parse(id)
	if err != nil {
		return nil, log.Err("failed to parse artist ID", err, "id", id)
	}

	var artist Artist
	if err := r.db.SQLWithContext(ctx).First(&artist, "id = ?", artistID).Error; err != nil {
		return nil, log.Err("failed to get artist by ID", err, "id", id)
	}

	return &artist, nil
}

func (r *artistRepository) GetByDiscogsID(ctx context.Context, discogsID int64) (*Artist, error) {
	log := r.log.Function("GetByDiscogsID")

	var artist Artist
	if err := r.db.SQLWithContext(ctx).First(&artist, "id = ?", discogsID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, log.Err("failed to get artist by Discogs ID", err, "discogsID", discogsID)
	}

	return &artist, nil
}

func (r *artistRepository) Create(ctx context.Context, artist *Artist) (*Artist, error) {
	log := r.log.Function("Create")

	if err := r.db.SQLWithContext(ctx).Create(artist).Error; err != nil {
		return nil, log.Err("failed to create artist", err, "artist", artist)
	}

	return artist, nil
}

func (r *artistRepository) Update(ctx context.Context, artist *Artist) error {
	log := r.log.Function("Update")

	if err := r.db.SQLWithContext(ctx).Save(artist).Error; err != nil {
		return log.Err("failed to update artist", err, "artist", artist)
	}

	return nil
}

func (r *artistRepository) Delete(ctx context.Context, id string) error {
	log := r.log.Function("Delete")

	artistID, err := uuid.Parse(id)
	if err != nil {
		return log.Err("failed to parse artist ID", err, "id", id)
	}

	if err := r.db.SQLWithContext(ctx).Delete(&Artist{}, "id = ?", artistID).Error; err != nil {
		return log.Err("failed to delete artist", err, "id", id)
	}

	return nil
}

func (r *artistRepository) UpsertBatch(ctx context.Context, artists []*Artist) error {
	log := r.log.Function("UpsertBatch")

	if len(artists) == 0 {
		return nil
	}

	discogsIDs := make([]int64, len(artists))
	for i, artist := range artists {
		discogsIDs[i] = artist.ID
	}

	existingHashes, err := r.GetHashesByDiscogsIDs(ctx, discogsIDs)
	if err != nil {
		_ = log.Err("failed to get existing hashes", err, "count", len(discogsIDs))
	}

	hashableRecords := make([]utils.DiscogsHashable, len(artists))
	for i, artist := range artists {
		hashableRecords[i] = artist
	}

	categories := utils.CategorizeRecordsByHash(hashableRecords, existingHashes)

	if len(categories.Insert) > 0 {
		insertArtists := make([]*Artist, len(categories.Insert))
		for i, record := range categories.Insert {
			insertArtists[i] = record.(*Artist)
		}
		err = r.InsertBatch(ctx, insertArtists)
		if err != nil {
			return log.Err("failed to insert artist batch", err, "count", len(insertArtists))
		}
	}

	if len(categories.Update) > 0 {
		updateArtists := make([]*Artist, len(categories.Update))
		for i, record := range categories.Update {
			updateArtists[i] = record.(*Artist)
		}
		err = r.UpdateBatch(ctx, updateArtists)
		if err != nil {
			return log.Err("failed to update artist batch", err, "count", len(updateArtists))
		}
	}

	return nil
}

func (r *artistRepository) GetBatchByDiscogsIDs(
	ctx context.Context,
	discogsIDs []int64,
) (map[int64]*Artist, error) {
	log := r.log.Function("GetBatchByDiscogsIDs")

	if len(discogsIDs) == 0 {
		return make(map[int64]*Artist), nil
	}

	var artists []*Artist
	if err := r.db.SQLWithContext(ctx).Where("id IN ?", discogsIDs).Find(&artists).Error; err != nil {
		return nil, log.Err("failed to get artists by Discogs IDs", err, "count", len(discogsIDs))
	}

	// Convert to map for O(1) lookup
	result := make(map[int64]*Artist, len(artists))
	for _, artist := range artists {
		result[artist.ID] = artist
	}

	return result, nil
}

func (r *artistRepository) GetHashesByDiscogsIDs(
	ctx context.Context,
	discogsIDs []int64,
) (map[int64]string, error) {
	log := r.log.Function("GetHashesByDiscogsIDs")

	if len(discogsIDs) == 0 {
		return make(map[int64]string), nil
	}

	var artists []struct {
		ID   int64  `json:"discogsId"`
		ContentHash string `json:"contentHash"`
	}

	if err := r.db.SQLWithContext(ctx).
		Model(&Artist{}).
		Select("id, content_hash").
		Where("id IN ?", discogsIDs).
		Find(&artists).Error; err != nil {
		return nil, log.Err(
			"failed to get artist hashes by Discogs IDs",
			err,
			"count",
			len(discogsIDs),
		)
	}

	result := make(map[int64]string, len(artists))
	for _, artist := range artists {
		result[artist.ID] = artist.ContentHash
	}

	return result, nil
}

func (r *artistRepository) InsertBatch(ctx context.Context, artists []*Artist) error {
	log := r.log.Function("InsertBatch")

	if len(artists) == 0 {
		return nil
	}

	if err := r.db.SQLWithContext(ctx).Create(&artists).Error; err != nil {
		return log.Err("failed to insert artist batch", err, "count", len(artists))
	}

	return nil
}

func (r *artistRepository) UpdateBatch(ctx context.Context, artists []*Artist) error {
	log := r.log.Function("UpdateBatch")

	if len(artists) == 0 {
		return nil
	}

	if err := r.db.SQLWithContext(ctx).Save(&artists).Error; err != nil {
		return log.Err("failed to update artist batch", err, "count", len(artists))
	}

	return nil
}

func (r *artistRepository) FindOrCreateByDiscogsID(
	ctx context.Context,
	discogsID int64,
	name string,
) (*Artist, error) {
	log := r.log.Function("FindOrCreateByDiscogsID")

	if discogsID == 0 || name == "" {
		return nil, log.Err(
			"artist discogsID and name cannot be empty",
			nil,
			"discogsID",
			discogsID,
			"name",
			name,
		)
	}

	// First, try to find existing artist by Discogs ID
	artist, err := r.GetByDiscogsID(ctx, discogsID)
	if err != nil {
		return nil, log.Err("failed to get artist by Discogs ID", err, "discogsID", discogsID)
	}

	if artist != nil {
		return artist, nil
	}

	// If not found, create new artist with the DiscogsID
	newArtist := &Artist{
		ID:   discogsID,
		Name: name,
	}

	createdArtist, err := r.Create(ctx, newArtist)
	if err != nil {
		return nil, log.Err(
			"failed to create new artist",
			err,
			"name",
			name,
			"discogsID",
			discogsID,
		)
	}

	log.Info("Created new artist", "name", name, "discogsID", createdArtist.ID)
	return createdArtist, nil
}
