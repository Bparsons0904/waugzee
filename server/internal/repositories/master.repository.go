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
	MASTER_BATCH_SIZE = 5000
)

type MasterRepository interface {
	GetByID(ctx context.Context, id string) (*Master, error)
	GetByDiscogsID(ctx context.Context, discogsID int64) (*Master, error)
	Create(ctx context.Context, master *Master) (*Master, error)
	Update(ctx context.Context, master *Master) error
	Delete(ctx context.Context, id string) error
	UpsertBatch(ctx context.Context, masters []*Master) (int, int, error)
	GetBatchByDiscogsIDs(ctx context.Context, discogsIDs []int64) (map[int64]*Master, error)
	// Association methods
	CreateMasterArtistAssociations(ctx context.Context, masterDiscogsIDs []int64, artistDiscogsIDs []int64) error
	CreateMasterGenreAssociations(ctx context.Context, masterDiscogsIDs []int64, genreNames []string) error
}

type masterRepository struct {
	db  database.DB
	log logger.Logger
}

func NewMasterRepository(db database.DB) MasterRepository {
	return &masterRepository{
		db:  db,
		log: logger.New("masterRepository"),
	}
}

func (r *masterRepository) getDB(ctx context.Context) *gorm.DB {
	if tx, ok := contextutil.GetTransaction(ctx); ok {
		return tx
	}
	return r.db.SQLWithContext(ctx)
}

func (r *masterRepository) GetByID(ctx context.Context, id string) (*Master, error) {
	log := r.log.Function("GetByID")

	masterID, err := uuid.Parse(id)
	if err != nil {
		return nil, log.Err("failed to parse master ID", err, "id", id)
	}

	var master Master
	if err := r.getDB(ctx).First(&master, "id = ?", masterID).Error; err != nil {
		return nil, log.Err("failed to get master by ID", err, "id", id)
	}

	return &master, nil
}

func (r *masterRepository) GetByDiscogsID(ctx context.Context, discogsID int64) (*Master, error) {
	log := r.log.Function("GetByDiscogsID")

	var master Master
	if err := r.getDB(ctx).First(&master, "discogs_id = ?", discogsID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, log.Err("failed to get master by Discogs ID", err, "discogsID", discogsID)
	}

	return &master, nil
}

func (r *masterRepository) Create(ctx context.Context, master *Master) (*Master, error) {
	log := r.log.Function("Create")

	if err := r.getDB(ctx).Create(master).Error; err != nil {
		return nil, log.Err("failed to create master", err, "master", master)
	}

	return master, nil
}

func (r *masterRepository) Update(ctx context.Context, master *Master) error {
	log := r.log.Function("Update")

	if err := r.getDB(ctx).Save(master).Error; err != nil {
		return log.Err("failed to update master", err, "master", master)
	}

	return nil
}

func (r *masterRepository) Delete(ctx context.Context, id string) error {
	log := r.log.Function("Delete")

	masterID, err := uuid.Parse(id)
	if err != nil {
		return log.Err("failed to parse master ID", err, "id", id)
	}

	if err := r.getDB(ctx).Delete(&Master{}, "id = ?", masterID).Error; err != nil {
		return log.Err("failed to delete master", err, "id", id)
	}

	return nil
}

func (r *masterRepository) UpsertBatch(ctx context.Context, masters []*Master) (int, int, error) {
	if len(masters) == 0 {
		return 0, 0, nil
	}

	// Service has already deduplicated - process directly without re-deduplication
	return r.upsertSingleBatch(ctx, masters)
}

func (r *masterRepository) upsertSingleBatch(ctx context.Context, masters []*Master) (int, int, error) {
	log := r.log.Function("upsertSingleBatch")

	if len(masters) == 0 {
		return 0, 0, nil
	}

	db := r.getDB(ctx)

	// Use native PostgreSQL UPSERT with ON CONFLICT for single database round-trip
	result := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "discogs_id"}}, // Use primary key (DiscogsID)
		DoUpdates: clause.AssignmentColumns([]string{
			"title", "main_release", "year", "updated_at",
		}),
	}).CreateInBatches(masters, MASTER_BATCH_SIZE)

	if result.Error != nil {
		return 0, 0, log.Err("failed to upsert master batch", result.Error, "count", len(masters))
	}

	affectedRows := int(result.RowsAffected)
	log.Info("Upserted masters", "count", affectedRows)
	return affectedRows, 0, nil
}

func (r *masterRepository) GetBatchByDiscogsIDs(ctx context.Context, discogsIDs []int64) (map[int64]*Master, error) {
	log := r.log.Function("GetBatchByDiscogsIDs")

	if len(discogsIDs) == 0 {
		return make(map[int64]*Master), nil
	}

	var masters []*Master
	if err := r.getDB(ctx).Where("discogs_id IN ?", discogsIDs).Find(&masters).Error; err != nil {
		return nil, log.Err("failed to get masters by Discogs IDs", err, "count", len(discogsIDs))
	}

	// Convert to map for O(1) lookup
	result := make(map[int64]*Master, len(masters))
	for _, master := range masters {
		result[master.DiscogsID] = master
	}

	log.Info("Retrieved masters by Discogs IDs", "requested", len(discogsIDs), "found", len(result))
	return result, nil
}

// CreateMasterArtistAssociations creates many-to-many associations between masters and artists
func (r *masterRepository) CreateMasterArtistAssociations(ctx context.Context, masterDiscogsIDs []int64, artistDiscogsIDs []int64) error {
	log := r.log.Function("CreateMasterArtistAssociations")

	if len(masterDiscogsIDs) == 0 || len(artistDiscogsIDs) == 0 {
		return nil
	}

	db := r.getDB(ctx)

	// Build cross-product associations with ON CONFLICT DO NOTHING for idempotency
	query := `
		INSERT INTO master_artists (master_discogs_id, artist_discogs_id)
		SELECT m.discogs_id, a.discogs_id
		FROM unnest($1::bigint[]) AS m(discogs_id)
		CROSS JOIN unnest($2::bigint[]) AS a(discogs_id)
		ON CONFLICT (master_discogs_id, artist_discogs_id) DO NOTHING
	`

	result := db.Exec(query, masterDiscogsIDs, artistDiscogsIDs)
	if result.Error != nil {
		return log.Err("failed to create master-artist associations", result.Error,
			"masterCount", len(masterDiscogsIDs), "artistCount", len(artistDiscogsIDs))
	}

	log.Info("Created master-artist associations",
		"masterCount", len(masterDiscogsIDs),
		"artistCount", len(artistDiscogsIDs),
		"associationsCreated", result.RowsAffected)

	return nil
}

// CreateMasterGenreAssociations creates many-to-many associations between masters and genres
func (r *masterRepository) CreateMasterGenreAssociations(ctx context.Context, masterDiscogsIDs []int64, genreNames []string) error {
	log := r.log.Function("CreateMasterGenreAssociations")

	if len(masterDiscogsIDs) == 0 || len(genreNames) == 0 {
		return nil
	}

	db := r.getDB(ctx)

	// Build cross-product associations using genre names with ON CONFLICT DO NOTHING
	query := `
		INSERT INTO master_genres (master_discogs_id, genre_id)
		SELECT m.discogs_id, g.id
		FROM unnest($1::bigint[]) AS m(discogs_id)
		CROSS JOIN genres g
		WHERE g.name = ANY($2::text[])
		ON CONFLICT (master_discogs_id, genre_id) DO NOTHING
	`

	result := db.Exec(query, masterDiscogsIDs, genreNames)
	if result.Error != nil {
		return log.Err("failed to create master-genre associations", result.Error,
			"masterCount", len(masterDiscogsIDs), "genreCount", len(genreNames))
	}

	log.Info("Created master-genre associations",
		"masterCount", len(masterDiscogsIDs),
		"genreCount", len(genreNames),
		"associationsCreated", result.RowsAffected)

	return nil
}