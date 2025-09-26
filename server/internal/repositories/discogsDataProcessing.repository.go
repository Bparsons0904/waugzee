package repositories

import (
	"context"
	"waugzee/internal/logger"
	. "waugzee/internal/models"

	"gorm.io/gorm"
)

type DiscogsDataProcessingRepository interface {
	Create(ctx context.Context, processing *DiscogsDataProcessing) (*DiscogsDataProcessing, error)
	GetByYearMonth(ctx context.Context, yearMonth string) (*DiscogsDataProcessing, error)
	Update(ctx context.Context, processing *DiscogsDataProcessing) error
	GetAll(ctx context.Context) ([]*DiscogsDataProcessing, error)
	GetLatestProcessing(ctx context.Context) (*DiscogsDataProcessing, error)
}

type discogsDataProcessingRepository struct {
	db  *gorm.DB
	log logger.Logger
}

func NewDiscogsDataProcessingRepository(db *gorm.DB) DiscogsDataProcessingRepository {
	return &discogsDataProcessingRepository{
		db:  db,
		log: logger.New("discogsDataProcessingRepository"),
	}
}

func (r *discogsDataProcessingRepository) Create(ctx context.Context, processing *DiscogsDataProcessing) (*DiscogsDataProcessing, error) {
	log := r.log.Function("Create")

	if err := r.db.WithContext(ctx).Create(processing).Error; err != nil {
		return nil, log.Err("failed to create discogs data processing record", err)
	}

	return processing, nil
}

func (r *discogsDataProcessingRepository) GetByYearMonth(ctx context.Context, yearMonth string) (*DiscogsDataProcessing, error) {
	log := r.log.Function("GetByYearMonth")

	var processing DiscogsDataProcessing
	err := r.db.WithContext(ctx).Where("year_month = ?", yearMonth).First(&processing).Error
	if err != nil {
		return nil, log.Err("failed to get discogs data processing record by year month", err)
	}

	return &processing, nil
}

func (r *discogsDataProcessingRepository) Update(ctx context.Context, processing *DiscogsDataProcessing) error {
	log := r.log.Function("Update")

	if err := r.db.WithContext(ctx).Save(processing).Error; err != nil {
		return log.Err("failed to update discogs data processing record", err)
	}

	return nil
}

func (r *discogsDataProcessingRepository) GetAll(ctx context.Context) ([]*DiscogsDataProcessing, error) {
	log := r.log.Function("GetAll")

	var processings []*DiscogsDataProcessing
	err := r.db.WithContext(ctx).Find(&processings).Error
	if err != nil {
		return nil, log.Err("failed to get all discogs data processing records", err)
	}

	return processings, nil
}

func (r *discogsDataProcessingRepository) GetLatestProcessing(ctx context.Context) (*DiscogsDataProcessing, error) {
	log := r.log.Function("GetLatestProcessing")

	var processing DiscogsDataProcessing
	err := r.db.WithContext(ctx).
		Where("status IN (?)", []ProcessingStatus{ProcessingStatusReadyForProcessing, ProcessingStatusProcessing}).
		Order("created_at DESC").
		First(&processing).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, log.Err("failed to get latest processing record", err)
	}

	return &processing, nil
}