package repositories

import (
	"context"
	"waugzee/internal/logger"
	. "waugzee/internal/models"
	"waugzee/internal/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ReleaseRepository interface {
	GetByID(ctx context.Context, tx *gorm.DB, id string) (*Release, error)
	GetByDiscogsID(ctx context.Context, tx *gorm.DB, discogsID int64) (*Release, error)
	Create(ctx context.Context, tx *gorm.DB, release *Release) (*Release, error)
	Update(ctx context.Context, tx *gorm.DB, release *Release) error
	Delete(ctx context.Context, tx *gorm.DB, id string) error
	UpsertBatch(ctx context.Context, tx *gorm.DB, releases []*Release) error
	GetBatchByDiscogsIDs(ctx context.Context, tx *gorm.DB, discogsIDs []int64) (map[int64]*Release, error)
	GetHashesByDiscogsIDs(ctx context.Context, tx *gorm.DB, discogsIDs []int64) (map[int64]string, error)
	InsertBatch(ctx context.Context, tx *gorm.DB, releases []*Release) error
	UpdateBatch(ctx context.Context, tx *gorm.DB, releases []*Release) error
	// Note: Release associations removed - use Master-level relationships instead
}

type releaseRepository struct {
	log logger.Logger
}

func NewReleaseRepository() ReleaseRepository {
	return &releaseRepository{
		log: logger.New("releaseRepository"),
	}
}

func (r *releaseRepository) GetByID(ctx context.Context, tx *gorm.DB, id string) (*Release, error) {
	log := r.log.Function("GetByID")

	releaseID, err := uuid.Parse(id)
	if err != nil {
		return nil, log.Err("failed to parse release ID", err, "id", id)
	}

	var release Release
	if err := tx.WithContext(ctx).Preload("Label").Preload("Master").Preload("Artists").Preload("Genres").First(&release, "id = ?", releaseID).Error; err != nil {
		return nil, log.Err("failed to get release by ID", err, "id", id)
	}

	return &release, nil
}

func (r *releaseRepository) GetByDiscogsID(ctx context.Context, tx *gorm.DB, discogsID int64) (*Release, error) {
	log := r.log.Function("GetByDiscogsID")

	var release Release
	if err := tx.WithContext(ctx).Preload("Label").Preload("Master").Preload("Artists").Preload("Genres").First(&release, "id = ?", discogsID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, log.Err("failed to get release by Discogs ID", err, "discogsID", discogsID)
	}

	return &release, nil
}

func (r *releaseRepository) Create(ctx context.Context, tx *gorm.DB, release *Release) (*Release, error) {
	log := r.log.Function("Create")

	if err := tx.WithContext(ctx).Create(release).Error; err != nil {
		return nil, log.Err("failed to create release", err, "release", release)
	}

	return release, nil
}

func (r *releaseRepository) Update(ctx context.Context, tx *gorm.DB, release *Release) error {
	log := r.log.Function("Update")

	if err := tx.WithContext(ctx).Save(release).Error; err != nil {
		return log.Err("failed to update release", err, "release", release)
	}

	return nil
}

func (r *releaseRepository) Delete(ctx context.Context, tx *gorm.DB, id string) error {
	log := r.log.Function("Delete")

	releaseID, err := uuid.Parse(id)
	if err != nil {
		return log.Err("failed to parse release ID", err, "id", id)
	}

	if err := tx.WithContext(ctx).Delete(&Release{}, "id = ?", releaseID).Error; err != nil {
		return log.Err("failed to delete release", err, "id", id)
	}

	return nil
}

func (r *releaseRepository) UpsertBatch(
	ctx context.Context,
	tx *gorm.DB,
	releases []*Release,
) error {
	log := r.log.Function("UpsertBatch")

	if len(releases) == 0 {
		return nil
	}

	discogsIDs := make([]int64, len(releases))
	for i, release := range releases {
		discogsIDs[i] = release.ID
	}

	existingHashes, err := r.GetHashesByDiscogsIDs(ctx, tx, discogsIDs)
	if err != nil {
		return log.Err("failed to get existing hashes", err, "count", len(discogsIDs))
	}

	hashableRecords := make([]utils.DiscogsHashable, len(releases))
	for i, release := range releases {
		hashableRecords[i] = release
	}

	categories := utils.CategorizeRecordsByHash(hashableRecords, existingHashes)

	if len(categories.Insert) > 0 {
		insertReleases := make([]*Release, len(categories.Insert))
		for i, record := range categories.Insert {
			insertReleases[i] = record.(*Release)
		}
		err = r.InsertBatch(ctx, tx, insertReleases)
		if err != nil {
			return log.Err(
				"failed to insert release batch",
				err,
				"count",
				len(insertReleases),
			)
		}
	}

	if len(categories.Update) > 0 {
		updateReleases := make([]*Release, len(categories.Update))
		for i, record := range categories.Update {
			updateReleases[i] = record.(*Release)
		}
		err = r.UpdateBatch(ctx, tx, updateReleases)
		if err != nil {
			return log.Err(
				"failed to update release batch",
				err,
				"count",
				len(updateReleases),
			)
		}
	}

	return nil
}

func (r *releaseRepository) GetBatchByDiscogsIDs(
	ctx context.Context,
	tx *gorm.DB,
	discogsIDs []int64,
) (map[int64]*Release, error) {
	log := r.log.Function("GetBatchByDiscogsIDs")

	if len(discogsIDs) == 0 {
		return make(map[int64]*Release), nil
	}

	var releases []*Release
	if err := tx.WithContext(ctx).Where("id IN ?", discogsIDs).Find(&releases).Error; err != nil {
		return nil, log.Err("failed to get releases by Discogs IDs", err, "count", len(discogsIDs))
	}

	// Convert to map for O(1) lookup
	result := make(map[int64]*Release, len(releases))
	for _, release := range releases {
		result[release.ID] = release
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
	tx *gorm.DB,
	discogsIDs []int64,
) (map[int64]string, error) {
	log := r.log.Function("GetHashesByDiscogsIDs")

	if len(discogsIDs) == 0 {
		return make(map[int64]string), nil
	}

	var releases []struct {
		ID   int64  `json:"discogsId"`
		ContentHash string `json:"contentHash"`
	}

	if err := tx.WithContext(ctx).
		Model(&Release{}).
		Select("id, content_hash").
		Where("id IN ?", discogsIDs).
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
		result[release.ID] = release.ContentHash
	}

	return result, nil
}

func (r *releaseRepository) InsertBatch(ctx context.Context, tx *gorm.DB, releases []*Release) error {
	log := r.log.Function("InsertBatch")

	if len(releases) == 0 {
		return nil
	}

	if err := tx.WithContext(ctx).Create(&releases).Error; err != nil {
		return log.Err("failed to insert release batch", err, "count", len(releases))
	}

	return nil
}

func (r *releaseRepository) UpdateBatch(ctx context.Context, tx *gorm.DB, releases []*Release) error {
	log := r.log.Function("UpdateBatch")

	if len(releases) == 0 {
		return nil
	}

	if err := tx.WithContext(ctx).Save(&releases).Error; err != nil {
		return log.Err("failed to update release batch", err, "count", len(releases))
	}

	return nil
}
