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

type ReleaseRepository interface {
	GetByID(ctx context.Context, id string) (*Release, error)
	GetByDiscogsID(ctx context.Context, discogsID int64) (*Release, error)
	Create(ctx context.Context, release *Release) (*Release, error)
	Update(ctx context.Context, release *Release) error
	Delete(ctx context.Context, id string) error
	UpsertBatch(ctx context.Context, releases []*Release) (int, int, error)
	GetBatchByDiscogsIDs(ctx context.Context, discogsIDs []int64) (map[int64]*Release, error)
	GetHashesByDiscogsIDs(ctx context.Context, discogsIDs []int64) (map[int64]string, error)
	InsertBatch(ctx context.Context, releases []*Release) (int, error)
	UpdateBatch(ctx context.Context, releases []*Release) (int, error)
	// Note: Release associations removed - use Master-level relationships instead
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

func (r *releaseRepository) GetByID(ctx context.Context, id string) (*Release, error) {
	log := r.log.Function("GetByID")

	releaseID, err := uuid.Parse(id)
	if err != nil {
		return nil, log.Err("failed to parse release ID", err, "id", id)
	}

	var release Release
	if err := r.db.SQLWithContext(ctx).Preload("Label").Preload("Master").Preload("Artists").Preload("Genres").First(&release, "id = ?", releaseID).Error; err != nil {
		return nil, log.Err("failed to get release by ID", err, "id", id)
	}

	return &release, nil
}

func (r *releaseRepository) GetByDiscogsID(ctx context.Context, discogsID int64) (*Release, error) {
	log := r.log.Function("GetByDiscogsID")

	var release Release
	if err := r.db.SQLWithContext(ctx).Preload("Label").Preload("Master").Preload("Artists").Preload("Genres").First(&release, "discogs_id = ?", discogsID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, log.Err("failed to get release by Discogs ID", err, "discogsID", discogsID)
	}

	return &release, nil
}

func (r *releaseRepository) Create(ctx context.Context, release *Release) (*Release, error) {
	log := r.log.Function("Create")

	if err := r.db.SQLWithContext(ctx).Create(release).Error; err != nil {
		return nil, log.Err("failed to create release", err, "release", release)
	}

	return release, nil
}

func (r *releaseRepository) Update(ctx context.Context, release *Release) error {
	log := r.log.Function("Update")

	if err := r.db.SQLWithContext(ctx).Save(release).Error; err != nil {
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

	if err := r.db.SQLWithContext(ctx).Delete(&Release{}, "id = ?", releaseID).Error; err != nil {
		return log.Err("failed to delete release", err, "id", id)
	}

	return nil
}

func (r *releaseRepository) UpsertBatch(
	ctx context.Context,
	releases []*Release,
) (int, int, error) {
	log := r.log.Function("UpsertBatch")

	if len(releases) == 0 {
		return 0, 0, nil
	}

	// 1. Extract Discogs IDs from incoming releases
	discogsIDs := make([]int64, len(releases))
	for i, release := range releases {
		discogsIDs[i] = release.DiscogsID
	}

	// 2. Get existing hashes for these Discogs IDs
	existingHashes, err := r.GetHashesByDiscogsIDs(ctx, discogsIDs)
	if err != nil {
		return 0, 0, log.Err("failed to get existing hashes", err, "count", len(discogsIDs))
	}

	// 3. Convert releases to DiscogsHashable interface
	hashableRecords := make([]utils.DiscogsHashable, len(releases))
	for i, release := range releases {
		hashableRecords[i] = release
	}

	// 4. Categorize records by hash comparison
	categories := utils.CategorizeRecordsByHash(hashableRecords, existingHashes)

	var insertedCount, updatedCount int

	// 5. Execute insert batch for new records
	if len(categories.Insert) > 0 {
		insertReleases := make([]*Release, len(categories.Insert))
		for i, record := range categories.Insert {
			insertReleases[i] = record.(*Release)
		}
		insertedCount, err = r.InsertBatch(ctx, insertReleases)
		if err != nil {
			return 0, 0, log.Err(
				"failed to insert release batch",
				err,
				"count",
				len(insertReleases),
			)
		}
	}

	// 6. Execute update batch for changed records
	if len(categories.Update) > 0 {
		updateReleases := make([]*Release, len(categories.Update))
		for i, record := range categories.Update {
			updateReleases[i] = record.(*Release)
		}
		updatedCount, err = r.UpdateBatch(ctx, updateReleases)
		if err != nil {
			return insertedCount, 0, log.Err(
				"failed to update release batch",
				err,
				"count",
				len(updateReleases),
			)
		}
	}

	log.Info("Hash-based upsert completed",
		"total", len(releases),
		"inserted", insertedCount,
		"updated", updatedCount,
		"skipped", len(categories.Skip))

	return insertedCount, updatedCount, nil
}

func (r *releaseRepository) GetBatchByDiscogsIDs(
	ctx context.Context,
	discogsIDs []int64,
) (map[int64]*Release, error) {
	log := r.log.Function("GetBatchByDiscogsIDs")

	if len(discogsIDs) == 0 {
		return make(map[int64]*Release), nil
	}

	var releases []*Release
	if err := r.db.SQLWithContext(ctx).Where("discogs_id IN ?", discogsIDs).Find(&releases).Error; err != nil {
		return nil, log.Err("failed to get releases by Discogs IDs", err, "count", len(discogsIDs))
	}

	// Convert to map for O(1) lookup
	result := make(map[int64]*Release, len(releases))
	for _, release := range releases {
		result[release.DiscogsID] = release
	}

	log.Info(
		"Retrieved releases by Discogs IDs",
		"requested",
		len(discogsIDs),
		"found",
		len(result),
	)
	return result, nil
}

func (r *releaseRepository) GetHashesByDiscogsIDs(
	ctx context.Context,
	discogsIDs []int64,
) (map[int64]string, error) {
	log := r.log.Function("GetHashesByDiscogsIDs")

	if len(discogsIDs) == 0 {
		return make(map[int64]string), nil
	}

	var releases []struct {
		DiscogsID   int64  `json:"discogsId"`
		ContentHash string `json:"contentHash"`
	}

	if err := r.db.SQLWithContext(ctx).
		Model(&Release{}).
		Select("discogs_id, content_hash").
		Where("discogs_id IN ?", discogsIDs).
		Find(&releases).Error; err != nil {
		return nil, log.Err(
			"failed to get release hashes by Discogs IDs",
			err,
			"count",
			len(discogsIDs),
		)
	}

	result := make(map[int64]string, len(releases))
	for _, release := range releases {
		result[release.DiscogsID] = release.ContentHash
	}

	log.Info(
		"Retrieved release hashes by Discogs IDs",
		"requested",
		len(discogsIDs),
		"found",
		len(result),
	)
	return result, nil
}

func (r *releaseRepository) InsertBatch(ctx context.Context, releases []*Release) (int, error) {
	log := r.log.Function("InsertBatch")

	if len(releases) == 0 {
		return 0, nil
	}

	if err := r.db.SQLWithContext(ctx).Create(&releases).Error; err != nil {
		return 0, log.Err("failed to insert release batch", err, "count", len(releases))
	}

	log.Info("Inserted releases", "count", len(releases))
	return len(releases), nil
}

func (r *releaseRepository) UpdateBatch(ctx context.Context, releases []*Release) (int, error) {
	log := r.log.Function("UpdateBatch")

	if len(releases) == 0 {
		return 0, nil
	}

	updatedCount := 0
	for _, release := range releases {
		// Get existing record first to ensure we have the complete model
		existingRelease := &Release{}
		err := r.db.SQLWithContext(ctx).
			Where("discogs_id = ?", release.DiscogsID).
			First(existingRelease).
			Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				// Skip if record doesn't exist (should not happen in our flow)
				log.Warn("Release not found for update", "discogsID", release.DiscogsID)
				continue
			}
			return updatedCount, log.Err(
				"failed to get existing release",
				err,
				"discogsID",
				release.DiscogsID,
			)
		}

		// Update only the specific fields we want to change
		existingRelease.Title = release.Title
		existingRelease.Year = release.Year
		existingRelease.Country = release.Country
		existingRelease.Format = release.Format
		existingRelease.ImageURL = release.ImageURL
		existingRelease.TrackCount = release.TrackCount
		existingRelease.LabelID = release.LabelID
		existingRelease.MasterID = release.MasterID
		existingRelease.ContentHash = release.ContentHash
		existingRelease.TracksJSON = release.TracksJSON
		existingRelease.ArtistsJSON = release.ArtistsJSON
		existingRelease.GenresJSON = release.GenresJSON

		// Use Save() which handles all GORM hooks properly
		result := r.db.SQLWithContext(ctx).Save(existingRelease)
		if result.Error != nil {
			return updatedCount, log.Err(
				"failed to save release",
				result.Error,
				"discogsID",
				release.DiscogsID,
			)
		}

		if result.RowsAffected > 0 {
			updatedCount++
		}
	}

	log.Info("Updated releases", "count", updatedCount)
	return updatedCount, nil
}
