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

type LabelRepository interface {
	GetByID(ctx context.Context, id string) (*Label, error)
	GetByDiscogsID(ctx context.Context, discogsID int64) (*Label, error)
	Create(ctx context.Context, label *Label) (*Label, error)
	Update(ctx context.Context, label *Label) error
	Delete(ctx context.Context, id string) error
	UpsertBatch(ctx context.Context, labels []*Label) error
	GetBatchByDiscogsIDs(ctx context.Context, discogsIDs []int64) (map[int64]*Label, error)
	GetHashesByDiscogsIDs(ctx context.Context, discogsIDs []int64) (map[int64]string, error)
	InsertBatch(ctx context.Context, labels []*Label) error
	UpdateBatch(ctx context.Context, labels []*Label) error
}

type labelRepository struct {
	db  database.DB
	log logger.Logger
}

func NewLabelRepository(db database.DB) LabelRepository {
	return &labelRepository{
		db:  db,
		log: logger.New("labelRepository"),
	}
}

func (r *labelRepository) GetByID(ctx context.Context, id string) (*Label, error) {
	log := r.log.Function("GetByID")

	labelID, err := uuid.Parse(id)
	if err != nil {
		return nil, log.Err("failed to parse label ID", err, "id", id)
	}

	var label Label
	if err := r.db.SQLWithContext(ctx).First(&label, "id = ?", labelID).Error; err != nil {
		return nil, log.Err("failed to get label by ID", err, "id", id)
	}

	return &label, nil
}

func (r *labelRepository) GetByDiscogsID(ctx context.Context, discogsID int64) (*Label, error) {
	log := r.log.Function("GetByDiscogsID")

	var label Label
	if err := r.db.SQLWithContext(ctx).First(&label, "id = ?", discogsID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, log.Err("failed to get label by Discogs ID", err, "discogsID", discogsID)
	}

	return &label, nil
}

func (r *labelRepository) Create(ctx context.Context, label *Label) (*Label, error) {
	log := r.log.Function("Create")

	if err := r.db.SQLWithContext(ctx).Create(label).Error; err != nil {
		return nil, log.Err("failed to create label", err, "label", label)
	}

	return label, nil
}

func (r *labelRepository) Update(ctx context.Context, label *Label) error {
	log := r.log.Function("Update")

	if err := r.db.SQLWithContext(ctx).Save(label).Error; err != nil {
		return log.Err("failed to update label", err, "label", label)
	}

	return nil
}

func (r *labelRepository) Delete(ctx context.Context, id string) error {
	log := r.log.Function("Delete")

	labelID, err := uuid.Parse(id)
	if err != nil {
		return log.Err("failed to parse label ID", err, "id", id)
	}

	if err := r.db.SQLWithContext(ctx).Delete(&Label{}, "id = ?", labelID).Error; err != nil {
		return log.Err("failed to delete label", err, "id", id)
	}

	return nil
}

func (r *labelRepository) UpsertBatch(ctx context.Context, labels []*Label) error {
	log := r.log.Function("UpsertBatch")

	if len(labels) == 0 {
		return nil
	}

	discogsIDs := make([]int64, len(labels))
	for i, label := range labels {
		discogsIDs[i] = label.ID
	}

	existingHashes, err := r.GetHashesByDiscogsIDs(ctx, discogsIDs)
	if err != nil {
		return log.Err("failed to get existing hashes", err, "count", len(discogsIDs))
	}

	hashableRecords := make([]utils.DiscogsHashable, len(labels))
	for i, label := range labels {
		hashableRecords[i] = label
	}

	categories := utils.CategorizeRecordsByHash(hashableRecords, existingHashes)
	if len(categories.Insert) > 0 {
		insertLabels := make([]*Label, len(categories.Insert))
		for i, record := range categories.Insert {
			insertLabels[i] = record.(*Label)
		}
		err = r.InsertBatch(ctx, insertLabels)
		if err != nil {
			return log.Err("failed to insert label batch", err, "count", len(insertLabels))
		}
	}

	if len(categories.Update) > 0 {
		updateLabels := make([]*Label, len(categories.Update))
		for i, record := range categories.Update {
			updateLabels[i] = record.(*Label)
		}
		err = r.UpdateBatch(ctx, updateLabels)
		if err != nil {
			return log.Err(
				"failed to update label batch",
				err,
				"count",
				len(updateLabels),
			)
		}
	}

	return nil
}

func (r *labelRepository) GetBatchByDiscogsIDs(
	ctx context.Context,
	discogsIDs []int64,
) (map[int64]*Label, error) {
	log := r.log.Function("GetBatchByDiscogsIDs")

	if len(discogsIDs) == 0 {
		return make(map[int64]*Label), nil
	}

	var labels []*Label
	if err := r.db.SQLWithContext(ctx).Where("id IN ?", discogsIDs).Find(&labels).Error; err != nil {
		return nil, log.Err("failed to get labels by Discogs IDs", err, "count", len(discogsIDs))
	}

	// Convert to map for O(1) lookup
	result := make(map[int64]*Label, len(labels))
	for _, label := range labels {
		result[label.ID] = label
	}

	log.Info("Retrieved labels by Discogs IDs", "requested", len(discogsIDs), "found", len(result))
	return result, nil
}

func (r *labelRepository) GetHashesByDiscogsIDs(
	ctx context.Context,
	discogsIDs []int64,
) (map[int64]string, error) {
	log := r.log.Function("GetHashesByDiscogsIDs")

	if len(discogsIDs) == 0 {
		return make(map[int64]string), nil
	}

	var labels []struct {
		ID   int64  `json:"discogsId"`
		ContentHash string `json:"contentHash"`
	}

	if err := r.db.SQLWithContext(ctx).
		Model(&Label{}).
		Select("id, content_hash").
		Where("id IN ?", discogsIDs).
		Find(&labels).Error; err != nil {
		return nil, log.Err(
			"failed to get label hashes by Discogs IDs",
			err,
			"count",
			len(discogsIDs),
		)
	}

	result := make(map[int64]string, len(labels))
	for _, label := range labels {
		result[label.ID] = label.ContentHash
	}

	return result, nil
}

func (r *labelRepository) InsertBatch(ctx context.Context, labels []*Label) error {
	log := r.log.Function("InsertBatch")

	if len(labels) == 0 {
		return nil
	}

	if err := r.db.SQLWithContext(ctx).Create(&labels).Error; err != nil {
		return log.Err("failed to insert label batch", err, "count", len(labels))
	}

	return nil
}

func (r *labelRepository) UpdateBatch(ctx context.Context, labels []*Label) error {
	log := r.log.Function("UpdateBatch")

	if len(labels) == 0 {
		return nil
	}

	if err := r.db.SQLWithContext(ctx).Save(&labels).Error; err != nil {
		return log.Err("failed to update label batch", err, "count", len(labels))
	}

	return nil
}
