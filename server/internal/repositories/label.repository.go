package repositories

import (
	"context"
	"strconv"
	"waugzee/internal/logger"
	. "waugzee/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type LabelRepository interface {
	GetByID(ctx context.Context, tx *gorm.DB, id string) (*Label, error)
	GetByDiscogsID(ctx context.Context, tx *gorm.DB, discogsID int64) (*Label, error)
	Create(ctx context.Context, tx *gorm.DB, label *Label) (*Label, error)
	Update(ctx context.Context, tx *gorm.DB, label *Label) error
	Delete(ctx context.Context, tx *gorm.DB, id string) error
	UpsertFileBatch(ctx context.Context, tx *gorm.DB, labels []*Label) error
	UpsertBatch(ctx context.Context, tx *gorm.DB, labels []*Label) error
	GetBatchByDiscogsIDs(
		ctx context.Context,
		tx *gorm.DB,
		discogsIDs []int64,
	) (map[int64]*Label, error)
	InsertBatch(ctx context.Context, tx *gorm.DB, labels []*Label) error
	UpdateBatch(ctx context.Context, tx *gorm.DB, labels []*Label) error
}

type labelRepository struct{}

func NewLabelRepository() LabelRepository {
	return &labelRepository{}
}

func (r *labelRepository) GetByID(ctx context.Context, tx *gorm.DB, id string) (*Label, error) {
	log := logger.NewWithContext(ctx, "labelRepository").Function("GetByID")

	labelID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return nil, log.Err("failed to parse label ID", err, "id", id)
	}

	label, err := gorm.G[*Label](tx).Where(BaseDiscogModel{ID: labelID}).First(ctx)
	if err != nil {
		return nil, log.Err("failed to get label by ID", err, "id", id)
	}

	return label, nil
}

func (r *labelRepository) GetByDiscogsID(
	ctx context.Context,
	tx *gorm.DB,
	discogsID int64,
) (*Label, error) {
	log := logger.NewWithContext(ctx, "labelRepository").Function("GetByDiscogsID")

	label, err := gorm.G[*Label](tx).Where(BaseDiscogModel{ID: discogsID}).First(ctx)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, log.Err("failed to get label by Discogs ID", err, "discogsID", discogsID)
	}

	return label, nil
}

func (r *labelRepository) Create(ctx context.Context, tx *gorm.DB, label *Label) (*Label, error) {
	log := logger.NewWithContext(ctx, "labelRepository").Function("Create")

	if err := tx.WithContext(ctx).Create(label).Error; err != nil {
		return nil, log.Err("failed to create label", err, "label", label)
	}

	return label, nil
}

func (r *labelRepository) Update(ctx context.Context, tx *gorm.DB, label *Label) error {
	log := logger.NewWithContext(ctx, "labelRepository").Function("Update")

	if err := tx.WithContext(ctx).Save(label).Error; err != nil {
		return log.Err("failed to update label", err, "label", label)
	}

	return nil
}

func (r *labelRepository) Delete(ctx context.Context, tx *gorm.DB, id string) error {
	log := logger.NewWithContext(ctx, "labelRepository").Function("Delete")

	labelID, err := uuid.Parse(id)
	if err != nil {
		return log.Err("failed to parse label ID", err, "id", id)
	}

	rowsAffected, err := gorm.G[*Label](tx).Where("id = ?", labelID).Delete(ctx)
	if err != nil {
		return log.Err("failed to delete label", err, "id", id)
	}

	if rowsAffected == 0 {
		return log.Err("label not found", nil, "id", id)
	}

	return nil
}

func (r *labelRepository) UpsertFileBatch(ctx context.Context, tx *gorm.DB, labels []*Label) error {
	log := logger.NewWithContext(ctx, "labelRepository").Function("UpsertFileBatch")

	if len(labels) == 0 {
		return nil
	}

	if err := tx.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"name", "profile", "updated_at", "resource_url", "uri"}),
	}).Create(&labels).Error; err != nil {
		return log.Err("failed to upsert label batch", err, "count", len(labels))
	}

	return nil
}

// TODO: This should handle updates from folder sync
func (r *labelRepository) UpsertBatch(ctx context.Context, tx *gorm.DB, labels []*Label) error {
	log := logger.NewWithContext(ctx, "labelRepository").Function("UpsertBatch")

	if len(labels) == 0 {
		return nil
	}

	log.Info("Upserting labels", "count", len(labels))

	if err := tx.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"name", "profile", "updated_at"}),
	}).Create(&labels).Error; err != nil {
		return log.Err("failed to upsert label batch", err, "count", len(labels))
	}

	log.Info("Successfully upserted labels", "count", len(labels))
	return nil
}

func (r *labelRepository) GetBatchByDiscogsIDs(
	ctx context.Context,
	tx *gorm.DB,
	discogsIDs []int64,
) (map[int64]*Label, error) {
	log := logger.NewWithContext(ctx, "labelRepository").Function("GetBatchByDiscogsIDs")

	if len(discogsIDs) == 0 {
		return make(map[int64]*Label), nil
	}

	labels, err := gorm.G[*Label](tx).Where("id IN ?", discogsIDs).Find(ctx)
	if err != nil {
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

func (r *labelRepository) InsertBatch(ctx context.Context, tx *gorm.DB, labels []*Label) error {
	log := logger.NewWithContext(ctx, "labelRepository").Function("InsertBatch")

	if len(labels) == 0 {
		return nil
	}

	if err := tx.WithContext(ctx).Create(labels).Error; err != nil {
		return log.Err("failed to insert label batch", err, "count", len(labels))
	}

	return nil
}

func (r *labelRepository) UpdateBatch(ctx context.Context, tx *gorm.DB, labels []*Label) error {
	log := logger.NewWithContext(ctx, "labelRepository").Function("UpdateBatch")

	if len(labels) == 0 {
		return nil
	}

	if err := tx.WithContext(ctx).Save(labels).Error; err != nil {
		return log.Err("failed to update label batch", err, "count", len(labels))
	}

	return nil
}
