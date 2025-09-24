package repositories

import (
	"context"
	"waugzee/internal/logger"
	. "waugzee/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ArtistRepository interface {
	GetByID(ctx context.Context, tx *gorm.DB, id string) (*Artist, error)
	GetByDiscogsID(ctx context.Context, tx *gorm.DB, discogsID int64) (*Artist, error)
	Create(ctx context.Context, tx *gorm.DB, artist *Artist) (*Artist, error)
	Update(ctx context.Context, tx *gorm.DB, artist *Artist) error
	Delete(ctx context.Context, tx *gorm.DB, id string) error
	UpsertBatch(ctx context.Context, tx *gorm.DB, artists []*Artist) error
	GetBatchByDiscogsIDs(
		ctx context.Context,
		tx *gorm.DB,
		discogsIDs []int64,
	) (map[int64]*Artist, error)
	InsertBatch(ctx context.Context, tx *gorm.DB, artists []*Artist) error
	UpdateBatch(ctx context.Context, tx *gorm.DB, artists []*Artist) error
}

type artistRepository struct {
	log logger.Logger
}

func NewArtistRepository() ArtistRepository {
	return &artistRepository{
		log: logger.New("artistRepository"),
	}
}

func (r *artistRepository) GetByID(ctx context.Context, tx *gorm.DB, id string) (*Artist, error) {
	log := r.log.Function("GetByID")

	artistID, err := uuid.Parse(id)
	if err != nil {
		return nil, log.Err("failed to parse artist ID", err, "id", id)
	}

	var artist Artist
	if err := tx.WithContext(ctx).First(&artist, "id = ?", artistID).Error; err != nil {
		return nil, log.Err("failed to get artist by ID", err, "id", id)
	}

	return &artist, nil
}

func (r *artistRepository) GetByDiscogsID(
	ctx context.Context,
	tx *gorm.DB,
	discogsID int64,
) (*Artist, error) {
	log := r.log.Function("GetByDiscogsID")

	var artist Artist
	if err := tx.WithContext(ctx).First(&artist, "id = ?", discogsID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, log.Err("failed to get artist by Discogs ID", err, "discogsID", discogsID)
	}

	return &artist, nil
}

func (r *artistRepository) Create(
	ctx context.Context,
	tx *gorm.DB,
	artist *Artist,
) (*Artist, error) {
	log := r.log.Function("Create")

	if err := tx.WithContext(ctx).Create(artist).Error; err != nil {
		return nil, log.Err("failed to create artist", err, "artist", artist)
	}

	return artist, nil
}

func (r *artistRepository) Update(ctx context.Context, tx *gorm.DB, artist *Artist) error {
	log := r.log.Function("Update")

	if err := tx.WithContext(ctx).Save(artist).Error; err != nil {
		return log.Err("failed to update artist", err, "artist", artist)
	}

	return nil
}

func (r *artistRepository) Delete(ctx context.Context, tx *gorm.DB, id string) error {
	log := r.log.Function("Delete")

	artistID, err := uuid.Parse(id)
	if err != nil {
		return log.Err("failed to parse artist ID", err, "id", id)
	}

	if err := tx.WithContext(ctx).Delete(&Artist{}, "id = ?", artistID).Error; err != nil {
		return log.Err("failed to delete artist", err, "id", id)
	}

	return nil
}

func (r *artistRepository) UpsertBatch(ctx context.Context, tx *gorm.DB, artists []*Artist) error {
	log := r.log.Function("UpsertBatch")

	if len(artists) == 0 {
		return nil
	}

	log.Info("Upserting artists", "count", len(artists))

	if err := tx.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"name", "profile", "updated_at", "resource_url", "uri", "releases_url"}),
	}).Create(&artists).Error; err != nil {
		return log.Err("failed to upsert artist batch", err, "count", len(artists))
	}

	log.Info("Successfully upserted artists", "count", len(artists))
	return nil
}

func (r *artistRepository) GetBatchByDiscogsIDs(
	ctx context.Context,
	tx *gorm.DB,
	discogsIDs []int64,
) (map[int64]*Artist, error) {
	log := r.log.Function("GetBatchByDiscogsIDs")

	if len(discogsIDs) == 0 {
		return make(map[int64]*Artist), nil
	}

	var artists []*Artist
	if err := tx.WithContext(ctx).Where("id IN ?", discogsIDs).Find(&artists).Error; err != nil {
		return nil, log.Err("failed to get artists by Discogs IDs", err, "count", len(discogsIDs))
	}

	// Convert to map for O(1) lookup
	result := make(map[int64]*Artist, len(artists))
	for _, artist := range artists {
		result[artist.ID] = artist
	}

	return result, nil
}

func (r *artistRepository) InsertBatch(ctx context.Context, tx *gorm.DB, artists []*Artist) error {
	log := r.log.Function("InsertBatch")

	if len(artists) == 0 {
		return nil
	}

	if err := tx.WithContext(ctx).Create(&artists).Error; err != nil {
		return log.Err("failed to insert artist batch", err, "count", len(artists))
	}

	return nil
}

func (r *artistRepository) UpdateBatch(ctx context.Context, tx *gorm.DB, artists []*Artist) error {
	log := r.log.Function("UpdateBatch")

	if len(artists) == 0 {
		return nil
	}

	if err := tx.WithContext(ctx).Save(&artists).Error; err != nil {
		return log.Err("failed to update artist batch", err, "count", len(artists))
	}

	return nil
}
