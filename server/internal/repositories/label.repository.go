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
	LABEL_BATCH_SIZE = 1000
)

type LabelRepository interface {
	GetByID(ctx context.Context, id string) (*Label, error)
	GetByDiscogsID(ctx context.Context, discogsID int64) (*Label, error)
	Create(ctx context.Context, label *Label) (*Label, error)
	Update(ctx context.Context, label *Label) error
	Delete(ctx context.Context, id string) error
	UpsertBatch(ctx context.Context, labels []*Label) (int, int, error)
	GetBatchByDiscogsIDs(ctx context.Context, discogsIDs []int64) (map[int64]*Label, error)
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

func (r *labelRepository) getDB(ctx context.Context) *gorm.DB {
	if tx, ok := contextutil.GetTransaction(ctx); ok {
		return tx
	}
	return r.db.SQLWithContext(ctx)
}

func (r *labelRepository) GetByID(ctx context.Context, id string) (*Label, error) {
	log := r.log.Function("GetByID")

	labelID, err := uuid.Parse(id)
	if err != nil {
		return nil, log.Err("failed to parse label ID", err, "id", id)
	}

	var label Label
	if err := r.getDB(ctx).First(&label, "id = ?", labelID).Error; err != nil {
		return nil, log.Err("failed to get label by ID", err, "id", id)
	}

	return &label, nil
}

func (r *labelRepository) GetByDiscogsID(ctx context.Context, discogsID int64) (*Label, error) {
	log := r.log.Function("GetByDiscogsID")

	var label Label
	if err := r.getDB(ctx).First(&label, "discogs_id = ?", discogsID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, log.Err("failed to get label by Discogs ID", err, "discogsID", discogsID)
	}

	return &label, nil
}

func (r *labelRepository) Create(ctx context.Context, label *Label) (*Label, error) {
	log := r.log.Function("Create")

	if err := r.getDB(ctx).Create(label).Error; err != nil {
		return nil, log.Err("failed to create label", err, "label", label)
	}

	return label, nil
}

func (r *labelRepository) Update(ctx context.Context, label *Label) error {
	log := r.log.Function("Update")

	if err := r.getDB(ctx).Save(label).Error; err != nil {
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

	if err := r.getDB(ctx).Delete(&Label{}, "id = ?", labelID).Error; err != nil {
		return log.Err("failed to delete label", err, "id", id)
	}

	return nil
}

func (r *labelRepository) UpsertBatch(ctx context.Context, labels []*Label) (int, int, error) {
	log := r.log.Function("UpsertBatch")

	if len(labels) == 0 {
		return 0, 0, nil
	}

	var totalInserted, totalUpdated int

	// Process in batches to avoid memory issues
	for i := 0; i < len(labels); i += LABEL_BATCH_SIZE {
		end := i + LABEL_BATCH_SIZE
		if end > len(labels) {
			end = len(labels)
		}

		batch := labels[i:end]
		inserted, updated, err := r.upsertSingleBatch(ctx, batch)
		if err != nil {
			return totalInserted, totalUpdated, log.Err("failed to upsert batch", err, "batchStart", i, "batchEnd", end)
		}

		totalInserted += inserted
		totalUpdated += updated

		log.Info("Processed label batch", "batchStart", i, "batchEnd", end, "inserted", inserted, "updated", updated)
	}

	return totalInserted, totalUpdated, nil
}

func (r *labelRepository) upsertSingleBatch(ctx context.Context, labels []*Label) (int, int, error) {
	log := r.log.Function("upsertSingleBatch")

	db := r.getDB(ctx)

	// Extract Discogs IDs for existing labels lookup
	discogsIDs := make([]int64, 0, len(labels))
	for _, label := range labels {
		if label.DiscogsID != nil {
			discogsIDs = append(discogsIDs, *label.DiscogsID)
		}
	}

	// Get existing labels by Discogs ID
	existingLabels, err := r.GetBatchByDiscogsIDs(ctx, discogsIDs)
	if err != nil {
		return 0, 0, log.Err("failed to get existing labels", err)
	}

	var toInsert []*Label
	var toUpdate []*Label

	// Separate into insert and update batches
	for _, label := range labels {
		if label.DiscogsID != nil {
			if existing, exists := existingLabels[*label.DiscogsID]; exists {
				// Update existing label
				existing.Name = label.Name
				existing.Country = label.Country
				existing.FoundedYear = label.FoundedYear
				existing.Website = label.Website
				existing.ImageURL = label.ImageURL
				toUpdate = append(toUpdate, existing)
			} else {
				// Insert new label
				toInsert = append(toInsert, label)
			}
		} else {
			// No Discogs ID, always insert (shouldn't happen in normal processing)
			toInsert = append(toInsert, label)
		}
	}

	var inserted, updated int

	// Handle inserts
	if len(toInsert) > 0 {
		if err := db.CreateInBatches(toInsert, LABEL_BATCH_SIZE).Error; err != nil {
			return 0, 0, log.Err("failed to insert label batch", err, "count", len(toInsert))
		}
		inserted = len(toInsert)
		log.Info("Inserted labels", "count", inserted)
	}

	// Handle updates using GORM's Clauses for upsert
	if len(toUpdate) > 0 {
		if err := db.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"name", "country", "founded_year", "website", "image_url", "updated_at",
			}),
		}).CreateInBatches(toUpdate, LABEL_BATCH_SIZE).Error; err != nil {
			return inserted, 0, log.Err("failed to update label batch", err, "count", len(toUpdate))
		}
		updated = len(toUpdate)
		log.Info("Updated labels", "count", updated)
	}

	return inserted, updated, nil
}

func (r *labelRepository) GetBatchByDiscogsIDs(ctx context.Context, discogsIDs []int64) (map[int64]*Label, error) {
	log := r.log.Function("GetBatchByDiscogsIDs")

	if len(discogsIDs) == 0 {
		return make(map[int64]*Label), nil
	}

	var labels []*Label
	if err := r.getDB(ctx).Where("discogs_id IN ?", discogsIDs).Find(&labels).Error; err != nil {
		return nil, log.Err("failed to get labels by Discogs IDs", err, "count", len(discogsIDs))
	}

	// Convert to map for O(1) lookup
	result := make(map[int64]*Label, len(labels))
	for _, label := range labels {
		if label.DiscogsID != nil {
			result[*label.DiscogsID] = label
		}
	}

	log.Info("Retrieved labels by Discogs IDs", "requested", len(discogsIDs), "found", len(result))
	return result, nil
}