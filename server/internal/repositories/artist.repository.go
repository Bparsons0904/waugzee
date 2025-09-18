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
	UpsertBatch(ctx context.Context, artists []*Artist) (int, int, error)
	GetBatchByDiscogsIDs(ctx context.Context, discogsIDs []int64) (map[int64]*Artist, error)
	GetHashesByDiscogsIDs(ctx context.Context, discogsIDs []int64) (map[int64]string, error)
	InsertBatch(ctx context.Context, artists []*Artist) (int, error)
	UpdateBatch(ctx context.Context, artists []*Artist) (int, error)
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
	if err := r.db.SQLWithContext(ctx).First(&artist, "discogs_id = ?", discogsID).Error; err != nil {
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

func (r *artistRepository) UpsertBatch(ctx context.Context, artists []*Artist) (int, int, error) {
	log := r.log.Function("UpsertBatch")

	if len(artists) == 0 {
		return 0, 0, nil
	}

	// 1. Extract Discogs IDs from incoming artists
	discogsIDs := make([]int64, len(artists))
	for i, artist := range artists {
		discogsIDs[i] = artist.DiscogsID
	}

	// 2. Get existing hashes for these Discogs IDs
	existingHashes, err := r.GetHashesByDiscogsIDs(ctx, discogsIDs)
	if err != nil {
		return 0, 0, log.Err("failed to get existing hashes", err, "count", len(discogsIDs))
	}

	// 3. Convert artists to DiscogsHashable interface
	hashableRecords := make([]utils.DiscogsHashable, len(artists))
	for i, artist := range artists {
		hashableRecords[i] = artist
	}

	// 4. Categorize records by hash comparison
	categories := utils.CategorizeRecordsByHash(hashableRecords, existingHashes)

	var insertedCount, updatedCount int

	// 5. Execute insert batch for new records
	if len(categories.Insert) > 0 {
		insertArtists := make([]*Artist, len(categories.Insert))
		for i, record := range categories.Insert {
			insertArtists[i] = record.(*Artist)
		}
		insertedCount, err = r.InsertBatch(ctx, insertArtists)
		if err != nil {
			return 0, 0, log.Err("failed to insert artist batch", err, "count", len(insertArtists))
		}
	}

	// 6. Execute update batch for changed records
	if len(categories.Update) > 0 {
		updateArtists := make([]*Artist, len(categories.Update))
		for i, record := range categories.Update {
			updateArtists[i] = record.(*Artist)
		}
		updatedCount, err = r.UpdateBatch(ctx, updateArtists)
		if err != nil {
			return insertedCount, 0, log.Err("failed to update artist batch", err, "count", len(updateArtists))
		}
	}

	log.Info("Hash-based upsert completed",
		"total", len(artists),
		"inserted", insertedCount,
		"updated", updatedCount,
		"skipped", len(categories.Skip))

	return insertedCount, updatedCount, nil
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
	if err := r.db.SQLWithContext(ctx).Where("discogs_id IN ?", discogsIDs).Find(&artists).Error; err != nil {
		return nil, log.Err("failed to get artists by Discogs IDs", err, "count", len(discogsIDs))
	}

	// Convert to map for O(1) lookup
	result := make(map[int64]*Artist, len(artists))
	for _, artist := range artists {
		result[artist.DiscogsID] = artist
	}

	log.Info("Retrieved artists by Discogs IDs", "requested", len(discogsIDs), "found", len(result))
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
		DiscogsID   int64  `json:"discogsId"`
		ContentHash string `json:"contentHash"`
	}

	if err := r.db.SQLWithContext(ctx).
		Model(&Artist{}).
		Select("discogs_id, content_hash").
		Where("discogs_id IN ?", discogsIDs).
		Find(&artists).Error; err != nil {
		return nil, log.Err("failed to get artist hashes by Discogs IDs", err, "count", len(discogsIDs))
	}

	result := make(map[int64]string, len(artists))
	for _, artist := range artists {
		result[artist.DiscogsID] = artist.ContentHash
	}

	log.Info("Retrieved artist hashes by Discogs IDs", "requested", len(discogsIDs), "found", len(result))
	return result, nil
}

func (r *artistRepository) InsertBatch(ctx context.Context, artists []*Artist) (int, error) {
	log := r.log.Function("InsertBatch")

	if len(artists) == 0 {
		return 0, nil
	}

	if err := r.db.SQLWithContext(ctx).Create(&artists).Error; err != nil {
		return 0, log.Err("failed to insert artist batch", err, "count", len(artists))
	}

	log.Info("Inserted artists", "count", len(artists))
	return len(artists), nil
}

func (r *artistRepository) UpdateBatch(ctx context.Context, artists []*Artist) (int, error) {
	log := r.log.Function("UpdateBatch")

	if len(artists) == 0 {
		return 0, nil
	}

	updatedCount := 0
	for _, artist := range artists {
		// Get existing record first to ensure we have the complete model
		existingArtist := &Artist{}
		err := r.db.SQLWithContext(ctx).Where("discogs_id = ?", artist.DiscogsID).First(existingArtist).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				// Skip if record doesn't exist (should not happen in our flow)
				log.Warn("Artist not found for update", "discogsID", artist.DiscogsID)
				continue
			}
			return updatedCount, log.Err("failed to get existing artist", err, "discogsID", artist.DiscogsID)
		}

		// Update only the specific fields we want to change
		existingArtist.Name = artist.Name
		existingArtist.IsActive = artist.IsActive
		existingArtist.ContentHash = artist.ContentHash

		// Use Save() which handles all GORM hooks properly
		result := r.db.SQLWithContext(ctx).Save(existingArtist)
		if result.Error != nil {
			return updatedCount, log.Err("failed to save artist", result.Error, "discogsID", artist.DiscogsID)
		}

		if result.RowsAffected > 0 {
			updatedCount++
		}
	}

	log.Info("Updated artists", "count", updatedCount)
	return updatedCount, nil
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
		DiscogsID: discogsID,
		Name:      name,
		IsActive:  true, // Default to active
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

	log.Info("Created new artist", "name", name, "discogsID", createdArtist.DiscogsID)
	return createdArtist, nil
}

