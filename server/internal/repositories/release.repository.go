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
	RELEASE_BATCH_SIZE = 1000
)

type ReleaseRepository interface {
	GetByID(ctx context.Context, id string) (*Release, error)
	GetByDiscogsID(ctx context.Context, discogsID int64) (*Release, error)
	Create(ctx context.Context, release *Release) (*Release, error)
	Update(ctx context.Context, release *Release) error
	Delete(ctx context.Context, id string) error
	UpsertBatch(ctx context.Context, releases []*Release) (int, int, error)
	GetBatchByDiscogsIDs(ctx context.Context, discogsIDs []int64) (map[int64]*Release, error)
}

type releaseRepository struct {
	db  database.DB
	log logger.Logger
}

func NewReleaseRepository(db database.DB) ReleaseRepository {
	return &releaseRepository{
		db:  db,
		log: logger.New("releaseRepository"),
	}
}

func (r *releaseRepository) getDB(ctx context.Context) *gorm.DB {
	if tx, ok := contextutil.GetTransaction(ctx); ok {
		return tx
	}
	return r.db.SQLWithContext(ctx)
}

func (r *releaseRepository) GetByID(ctx context.Context, id string) (*Release, error) {
	log := r.log.Function("GetByID")

	releaseID, err := uuid.Parse(id)
	if err != nil {
		return nil, log.Err("failed to parse release ID", err, "id", id)
	}

	var release Release
	if err := r.getDB(ctx).Preload("Label").Preload("Master").Preload("Artists").Preload("Genres").First(&release, "id = ?", releaseID).Error; err != nil {
		return nil, log.Err("failed to get release by ID", err, "id", id)
	}

	return &release, nil
}

func (r *releaseRepository) GetByDiscogsID(ctx context.Context, discogsID int64) (*Release, error) {
	log := r.log.Function("GetByDiscogsID")

	var release Release
	if err := r.getDB(ctx).Preload("Label").Preload("Master").Preload("Artists").Preload("Genres").First(&release, "id = ?", discogsID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, log.Err("failed to get release by Discogs ID", err, "discogsID", discogsID)
	}

	return &release, nil
}

func (r *releaseRepository) Create(ctx context.Context, release *Release) (*Release, error) {
	log := r.log.Function("Create")

	if err := r.getDB(ctx).Create(release).Error; err != nil {
		return nil, log.Err("failed to create release", err, "release", release)
	}

	return release, nil
}

func (r *releaseRepository) Update(ctx context.Context, release *Release) error {
	log := r.log.Function("Update")

	if err := r.getDB(ctx).Save(release).Error; err != nil {
		return log.Err("failed to update release", err, "release", release)
	}

	return nil
}

func (r *releaseRepository) Delete(ctx context.Context, id string) error {
	log := r.log.Function("Delete")

	releaseID, err := uuid.Parse(id)
	if err != nil {
		return log.Err("failed to parse release ID", err, "id", id)
	}

	if err := r.getDB(ctx).Delete(&Release{}, "id = ?", releaseID).Error; err != nil {
		return log.Err("failed to delete release", err, "id", id)
	}

	return nil
}

func (r *releaseRepository) UpsertBatch(ctx context.Context, releases []*Release) (int, int, error) {
	log := r.log.Function("UpsertBatch")

	if len(releases) == 0 {
		return 0, 0, nil
	}

	var totalInserted, totalUpdated int

	// Process in batches to avoid memory issues
	for i := 0; i < len(releases); i += RELEASE_BATCH_SIZE {
		end := i + RELEASE_BATCH_SIZE
		if end > len(releases) {
			end = len(releases)
		}

		batch := releases[i:end]
		inserted, updated, err := r.upsertSingleBatch(ctx, batch)
		if err != nil {
			return totalInserted, totalUpdated, log.Err("failed to upsert batch", err, "batchStart", i, "batchEnd", end)
		}

		totalInserted += inserted
		totalUpdated += updated

		log.Info("Processed release batch", "batchStart", i, "batchEnd", end, "inserted", inserted, "updated", updated)
	}

	return totalInserted, totalUpdated, nil
}

func (r *releaseRepository) upsertSingleBatch(ctx context.Context, releases []*Release) (int, int, error) {
	log := r.log.Function("upsertSingleBatch")

	if len(releases) == 0 {
		return 0, 0, nil
	}

	db := r.getDB(ctx)

	// Use native PostgreSQL UPSERT with ON CONFLICT for single database round-trip
	result := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "id"}}, // Use primary key (ID as DiscogsID)
		DoUpdates: clause.AssignmentColumns([]string{
			"title", "year", "country", "format", "image_url", "track_count", "label_id", "master_id", "updated_at",
		}),
	}).CreateInBatches(releases, RELEASE_BATCH_SIZE)

	if result.Error != nil {
		return 0, 0, log.Err("failed to upsert release batch", result.Error, "count", len(releases))
	}

	affectedRows := int(result.RowsAffected)
	log.Info("Upserted releases", "count", affectedRows)
	return affectedRows, 0, nil
}

func (r *releaseRepository) GetBatchByDiscogsIDs(ctx context.Context, discogsIDs []int64) (map[int64]*Release, error) {
	log := r.log.Function("GetBatchByDiscogsIDs")

	if len(discogsIDs) == 0 {
		return make(map[int64]*Release), nil
	}

	var releases []*Release
	if err := r.getDB(ctx).Where("id IN ?", discogsIDs).Find(&releases).Error; err != nil {
		return nil, log.Err("failed to get releases by Discogs IDs", err, "count", len(discogsIDs))
	}

	// Convert to map for O(1) lookup
	result := make(map[int64]*Release, len(releases))
	for _, release := range releases {
		result[int64(release.ID)] = release
	}

	log.Info("Retrieved releases by Discogs IDs", "requested", len(discogsIDs), "found", len(result))
	return result, nil
}