package repositories

import (
	"context"
	"waugzee/internal/database"
	"waugzee/internal/logger"
	. "waugzee/internal/models"
	contextutil "waugzee/internal/context"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	ARTIST_BATCH_SIZE = 3000
)

type ArtistRepository interface {
	GetByID(ctx context.Context, id string) (*Artist, error)
	GetByDiscogsID(ctx context.Context, discogsID int64) (*Artist, error)
	Create(ctx context.Context, artist *Artist) (*Artist, error)
	Update(ctx context.Context, artist *Artist) error
	Delete(ctx context.Context, id string) error
	UpsertBatch(ctx context.Context, artists []*Artist) (int, int, error)
	GetBatchByDiscogsIDs(ctx context.Context, discogsIDs []int64) (map[int64]*Artist, error)
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

func (r *artistRepository) getDB(ctx context.Context) *gorm.DB {
	if tx, ok := contextutil.GetTransaction(ctx); ok {
		return tx
	}
	return r.db.SQLWithContext(ctx)
}

func (r *artistRepository) GetByID(ctx context.Context, id string) (*Artist, error) {
	log := r.log.Function("GetByID")

	artistID, err := uuid.Parse(id)
	if err != nil {
		return nil, log.Err("failed to parse artist ID", err, "id", id)
	}

	var artist Artist
	if err := r.getDB(ctx).First(&artist, "id = ?", artistID).Error; err != nil {
		return nil, log.Err("failed to get artist by ID", err, "id", id)
	}

	return &artist, nil
}

func (r *artistRepository) GetByDiscogsID(ctx context.Context, discogsID int64) (*Artist, error) {
	log := r.log.Function("GetByDiscogsID")

	var artist Artist
	if err := r.getDB(ctx).First(&artist, "discogs_id = ?", discogsID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, log.Err("failed to get artist by Discogs ID", err, "discogsID", discogsID)
	}

	return &artist, nil
}

func (r *artistRepository) Create(ctx context.Context, artist *Artist) (*Artist, error) {
	log := r.log.Function("Create")

	if err := r.getDB(ctx).Create(artist).Error; err != nil {
		return nil, log.Err("failed to create artist", err, "artist", artist)
	}

	return artist, nil
}

func (r *artistRepository) Update(ctx context.Context, artist *Artist) error {
	log := r.log.Function("Update")

	if err := r.getDB(ctx).Save(artist).Error; err != nil {
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

	if err := r.getDB(ctx).Delete(&Artist{}, "id = ?", artistID).Error; err != nil {
		return log.Err("failed to delete artist", err, "id", id)
	}

	return nil
}

func (r *artistRepository) UpsertBatch(ctx context.Context, artists []*Artist) (int, int, error) {
	log := r.log.Function("UpsertBatch")

	if len(artists) == 0 {
		return 0, 0, nil
	}

	var totalInserted, totalUpdated int

	// Process in batches to avoid memory issues
	for i := 0; i < len(artists); i += ARTIST_BATCH_SIZE {
		end := i + ARTIST_BATCH_SIZE
		if end > len(artists) {
			end = len(artists)
		}

		batch := artists[i:end]
		inserted, updated, err := r.upsertSingleBatch(ctx, batch)
		if err != nil {
			return totalInserted, totalUpdated, log.Err("failed to upsert batch", err, "batchStart", i, "batchEnd", end)
		}

		totalInserted += inserted
		totalUpdated += updated

		log.Info("Processed artist batch", "batchStart", i, "batchEnd", end, "inserted", inserted, "updated", updated)
	}

	return totalInserted, totalUpdated, nil
}

func (r *artistRepository) upsertSingleBatch(ctx context.Context, artists []*Artist) (int, int, error) {
	log := r.log.Function("upsertSingleBatch")

	if len(artists) == 0 {
		return 0, 0, nil
	}

	db := r.getDB(ctx)

	// Use native PostgreSQL UPSERT with ON CONFLICT for single database round-trip
	result := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "discogs_id"}}, // Use unique index on discogs_id
		DoUpdates: clause.AssignmentColumns([]string{
			"name", "biography", "image_url", "is_active", "updated_at",
		}),
	}).CreateInBatches(artists, ARTIST_BATCH_SIZE)

	if result.Error != nil {
		return 0, 0, log.Err("failed to upsert artist batch", result.Error, "count", len(artists))
	}

	affectedRows := int(result.RowsAffected)
	log.Info("Upserted artists", "count", affectedRows)
	return affectedRows, 0, nil
}

func (r *artistRepository) GetBatchByDiscogsIDs(ctx context.Context, discogsIDs []int64) (map[int64]*Artist, error) {
	log := r.log.Function("GetBatchByDiscogsIDs")

	if len(discogsIDs) == 0 {
		return make(map[int64]*Artist), nil
	}

	var artists []*Artist
	if err := r.getDB(ctx).Where("discogs_id IN ?", discogsIDs).Find(&artists).Error; err != nil {
		return nil, log.Err("failed to get artists by Discogs IDs", err, "count", len(discogsIDs))
	}

	// Convert to map for O(1) lookup
	result := make(map[int64]*Artist, len(artists))
	for _, artist := range artists {
		if artist.DiscogsID != nil {
			result[*artist.DiscogsID] = artist
		}
	}

	log.Info("Retrieved artists by Discogs IDs", "requested", len(discogsIDs), "found", len(result))
	return result, nil
}