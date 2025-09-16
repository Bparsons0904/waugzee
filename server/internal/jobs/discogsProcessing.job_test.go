package jobs

import (
	"context"
	"errors"
	"testing"
	"time"
	"waugzee/internal/models"
	"waugzee/internal/services"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock repositories and services
type MockDiscogsDataProcessingRepository struct {
	mock.Mock
}

func (m *MockDiscogsDataProcessingRepository) GetByYearMonth(ctx context.Context, yearMonth string) (*models.DiscogsDataProcessing, error) {
	args := m.Called(ctx, yearMonth)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.DiscogsDataProcessing), args.Error(1)
}

func (m *MockDiscogsDataProcessingRepository) GetByID(ctx context.Context, id string) (*models.DiscogsDataProcessing, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.DiscogsDataProcessing), args.Error(1)
}

func (m *MockDiscogsDataProcessingRepository) Create(ctx context.Context, processing *models.DiscogsDataProcessing) (*models.DiscogsDataProcessing, error) {
	args := m.Called(ctx, processing)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.DiscogsDataProcessing), args.Error(1)
}

func (m *MockDiscogsDataProcessingRepository) Update(ctx context.Context, processing *models.DiscogsDataProcessing) error {
	args := m.Called(ctx, processing)
	return args.Error(0)
}

func (m *MockDiscogsDataProcessingRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockDiscogsDataProcessingRepository) GetByStatus(ctx context.Context, status models.ProcessingStatus) ([]*models.DiscogsDataProcessing, error) {
	args := m.Called(ctx, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.DiscogsDataProcessing), args.Error(1)
}

func (m *MockDiscogsDataProcessingRepository) GetCurrentProcessing(ctx context.Context) (*models.DiscogsDataProcessing, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.DiscogsDataProcessing), args.Error(1)
}

func (m *MockDiscogsDataProcessingRepository) UpdateStatus(ctx context.Context, id string, status models.ProcessingStatus, errorMessage *string) error {
	args := m.Called(ctx, id, status, errorMessage)
	return args.Error(0)
}


type MockXMLProcessingService struct {
	mock.Mock
}

func (m *MockXMLProcessingService) ProcessLabelsFile(ctx context.Context, filePath string, processingID string) (*services.ProcessingResult, error) {
	args := m.Called(ctx, filePath, processingID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.ProcessingResult), args.Error(1)
}

func TestDiscogsProcessingJob_Name(t *testing.T) {
	job := &DiscogsProcessingJob{}
	name := job.Name()
	assert.Equal(t, "DiscogsDailyProcessingCheck", name)
}

func TestDiscogsProcessingJob_Schedule(t *testing.T) {
	schedule := services.DailyProcessing
	job := NewDiscogsProcessingJob(nil, nil, schedule)
	assert.Equal(t, schedule, job.Schedule())
}

func TestDiscogsProcessingJob_Execute_NoRecordsToProcess(t *testing.T) {
	// Setup mocks
	repo := &MockDiscogsDataProcessingRepository{}
	xmlProcessing := &MockXMLProcessingService{}

	// Mock no records ready for processing
	repo.On("GetByStatus", mock.Anything, models.ProcessingStatusReadyForProcessing).Return([]*models.DiscogsDataProcessing{}, nil)
	repo.On("GetByStatus", mock.Anything, models.ProcessingStatusProcessing).Return([]*models.DiscogsDataProcessing{}, nil)

	job := NewDiscogsProcessingJob(repo, xmlProcessing, services.DailyProcessing)

	err := job.Execute(context.Background())
	assert.NoError(t, err)

	repo.AssertExpectations(t)
}

func TestDiscogsProcessingJob_FindAndHandleProcessingRecords(t *testing.T) {
	// Setup mocks
	repo := &MockDiscogsDataProcessingRepository{}
	xmlProcessing := &MockXMLProcessingService{}

	// Create test data - stuck record (processing for > 24 hours)
	twentyFiveHoursAgo := time.Now().UTC().Add(-25 * time.Hour)
	stuckRecord := &models.DiscogsDataProcessing{
		ID:        uuid.New(),
		YearMonth: "2025-01",
		Status:    models.ProcessingStatusProcessing,
		StartedAt: &twentyFiveHoursAgo,
	}

	// Create test data - recent processing record (< 24 hours)
	oneHourAgo := time.Now().UTC().Add(-1 * time.Hour)
	recentRecord := &models.DiscogsDataProcessing{
		ID:        uuid.New(),
		YearMonth: "2025-02",
		Status:    models.ProcessingStatusProcessing,
		StartedAt: &oneHourAgo,
	}

	// Mock repository calls
	repo.On("GetByStatus", mock.Anything, models.ProcessingStatusProcessing).Return([]*models.DiscogsDataProcessing{stuckRecord, recentRecord}, nil)
	repo.On("Update", mock.Anything, stuckRecord).Return(nil)

	job := NewDiscogsProcessingJob(repo, xmlProcessing, services.DailyProcessing)

	handledRecords, err := job.findAndHandleProcessingRecords(context.Background())
	assert.NoError(t, err)
	assert.Len(t, handledRecords, 1)
	assert.Equal(t, stuckRecord.ID, handledRecords[0].ID)
	assert.Equal(t, models.ProcessingStatusReadyForProcessing, handledRecords[0].Status)

	repo.AssertExpectations(t)
}

func TestDiscogsProcessingJob_Execute_WithMixedRecords(t *testing.T) {
	// Setup mocks
	repo := &MockDiscogsDataProcessingRepository{}
	xmlProcessing := &MockXMLProcessingService{}

	// Create test data - one ready record and one stuck record
	readyRecord := &models.DiscogsDataProcessing{
		ID:        uuid.New(),
		YearMonth: "2025-01",
		Status:    models.ProcessingStatusReadyForProcessing,
		ProcessingStats: &models.ProcessingStats{
			Files: map[string]*models.FileStatus{
				"labels": {
					Status:    models.FileDownloadStatusFailed, // No validated file
					Downloaded: false,
					Validated: false,
				},
			},
		},
	}

	twentyFiveHoursAgo := time.Now().UTC().Add(-25 * time.Hour)
	stuckRecord := &models.DiscogsDataProcessing{
		ID:        uuid.New(),
		YearMonth: "2025-02",
		Status:    models.ProcessingStatusProcessing,
		StartedAt: &twentyFiveHoursAgo,
		ProcessingStats: &models.ProcessingStats{
			Files: map[string]*models.FileStatus{
				"labels": {
					Status:    models.FileDownloadStatusFailed, // No validated file
					Downloaded: false,
					Validated: false,
				},
			},
		},
	}

	// Mock repository calls
	repo.On("GetByStatus", mock.Anything, models.ProcessingStatusReadyForProcessing).Return([]*models.DiscogsDataProcessing{readyRecord}, nil)
	repo.On("GetByStatus", mock.Anything, models.ProcessingStatusProcessing).Return([]*models.DiscogsDataProcessing{stuckRecord}, nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*models.DiscogsDataProcessing")).Return(nil)

	job := NewDiscogsProcessingJob(repo, xmlProcessing, services.DailyProcessing)

	err := job.Execute(context.Background())
	assert.NoError(t, err)

	// Both records should have been processed (even if they complete without processing files)
	repo.AssertExpectations(t)
}

func TestDiscogsProcessingJob_Execute_ErrorHandling(t *testing.T) {
	// Setup mocks
	repo := &MockDiscogsDataProcessingRepository{}
	xmlProcessing := &MockXMLProcessingService{}

	// Mock repository error
	repoError := errors.New("database connection failed")
	repo.On("GetByStatus", mock.Anything, models.ProcessingStatusReadyForProcessing).Return(nil, repoError)

	job := NewDiscogsProcessingJob(repo, xmlProcessing, services.DailyProcessing)

	err := job.Execute(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database connection failed")

	repo.AssertExpectations(t)
}

func TestDiscogsProcessingJob_ProcessRecord_RepositoryError(t *testing.T) {
	// Setup mocks
	repo := &MockDiscogsDataProcessingRepository{}
	xmlProcessing := &MockXMLProcessingService{}

	record := &models.DiscogsDataProcessing{
		ID:        uuid.New(),
		YearMonth: "2025-01",
		Status:    models.ProcessingStatusReadyForProcessing,
	}

	// Mock repository update error
	updateError := errors.New("database update failed")
	repo.On("Update", mock.Anything, mock.AnythingOfType("*models.DiscogsDataProcessing")).Return(updateError)

	job := NewDiscogsProcessingJob(repo, xmlProcessing, services.DailyProcessing)

	err := job.processRecord(context.Background(), record)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database update failed")

	repo.AssertExpectations(t)
}

func TestDiscogsProcessingJob_CompleteProcessing(t *testing.T) {
	// Setup mocks
	repo := &MockDiscogsDataProcessingRepository{}
	xmlProcessing := &MockXMLProcessingService{}

	record := &models.DiscogsDataProcessing{
		ID:        uuid.New(),
		YearMonth: "2025-01",
		Status:    models.ProcessingStatusProcessing,
	}

	// Mock repository calls
	repo.On("Update", mock.Anything, mock.AnythingOfType("*models.DiscogsDataProcessing")).Return(nil)

	job := NewDiscogsProcessingJob(repo, xmlProcessing, services.DailyProcessing)

	err := job.completeProcessing(context.Background(), record, "2025-01")
	assert.NoError(t, err)
	assert.Equal(t, models.ProcessingStatusCompleted, record.Status)
	assert.NotNil(t, record.ProcessingCompletedAt)

	repo.AssertExpectations(t)
}

func TestDiscogsProcessingJob_Constructor(t *testing.T) {
	repo := &MockDiscogsDataProcessingRepository{}
	xmlProcessing := &MockXMLProcessingService{}
	schedule := services.DailyProcessing

	job := NewDiscogsProcessingJob(repo, xmlProcessing, schedule)

	assert.NotNil(t, job)
	assert.Equal(t, repo, job.repo)
	assert.Equal(t, xmlProcessing, job.xmlProcessing)
	assert.Equal(t, schedule, job.schedule)
	assert.Equal(t, "DiscogsDailyProcessingCheck", job.Name())
}