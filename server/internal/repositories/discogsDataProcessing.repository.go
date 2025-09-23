package repositories

import (
	"context"
	"waugzee/internal/database"
	"waugzee/internal/logger"
	"waugzee/internal/models"
)

type DiscogsDataProcessingRepository interface {
	Create(ctx context.Context, processing *models.DiscogsDataProcessing) (*models.DiscogsDataProcessing, error)
	Update(ctx context.Context, processing *models.DiscogsDataProcessing) error
	GetByID(ctx context.Context, id string) (*models.DiscogsDataProcessing, error)
	GetByYearMonth(ctx context.Context, yearMonth string) (*models.DiscogsDataProcessing, error)
	GetAll(ctx context.Context) ([]*models.DiscogsDataProcessing, error)
	GetByStatus(ctx context.Context, status models.ProcessingStatus) ([]*models.DiscogsDataProcessing, error)
	GetReadyForProcessing(ctx context.Context) ([]*models.DiscogsDataProcessing, error)
	Delete(ctx context.Context, id string) error
}

type discogsDataProcessingRepository struct {
	db  database.DB
	log logger.Logger
}

func NewDiscogsDataProcessingRepository(db database.DB) DiscogsDataProcessingRepository {
	return &discogsDataProcessingRepository{
		db:  db,
		log: logger.New("discogsDataProcessingRepository"),
	}
}

func (r *discogsDataProcessingRepository) Create(ctx context.Context, processing *models.DiscogsDataProcessing) (*models.DiscogsDataProcessing, error) {
	log := r.log.Function("Create")

	if err := r.db.Primary.WithContext(ctx).Create(processing).Error; err != nil {
		return nil, log.Err("failed to create discogs data processing record", err)
	}

	log.Info("Created discogs data processing record", "id", processing.ID, "yearMonth", processing.YearMonth)
	return processing, nil
}

func (r *discogsDataProcessingRepository) Update(ctx context.Context, processing *models.DiscogsDataProcessing) error {
	log := r.log.Function("Update")

	if err := r.db.Primary.WithContext(ctx).Save(processing).Error; err != nil {
		return log.Err("failed to update discogs data processing record", err, "id", processing.ID)
	}

	log.Debug("Updated discogs data processing record", "id", processing.ID, "status", processing.Status)
	return nil
}

func (r *discogsDataProcessingRepository) GetByID(ctx context.Context, id string) (*models.DiscogsDataProcessing, error) {
	log := r.log.Function("GetByID")

	var processing models.DiscogsDataProcessing
	if err := r.db.Primary.WithContext(ctx).Where("id = ?", id).First(&processing).Error; err != nil {
		return nil, log.Err("failed to get discogs data processing record by ID", err, "id", id)
	}

	return &processing, nil
}

func (r *discogsDataProcessingRepository) GetByYearMonth(ctx context.Context, yearMonth string) (*models.DiscogsDataProcessing, error) {
	log := r.log.Function("GetByYearMonth")

	var processing models.DiscogsDataProcessing
	if err := r.db.Primary.WithContext(ctx).Where("year_month = ?", yearMonth).First(&processing).Error; err != nil {
		return nil, log.Err("failed to get discogs data processing record by year month", err, "yearMonth", yearMonth)
	}

	return &processing, nil
}

func (r *discogsDataProcessingRepository) GetAll(ctx context.Context) ([]*models.DiscogsDataProcessing, error) {
	log := r.log.Function("GetAll")

	var processings []*models.DiscogsDataProcessing
	if err := r.db.Primary.WithContext(ctx).Order("year_month DESC").Find(&processings).Error; err != nil {
		return nil, log.Err("failed to get all discogs data processing records", err)
	}

	return processings, nil
}

func (r *discogsDataProcessingRepository) GetByStatus(ctx context.Context, status models.ProcessingStatus) ([]*models.DiscogsDataProcessing, error) {
	log := r.log.Function("GetByStatus")

	var processings []*models.DiscogsDataProcessing
	if err := r.db.Primary.WithContext(ctx).Where("status = ?", status).Order("year_month DESC").Find(&processings).Error; err != nil {
		return nil, log.Err("failed to get discogs data processing records by status", err, "status", status)
	}

	return processings, nil
}

func (r *discogsDataProcessingRepository) GetReadyForProcessing(ctx context.Context) ([]*models.DiscogsDataProcessing, error) {
	log := r.log.Function("GetReadyForProcessing")

	var processings []*models.DiscogsDataProcessing
	if err := r.db.Primary.WithContext(ctx).
		Where("status = ?", models.ProcessingStatusReadyForProcessing).
		Where("file_checksums IS NOT NULL").
		Where("download_completed_at IS NOT NULL").
		Order("year_month DESC").
		Find(&processings).Error; err != nil {
		return nil, log.Err("failed to get ready for processing records", err)
	}

	return processings, nil
}

func (r *discogsDataProcessingRepository) Delete(ctx context.Context, id string) error {
	log := r.log.Function("Delete")

	if err := r.db.Primary.WithContext(ctx).Delete(&models.DiscogsDataProcessing{}, "id = ?", id).Error; err != nil {
		return log.Err("failed to delete discogs data processing record", err, "id", id)
	}

	log.Info("Deleted discogs data processing record", "id", id)
	return nil
}