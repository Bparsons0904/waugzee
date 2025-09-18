package repositories

import (
	"context"
	"waugzee/internal/database"
	"waugzee/internal/logger"
	. "waugzee/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm/clause"
)



type ImageRepository interface {
	GetByID(ctx context.Context, id string) (*Image, error)
	GetByImageableID(
		ctx context.Context,
		imageableID string,
		imageableType string,
	) ([]*Image, error)
	Create(ctx context.Context, image *Image) (*Image, error)
	Update(ctx context.Context, image *Image) error
	Delete(ctx context.Context, id string) error
	UpsertBatch(ctx context.Context, images []*Image) (int, int, error)
	DeleteByImageableID(ctx context.Context, imageableID string, imageableType string) error
}

type imageRepository struct {
	db  database.DB
	log logger.Logger
}

func NewImageRepository(db database.DB) ImageRepository {
	return &imageRepository{
		db:  db,
		log: logger.New("imageRepository"),
	}
}

func (r *imageRepository) GetByID(ctx context.Context, id string) (*Image, error) {
	log := r.log.Function("GetByID")

	imageID, err := uuid.Parse(id)
	if err != nil {
		return nil, log.Err("failed to parse image ID", err, "id", id)
	}

	var image Image
	if err := r.db.SQLWithContext(ctx).First(&image, "id = ?", imageID).Error; err != nil {
		return nil, log.Err("failed to get image by ID", err, "id", id)
	}

	return &image, nil
}

func (r *imageRepository) GetByImageableID(
	ctx context.Context,
	imageableID string,
	imageableType string,
) ([]*Image, error) {
	log := r.log.Function("GetByImageableID")

	imageableUUID, err := uuid.Parse(imageableID)
	if err != nil {
		return nil, log.Err("failed to parse imageable ID", err, "imageableID", imageableID)
	}

	var images []*Image
	if err := r.db.SQLWithContext(ctx).Where("imageable_id = ? AND imageable_type = ?", imageableUUID, imageableType).Order("sort_order").Find(&images).Error; err != nil {
		return nil, log.Err(
			"failed to get images by imageable ID",
			err,
			"imageableID",
			imageableID,
			"imageableType",
			imageableType,
		)
	}

	return images, nil
}

func (r *imageRepository) Create(ctx context.Context, image *Image) (*Image, error) {
	log := r.log.Function("Create")

	if err := r.db.SQLWithContext(ctx).Create(image).Error; err != nil {
		return nil, log.Err("failed to create image", err, "image", image)
	}

	return image, nil
}

func (r *imageRepository) Update(ctx context.Context, image *Image) error {
	log := r.log.Function("Update")

	if err := r.db.SQLWithContext(ctx).Save(image).Error; err != nil {
		return log.Err("failed to update image", err, "imageID", image.ID)
	}

	return nil
}

func (r *imageRepository) Delete(ctx context.Context, id string) error {
	log := r.log.Function("Delete")

	imageID, err := uuid.Parse(id)
	if err != nil {
		return log.Err("failed to parse image ID", err, "id", id)
	}

	if err := r.db.SQLWithContext(ctx).Delete(&Image{}, "id = ?", imageID).Error; err != nil {
		return log.Err("failed to delete image", err, "id", id)
	}

	return nil
}

func (r *imageRepository) UpsertBatch(ctx context.Context, images []*Image) (int, int, error) {
	log := r.log.Function("UpsertBatch")

	if len(images) == 0 {
		return 0, 0, nil
	}

	db := r.db.SQLWithContext(ctx)

	// Use native PostgreSQL UPSERT with ON CONFLICT for single database round-trip
	// We'll use a composite key of imageable_id, imageable_type, and url for uniqueness
	result := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "imageable_id"},
			{Name: "imageable_type"},
			{Name: "url"},
		},
		DoUpdates: clause.AssignmentColumns([]string{
			"alt_text", "width", "height", "file_size", "mime_type",
			"image_type", "sort_order", "discogs_id", "discogs_type",
			"discogs_uri", "discogs_uri150", "updated_at",
		}),
	}).Create(images)

	if result.Error != nil {
		return 0, 0, log.Err("failed to upsert image batch", result.Error, "count", len(images))
	}

	affectedRows := int(result.RowsAffected)
	log.Info("Upserted images", "count", affectedRows)
	return affectedRows, 0, nil
}

func (r *imageRepository) DeleteByImageableID(
	ctx context.Context,
	imageableID string,
	imageableType string,
) error {
	log := r.log.Function("DeleteByImageableID")

	imageableUUID, err := uuid.Parse(imageableID)
	if err != nil {
		return log.Err("failed to parse imageable ID", err, "imageableID", imageableID)
	}

	if err := r.db.SQLWithContext(ctx).Where("imageable_id = ? AND imageable_type = ?", imageableUUID, imageableType).Delete(&Image{}).Error; err != nil {
		return log.Err(
			"failed to delete images by imageable ID",
			err,
			"imageableID",
			imageableID,
			"imageableType",
			imageableType,
		)
	}

	return nil
}

