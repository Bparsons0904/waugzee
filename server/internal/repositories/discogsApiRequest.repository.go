package repositories

import (
	"context"
	"waugzee/internal/database"
	"waugzee/internal/models"

	"github.com/google/uuid"
)

type DiscogsApiRequestRepository interface {
	Create(ctx context.Context, request *models.DiscogsApiRequest) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.DiscogsApiRequest, error)
	GetByRequestID(ctx context.Context, requestID string) (*models.DiscogsApiRequest, error)
	GetBySyncSession(ctx context.Context, syncSessionID uuid.UUID) ([]models.DiscogsApiRequest, error)
	GetPendingBySyncSession(ctx context.Context, syncSessionID uuid.UUID) ([]models.DiscogsApiRequest, error)
	GetByUserAndStatus(ctx context.Context, userID uuid.UUID, status models.RequestStatus) ([]models.DiscogsApiRequest, error)
	Update(ctx context.Context, request *models.DiscogsApiRequest) error
	UpdateStatus(ctx context.Context, requestID string, status models.RequestStatus) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetFailedRequests(ctx context.Context, syncSessionID uuid.UUID) ([]models.DiscogsApiRequest, error)
	GetCompletedRequests(ctx context.Context, syncSessionID uuid.UUID) ([]models.DiscogsApiRequest, error)
	CountByStatus(ctx context.Context, syncSessionID uuid.UUID, status models.RequestStatus) (int64, error)
}

type discogsApiRequestRepository struct {
	db database.DB
}

func NewDiscogsApiRequestRepository(db database.DB) DiscogsApiRequestRepository {
	return &discogsApiRequestRepository{
		db: db,
	}
}

func (r *discogsApiRequestRepository) Create(ctx context.Context, request *models.DiscogsApiRequest) error {
	return r.db.WithContext(ctx).Create(request).Error
}

func (r *discogsApiRequestRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.DiscogsApiRequest, error) {
	var request models.DiscogsApiRequest
	err := r.db.WithContext(ctx).
		Preload("User").
		Preload("SyncSession").
		Where("id = ?", id).
		First(&request).Error
	if err != nil {
		return nil, err
	}
	return &request, nil
}

func (r *discogsApiRequestRepository) GetByRequestID(ctx context.Context, requestID string) (*models.DiscogsApiRequest, error) {
	var request models.DiscogsApiRequest
	err := r.db.WithContext(ctx).
		Preload("User").
		Preload("SyncSession").
		Where("request_id = ?", requestID).
		First(&request).Error
	if err != nil {
		return nil, err
	}
	return &request, nil
}

func (r *discogsApiRequestRepository) GetBySyncSession(ctx context.Context, syncSessionID uuid.UUID) ([]models.DiscogsApiRequest, error) {
	var requests []models.DiscogsApiRequest
	err := r.db.WithContext(ctx).
		Where("sync_session_id = ?", syncSessionID).
		Order("created_at ASC").
		Find(&requests).Error
	return requests, err
}

func (r *discogsApiRequestRepository) GetPendingBySyncSession(ctx context.Context, syncSessionID uuid.UUID) ([]models.DiscogsApiRequest, error) {
	var requests []models.DiscogsApiRequest
	err := r.db.WithContext(ctx).
		Where("sync_session_id = ? AND status = ?", syncSessionID, models.RequestStatusPending).
		Order("created_at ASC").
		Find(&requests).Error
	return requests, err
}

func (r *discogsApiRequestRepository) GetByUserAndStatus(ctx context.Context, userID uuid.UUID, status models.RequestStatus) ([]models.DiscogsApiRequest, error) {
	var requests []models.DiscogsApiRequest
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND status = ?", userID, status).
		Order("created_at ASC").
		Find(&requests).Error
	return requests, err
}

func (r *discogsApiRequestRepository) Update(ctx context.Context, request *models.DiscogsApiRequest) error {
	return r.db.WithContext(ctx).Save(request).Error
}

func (r *discogsApiRequestRepository) UpdateStatus(ctx context.Context, requestID string, status models.RequestStatus) error {
	return r.db.WithContext(ctx).
		Model(&models.DiscogsApiRequest{}).
		Where("request_id = ?", requestID).
		Update("status", status).Error
}

func (r *discogsApiRequestRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&models.DiscogsApiRequest{}, id).Error
}

func (r *discogsApiRequestRepository) GetFailedRequests(ctx context.Context, syncSessionID uuid.UUID) ([]models.DiscogsApiRequest, error) {
	var requests []models.DiscogsApiRequest
	err := r.db.WithContext(ctx).
		Where("sync_session_id = ? AND status = ?", syncSessionID, models.RequestStatusFailed).
		Order("created_at ASC").
		Find(&requests).Error
	return requests, err
}

func (r *discogsApiRequestRepository) GetCompletedRequests(ctx context.Context, syncSessionID uuid.UUID) ([]models.DiscogsApiRequest, error) {
	var requests []models.DiscogsApiRequest
	err := r.db.WithContext(ctx).
		Where("sync_session_id = ? AND status = ?", syncSessionID, models.RequestStatusCompleted).
		Order("created_at ASC").
		Find(&requests).Error
	return requests, err
}

func (r *discogsApiRequestRepository) CountByStatus(ctx context.Context, syncSessionID uuid.UUID, status models.RequestStatus) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.DiscogsApiRequest{}).
		Where("sync_session_id = ? AND status = ?", syncSessionID, status).
		Count(&count).Error
	return count, err
}