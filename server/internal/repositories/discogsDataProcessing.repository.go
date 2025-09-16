package repositories

import (
	"context"
	contextUtil "waugzee/internal/context"
	"waugzee/internal/database"
	"waugzee/internal/logger"
	. "waugzee/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DiscogsDataProcessingRepository interface {
	GetByYearMonth(ctx context.Context, yearMonth string) (*DiscogsDataProcessing, error)
	GetByID(ctx context.Context, id string) (*DiscogsDataProcessing, error)
	Create(ctx context.Context, processing *DiscogsDataProcessing) (*DiscogsDataProcessing, error)
	Update(ctx context.Context, processing *DiscogsDataProcessing) error
	Delete(ctx context.Context, id string) error
	GetByStatus(ctx context.Context, status ProcessingStatus) ([]*DiscogsDataProcessing, error)
	GetCurrentProcessing(ctx context.Context) (*DiscogsDataProcessing, error)
	UpdateStatus(
		ctx context.Context,
		id string,
		status ProcessingStatus,
		errorMessage *string,
	) error
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

func (r *discogsDataProcessingRepository) getDB(ctx context.Context) *gorm.DB {
	if tx, ok := contextUtil.GetTransaction(ctx); ok {
		return tx
	}
	return r.db.SQLWithContext(ctx)
}

func (r *discogsDataProcessingRepository) GetByYearMonth(
	ctx context.Context,
	yearMonth string,
) (*DiscogsDataProcessing, error) {
	log := r.log.Function("GetByYearMonth")

	var processing DiscogsDataProcessing
	if err := r.getDB(ctx).First(&processing, "year_month = ?", yearMonth).Error; err != nil {
		return nil, log.Err("failed to get processing by year month", err, "yearMonth", yearMonth)
	}

	return &processing, nil
}

func (r *discogsDataProcessingRepository) GetByID(
	ctx context.Context,
	id string,
) (*DiscogsDataProcessing, error) {
	log := r.log.Function("GetByID")

	parsedID, err := uuid.Parse(id)
	if err != nil {
		return nil, log.Err("failed to parse processing ID", err, "id", id)
	}

	var processing DiscogsDataProcessing
	if err := r.getDB(ctx).First(&processing, "id = ?", parsedID).Error; err != nil {
		return nil, log.Err("failed to get processing by id", err, "id", id)
	}

	return &processing, nil
}

func (r *discogsDataProcessingRepository) Create(
	ctx context.Context,
	processing *DiscogsDataProcessing,
) (*DiscogsDataProcessing, error) {
	log := r.log.Function("Create")

	if err := r.getDB(ctx).Create(processing).Error; err != nil {
		return nil, log.Err("failed to create processing", err, "yearMonth", processing.YearMonth)
	}

	return processing, nil
}

func (r *discogsDataProcessingRepository) Update(
	ctx context.Context,
	processing *DiscogsDataProcessing,
) error {
	log := r.log.Function("Update")

	if err := r.getDB(ctx).Save(processing).Error; err != nil {
		return log.Err("failed to update processing", err, "processing", processing)
	}

	return nil
}

func (r *discogsDataProcessingRepository) Delete(ctx context.Context, id string) error {
	log := r.log.Function("Delete")

	parsedID, err := uuid.Parse(id)
	if err != nil {
		return log.Err("failed to parse processing ID", err, "id", id)
	}

	if err := r.getDB(ctx).Delete(&DiscogsDataProcessing{}, "id = ?", parsedID).Error; err != nil {
		return log.Err("failed to delete processing", err, "id", id)
	}

	return nil
}

func (r *discogsDataProcessingRepository) GetByStatus(
	ctx context.Context,
	status ProcessingStatus,
) ([]*DiscogsDataProcessing, error) {
	log := r.log.Function("GetByStatus")

	var processing []*DiscogsDataProcessing
	if err := r.getDB(ctx).Find(&processing, &DiscogsDataProcessing{Status: status}).Error; err != nil {
		return nil, log.Err("failed to get processing by status", err, "status", status)
	}

	return processing, nil
}

func (r *discogsDataProcessingRepository) GetCurrentProcessing(
	ctx context.Context,
) (*DiscogsDataProcessing, error) {
	log := r.log.Function("GetCurrentProcessing")

	var processing DiscogsDataProcessing
	if err := r.getDB(ctx).Where("status IN ?", []ProcessingStatus{
		ProcessingStatusDownloading,
		ProcessingStatusReadyForProcessing,
		ProcessingStatusProcessing,
	}).First(&processing).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, log.Err("failed to get current processing", err)
	}

	return &processing, nil
}

// Claude is this even used? I have found multiple versions of the same thing
func (r *discogsDataProcessingRepository) UpdateStatus(
	ctx context.Context,
	id string,
	status ProcessingStatus,
	errorMessage *string,
) error {
	log := r.log.Function("UpdateStatus")

	parsedID, err := uuid.Parse(id)
	if err != nil {
		return log.Err("failed to parse processing ID", err, "id", id)
	}

	updates := map[string]interface{}{
		"status": status,
	}

	if errorMessage != nil {
		updates["error_message"] = *errorMessage
	}

	if err := r.getDB(ctx).Model(&DiscogsDataProcessing{}).Where("id = ?", parsedID).Updates(updates).Error; err != nil {
		return log.Err("failed to update processing status", err, "id", id, "status", status)
	}

	return nil
}
