package repositories

import (
	"context"
	"waugzee/internal/database"
	"waugzee/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DiscogsCollectionSyncRepository interface {
	Create(ctx context.Context, sync *models.DiscogsCollectionSync) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.DiscogsCollectionSync, error)
	GetBySessionID(ctx context.Context, sessionID string) (*models.DiscogsCollectionSync, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]models.DiscogsCollectionSync, error)
	GetActiveByUserID(ctx context.Context, userID uuid.UUID) ([]models.DiscogsCollectionSync, error)
	GetPausedByUserID(ctx context.Context, userID uuid.UUID) ([]models.DiscogsCollectionSync, error)
	GetByStatus(ctx context.Context, status models.SyncStatus) ([]models.DiscogsCollectionSync, error)
	Update(ctx context.Context, sync *models.DiscogsCollectionSync) error
	UpdateStatus(ctx context.Context, sessionID string, status models.SyncStatus) error
	UpdateProgress(ctx context.Context, sessionID string, completed, failed int) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetLatestByUserAndType(ctx context.Context, userID uuid.UUID, syncType models.SyncType) (*models.DiscogsCollectionSync, error)
	GetWithApiRequests(ctx context.Context, sessionID string) (*models.DiscogsCollectionSync, error)
}

type discogsCollectionSyncRepository struct {
	db database.DB
}

func NewDiscogsCollectionSyncRepository(db database.DB) DiscogsCollectionSyncRepository {
	return &discogsCollectionSyncRepository{
		db: db,
	}
}

func (r *discogsCollectionSyncRepository) Create(ctx context.Context, sync *models.DiscogsCollectionSync) error {
	return r.db.SQLWithContext(ctx).Create(sync).Error
}

func (r *discogsCollectionSyncRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.DiscogsCollectionSync, error) {
	var sync models.DiscogsCollectionSync
	err := r.db.SQLWithContext(ctx).
		Preload("User").
		Where("id = ?", id).
		First(&sync).Error
	if err != nil {
		return nil, err
	}
	return &sync, nil
}

func (r *discogsCollectionSyncRepository) GetBySessionID(ctx context.Context, sessionID string) (*models.DiscogsCollectionSync, error) {
	var sync models.DiscogsCollectionSync
	err := r.db.SQLWithContext(ctx).
		Preload("User").
		Where("session_id = ?", sessionID).
		First(&sync).Error
	if err != nil {
		return nil, err
	}
	return &sync, nil
}

func (r *discogsCollectionSyncRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]models.DiscogsCollectionSync, error) {
	var syncs []models.DiscogsCollectionSync
	err := r.db.SQLWithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&syncs).Error
	return syncs, err
}

func (r *discogsCollectionSyncRepository) GetActiveByUserID(ctx context.Context, userID uuid.UUID) ([]models.DiscogsCollectionSync, error) {
	var syncs []models.DiscogsCollectionSync
	err := r.db.SQLWithContext(ctx).
		Where("user_id = ? AND status IN (?)", userID, []models.SyncStatus{
			models.SyncStatusInitiated,
			models.SyncStatusInProgress,
		}).
		Order("created_at DESC").
		Find(&syncs).Error
	return syncs, err
}

func (r *discogsCollectionSyncRepository) GetPausedByUserID(ctx context.Context, userID uuid.UUID) ([]models.DiscogsCollectionSync, error) {
	var syncs []models.DiscogsCollectionSync
	err := r.db.SQLWithContext(ctx).
		Where("user_id = ? AND status = ?", userID, models.SyncStatusPaused).
		Order("created_at DESC").
		Find(&syncs).Error
	return syncs, err
}

func (r *discogsCollectionSyncRepository) GetByStatus(ctx context.Context, status models.SyncStatus) ([]models.DiscogsCollectionSync, error) {
	var syncs []models.DiscogsCollectionSync
	err := r.db.SQLWithContext(ctx).
		Preload("User").
		Where("status = ?", status).
		Order("created_at ASC").
		Find(&syncs).Error
	return syncs, err
}

func (r *discogsCollectionSyncRepository) Update(ctx context.Context, sync *models.DiscogsCollectionSync) error {
	return r.db.SQLWithContext(ctx).Save(sync).Error
}

func (r *discogsCollectionSyncRepository) UpdateStatus(ctx context.Context, sessionID string, status models.SyncStatus) error {
	return r.db.SQLWithContext(ctx).
		Model(&models.DiscogsCollectionSync{}).
		Where("session_id = ?", sessionID).
		Update("status", status).Error
}

func (r *discogsCollectionSyncRepository) UpdateProgress(ctx context.Context, sessionID string, completed, failed int) error {
	return r.db.SQLWithContext(ctx).
		Model(&models.DiscogsCollectionSync{}).
		Where("session_id = ?", sessionID).
		Updates(map[string]interface{}{
			"completed_requests": completed,
			"failed_requests":    failed,
		}).Error
}

func (r *discogsCollectionSyncRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.SQLWithContext(ctx).Delete(&models.DiscogsCollectionSync{}, id).Error
}

func (r *discogsCollectionSyncRepository) GetLatestByUserAndType(ctx context.Context, userID uuid.UUID, syncType models.SyncType) (*models.DiscogsCollectionSync, error) {
	var sync models.DiscogsCollectionSync
	err := r.db.SQLWithContext(ctx).
		Where("user_id = ? AND sync_type = ?", userID, syncType).
		Order("created_at DESC").
		First(&sync).Error
	if err != nil {
		return nil, err
	}
	return &sync, nil
}

func (r *discogsCollectionSyncRepository) GetWithApiRequests(ctx context.Context, sessionID string) (*models.DiscogsCollectionSync, error) {
	var sync models.DiscogsCollectionSync
	err := r.db.SQLWithContext(ctx).
		Preload("User").
		Preload("ApiRequests", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at ASC")
		}).
		Where("session_id = ?", sessionID).
		First(&sync).Error
	if err != nil {
		return nil, err
	}
	return &sync, nil
}