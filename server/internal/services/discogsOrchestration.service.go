package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"waugzee/internal/logger"
	"waugzee/internal/models"
	"waugzee/internal/repositories"
	"waugzee/internal/types"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type DiscogsOrchestrationService interface {
	InitiateCollectionSync(
		ctx context.Context,
		userID uuid.UUID,
		syncType models.SyncType,
		fullSync bool,
		pageLimit *int,
	) (*models.DiscogsCollectionSync, error)
	CreateApiRequestQueue(
		ctx context.Context,
		syncSession *models.DiscogsCollectionSync,
		discogsToken string,
	) error
	ProcessApiResponse(ctx context.Context, requestID string, response *types.ApiResponse) error
	GetSyncProgress(ctx context.Context, sessionID string) (*SyncProgress, error)
	CancelSync(ctx context.Context, sessionID string) error
	ResumeSync(ctx context.Context, sessionID string) error
	PauseSync(ctx context.Context, sessionID string) error
	GetActiveSyncs(ctx context.Context, userID uuid.UUID) ([]models.DiscogsCollectionSync, error)
	GetPausedSyncs(ctx context.Context, userID uuid.UUID) ([]models.DiscogsCollectionSync, error)
	HandleSyncDisconnection(ctx context.Context, userID uuid.UUID) error
	HandleSyncReconnection(ctx context.Context, userID uuid.UUID) error
	SetWebSocketSender(sender types.WebSocketSender)
}

type SyncProgress struct {
	SessionID         string    `json:"sessionId"`
	Status            string    `json:"status"`
	SyncType          string    `json:"syncType"`
	TotalRequests     int       `json:"totalRequests"`
	CompletedRequests int       `json:"completedRequests"`
	FailedRequests    int       `json:"failedRequests"`
	PercentComplete   float64   `json:"percentComplete"`
	EstimatedTimeLeft *string   `json:"estimatedTimeLeft,omitempty"`
	StartedAt         time.Time `json:"startedAt"`
	LastPage          *int      `json:"lastPage,omitempty"`
	TotalPages        *int      `json:"totalPages,omitempty"`
	CurrentAction     string    `json:"currentAction"`
}

type discogsOrchestrationService struct {
	syncRepo        repositories.DiscogsCollectionSyncRepository
	requestRepo     repositories.DiscogsApiRequestRepository
	userRepo        repositories.UserRepository
	rateLimitSvc    DiscogsRateLimitService
	websocketSender types.WebSocketSender
	log             logger.Logger
	maxRetries      int
	requestTimeout  time.Duration
}

func NewDiscogsOrchestrationService(
	syncRepo repositories.DiscogsCollectionSyncRepository,
	requestRepo repositories.DiscogsApiRequestRepository,
	userRepo repositories.UserRepository,
	rateLimitSvc DiscogsRateLimitService,
	websocketSender types.WebSocketSender,
) DiscogsOrchestrationService {
	return &discogsOrchestrationService{
		syncRepo:        syncRepo,
		requestRepo:     requestRepo,
		userRepo:        userRepo,
		rateLimitSvc:    rateLimitSvc,
		websocketSender: websocketSender,
		log:             logger.New("DiscogsOrchestrationService"),
		maxRetries:      3,
		requestTimeout:  30 * time.Second,
	}
}

func (s *discogsOrchestrationService) SetWebSocketSender(sender types.WebSocketSender) {
	s.websocketSender = sender
}

func (s *discogsOrchestrationService) InitiateCollectionSync(
	ctx context.Context,
	userID uuid.UUID,
	syncType models.SyncType,
	fullSync bool,
	pageLimit *int,
) (*models.DiscogsCollectionSync, error) {
	log := s.log.Function("InitiateCollectionSync")

	// Check for existing active syncs
	activeSyncs, err := s.syncRepo.GetActiveByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check active syncs: %w", err)
	}

	if len(activeSyncs) > 0 {
		return nil, fmt.Errorf("user already has an active sync session")
	}

	// Get user to validate Discogs token
	user, err := s.userRepo.GetByID(ctx, userID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if user.DiscogsToken == nil || *user.DiscogsToken == "" {
		return nil, fmt.Errorf("user does not have a Discogs token configured")
	}

	// Create sync session
	sessionID := uuid.New().String()
	syncSession := &models.DiscogsCollectionSync{
		UserID:    userID,
		SessionID: sessionID,
		Status:    models.SyncStatusInitiated,
		SyncType:  syncType,
		FullSync:  fullSync,
		PageLimit: pageLimit,
	}

	if err := s.syncRepo.Create(ctx, syncSession); err != nil {
		return nil, fmt.Errorf("failed to create sync session: %w", err)
	}

	log.Info("Sync session created", "sessionID", sessionID, "userID", userID, "syncType", syncType)

	// Create API request queue
	go func() {
		if err := s.CreateApiRequestQueue(context.Background(), syncSession, *user.DiscogsToken); err != nil {
			_ = log.Error("Failed to create API request queue", "error", err)
			s.markSyncAsFailed(context.Background(), sessionID, err.Error())
		}
	}()

	return syncSession, nil
}

func (s *discogsOrchestrationService) CreateApiRequestQueue(
	ctx context.Context,
	syncSession *models.DiscogsCollectionSync,
	discogsToken string,
) error {
	log := s.log.Function("CreateApiRequestQueue")

	// Get user to determine username
	user, err := s.userRepo.GetByID(ctx, syncSession.UserID.String())
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// For now, we'll create a basic request to get the first page of collection
	// This would be expanded to handle pagination based on actual Discogs API response
	baseURL := "https://api.discogs.com"
	var endpoint string

	var username string
	if user.Email != nil {
		username = *user.Email
	} else {
		return fmt.Errorf("user has no email configured for Discogs username")
	}

	switch syncSession.SyncType {
	case models.SyncTypeCollection:
		endpoint = fmt.Sprintf(
			"/users/%s/collection/folders/0/releases",
			username,
		) // Using email as username for now
	case models.SyncTypeWantlist:
		endpoint = fmt.Sprintf("/users/%s/wants", username)
	default:
		return fmt.Errorf("unsupported sync type: %s", syncSession.SyncType)
	}

	// Create initial request
	requestID := uuid.New().String()
	url := fmt.Sprintf("%s%s?page=1&per_page=100", baseURL, endpoint)

	headers := map[string]interface{}{
		"Authorization": fmt.Sprintf("Discogs token=%s", discogsToken),
		"User-Agent":    "Waugzee/1.0",
	}

	headersJSON, err := json.Marshal(headers)
	if err != nil {
		return fmt.Errorf("failed to marshal headers: %w", err)
	}

	apiRequest := &models.DiscogsApiRequest{
		UserID:        syncSession.UserID,
		SyncSessionID: syncSession.ID,
		RequestID:     requestID,
		URL:           url,
		Method:        "GET",
		Headers:       datatypes.JSON(headersJSON),
		Status:        models.RequestStatusPending,
	}

	if err := s.requestRepo.Create(ctx, apiRequest); err != nil {
		return fmt.Errorf("failed to create API request: %w", err)
	}

	// Update sync session
	syncSession.TotalRequests = 1
	syncSession.MarkAsInProgress()
	if err := s.syncRepo.Update(ctx, syncSession); err != nil {
		return fmt.Errorf("failed to update sync session: %w", err)
	}

	log.Info("API request queue created", "sessionID", syncSession.SessionID, "totalRequests", 1)

	// Send first API request to client
	s.sendApiRequestToClient(ctx, syncSession.UserID, apiRequest)

	// Send progress update
	s.sendProgressUpdate(ctx, syncSession)

	return nil
}

func (s *discogsOrchestrationService) ProcessApiResponse(
	ctx context.Context,
	requestID string,
	response *types.ApiResponse,
) error {
	log := s.log.Function("ProcessApiResponse")

	// Get the request
	apiRequest, err := s.requestRepo.GetByRequestID(ctx, requestID)
	if err != nil {
		return fmt.Errorf("failed to get API request: %w", err)
	}

	// Update rate limit info
	if err = s.rateLimitSvc.UpdateRateLimit(ctx, apiRequest.UserID, response.Headers); err != nil {
		log.Warn("Failed to update rate limit", "error", err)
	}

	// Process response
	if response.Error != nil {
		apiRequest.MarkAsFailed(*response.Error)
	} else {
		headersJSON, _ := json.Marshal(response.Headers)
		apiRequest.MarkAsCompleted(response.Status, datatypes.JSON(headersJSON), datatypes.JSON(response.Body))

		// Process pagination if this is the first request
		if apiRequest.URL == s.getFirstPageURL(apiRequest) {
			if err := s.handlePagination(ctx, apiRequest, response.Body); err != nil {
				_ = log.Error("Failed to handle pagination", "error", err)
			}
		}

		// Process the actual data (releases, wants, etc.)
		if err := s.processResponseData(ctx, apiRequest, response.Body); err != nil {
			_ = log.Error("Failed to process response data", "error", err)
		}
	}

	// Save request
	if err := s.requestRepo.Update(ctx, apiRequest); err != nil {
		return fmt.Errorf("failed to update API request: %w", err)
	}

	// Update sync progress
	syncSession, err := s.syncRepo.GetBySessionID(ctx, apiRequest.SyncSession.SessionID)
	if err != nil {
		return fmt.Errorf("failed to get sync session: %w", err)
	}

	completed, _ := s.requestRepo.CountByStatus(ctx, syncSession.ID, models.RequestStatusCompleted)
	failed, _ := s.requestRepo.CountByStatus(ctx, syncSession.ID, models.RequestStatusFailed)

	syncSession.UpdateProgress(int(completed), int(failed))

	// Check if sync is complete
	if int(completed+failed) >= syncSession.TotalRequests {
		syncSession.MarkAsCompleted()
		s.sendSyncCompleteMessage(ctx, syncSession.UserID, syncSession.SessionID)
	}

	if err := s.syncRepo.Update(ctx, syncSession); err != nil {
		return fmt.Errorf("failed to update sync session: %w", err)
	}

	// Send progress update
	s.sendProgressUpdate(ctx, syncSession)

	// Send next request if available
	s.sendNextPendingRequest(ctx, syncSession)

	log.Info("API response processed", "requestID", requestID, "status", response.Status)

	return nil
}

func (s *discogsOrchestrationService) GetSyncProgress(
	ctx context.Context,
	sessionID string,
) (*SyncProgress, error) {
	syncSession, err := s.syncRepo.GetBySessionID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get sync session: %w", err)
	}

	progress := &SyncProgress{
		SessionID:         syncSession.SessionID,
		Status:            string(syncSession.Status),
		SyncType:          string(syncSession.SyncType),
		TotalRequests:     syncSession.TotalRequests,
		CompletedRequests: syncSession.CompletedRequests,
		FailedRequests:    syncSession.FailedRequests,
		PercentComplete:   syncSession.GetPercentComplete(),
		StartedAt:         syncSession.StartedAt,
		LastPage:          syncSession.LastPage,
		TotalPages:        syncSession.TotalPages,
		CurrentAction:     s.getCurrentAction(syncSession),
	}

	if timeLeft := syncSession.GetEstimatedTimeLeft(); timeLeft != nil {
		timeLeftStr := timeLeft.Round(time.Second).String()
		progress.EstimatedTimeLeft = &timeLeftStr
	}

	return progress, nil
}

func (s *discogsOrchestrationService) CancelSync(ctx context.Context, sessionID string) error {
	syncSession, err := s.syncRepo.GetBySessionID(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get sync session: %w", err)
	}

	syncSession.MarkAsCancelled()
	if err := s.syncRepo.Update(ctx, syncSession); err != nil {
		return fmt.Errorf("failed to update sync session: %w", err)
	}

	// Send cancellation message
	s.sendSyncCancelledMessage(ctx, syncSession.UserID, sessionID)

	return nil
}

func (s *discogsOrchestrationService) ResumeSync(ctx context.Context, sessionID string) error {
	syncSession, err := s.syncRepo.GetBySessionID(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get sync session: %w", err)
	}

	if !syncSession.CanResume() {
		return fmt.Errorf("sync session cannot be resumed")
	}

	syncSession.MarkAsInProgress()
	if err := s.syncRepo.Update(ctx, syncSession); err != nil {
		return fmt.Errorf("failed to update sync session: %w", err)
	}

	// Send next pending request
	s.sendNextPendingRequest(ctx, syncSession)

	return nil
}

func (s *discogsOrchestrationService) PauseSync(ctx context.Context, sessionID string) error {
	syncSession, err := s.syncRepo.GetBySessionID(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get sync session: %w", err)
	}

	syncSession.MarkAsPaused()
	if err := s.syncRepo.Update(ctx, syncSession); err != nil {
		return fmt.Errorf("failed to update sync session: %w", err)
	}

	return nil
}

func (s *discogsOrchestrationService) GetActiveSyncs(
	ctx context.Context,
	userID uuid.UUID,
) ([]models.DiscogsCollectionSync, error) {
	return s.syncRepo.GetActiveByUserID(ctx, userID)
}

func (s *discogsOrchestrationService) GetPausedSyncs(
	ctx context.Context,
	userID uuid.UUID,
) ([]models.DiscogsCollectionSync, error) {
	return s.syncRepo.GetPausedByUserID(ctx, userID)
}

func (s *discogsOrchestrationService) HandleSyncDisconnection(
	ctx context.Context,
	userID uuid.UUID,
) error {
	activeSyncs, err := s.GetActiveSyncs(ctx, userID)
	if err != nil {
		return err
	}

	for _, sync := range activeSyncs {
		if err := s.PauseSync(ctx, sync.SessionID); err != nil {
			_ = s.log.Error(
				"Failed to pause sync on disconnection",
				"sessionID",
				sync.SessionID,
				"error",
				err,
			)
		}
	}

	return nil
}

func (s *discogsOrchestrationService) HandleSyncReconnection(
	ctx context.Context,
	userID uuid.UUID,
) error {
	pausedSyncs, err := s.GetPausedSyncs(ctx, userID)
	if err != nil {
		return err
	}

	for _, sync := range pausedSyncs {
		if err := s.ResumeSync(ctx, sync.SessionID); err != nil {
			_ = s.log.Error(
				"Failed to resume sync on reconnection",
				"sessionID",
				sync.SessionID,
				"error",
				err,
			)
		}
	}

	return nil
}

// Helper methods

func (s *discogsOrchestrationService) sendApiRequestToClient(
	ctx context.Context,
	userID uuid.UUID,
	apiRequest *models.DiscogsApiRequest,
) {
	var headers map[string]interface{}
	if err := json.Unmarshal(apiRequest.Headers, &headers); err != nil {
		_ = s.log.Error("Failed to unmarshal headers", "error", err)
		return
	}

	message := &types.WebSocketMessageImpl{
		ID:      uuid.New().String(),
		Type:    "discogs_api_request",
		Channel: "sync",
		Action:  "make_request",
		Data: map[string]interface{}{
			"requestId": apiRequest.RequestID,
			"url":       apiRequest.URL,
			"method":    apiRequest.Method,
			"headers":   headers,
		},
		Timestamp: time.Now(),
	}

	s.websocketSender.SendMessageToUser(userID, message)
	apiRequest.MarkAsSent()
	_ = s.requestRepo.Update(ctx, apiRequest)
}

func (s *discogsOrchestrationService) sendProgressUpdate(
	ctx context.Context,
	syncSession *models.DiscogsCollectionSync,
) {
	progress, err := s.GetSyncProgress(ctx, syncSession.SessionID)
	if err != nil {
		_ = s.log.Error("Failed to get sync progress", "error", err)
		return
	}

	message := &types.WebSocketMessageImpl{
		ID:        uuid.New().String(),
		Type:      "sync_progress",
		Channel:   "sync",
		Action:    "progress_update",
		Data:      map[string]interface{}{"progress": progress},
		Timestamp: time.Now(),
	}

	s.websocketSender.SendMessageToUser(syncSession.UserID, message)
}

func (s *discogsOrchestrationService) sendSyncCompleteMessage(
	ctx context.Context,
	userID uuid.UUID,
	sessionID string,
) {
	message := &types.WebSocketMessageImpl{
		ID:        uuid.New().String(),
		Type:      "sync_complete",
		Channel:   "sync",
		Action:    "sync_complete",
		Data:      map[string]interface{}{"sessionId": sessionID},
		Timestamp: time.Now(),
	}

	s.websocketSender.SendMessageToUser(userID, message)
}

func (s *discogsOrchestrationService) sendSyncCancelledMessage(
	ctx context.Context,
	userID uuid.UUID,
	sessionID string,
) {
	message := &types.WebSocketMessageImpl{
		ID:        uuid.New().String(),
		Type:      "sync_error",
		Channel:   "sync",
		Action:    "sync_cancelled",
		Data:      map[string]interface{}{"sessionId": sessionID, "reason": "cancelled by user"},
		Timestamp: time.Now(),
	}

	s.websocketSender.SendMessageToUser(userID, message)
}

func (s *discogsOrchestrationService) sendNextPendingRequest(
	ctx context.Context,
	syncSession *models.DiscogsCollectionSync,
) {
	if !syncSession.IsActive() {
		return
	}

	pendingRequests, err := s.requestRepo.GetPendingBySyncSession(ctx, syncSession.ID)
	if err != nil || len(pendingRequests) == 0 {
		return
	}

	// Check rate limit before sending
	canMakeRequest, delay, err := s.rateLimitSvc.CanMakeRequest(ctx, syncSession.UserID)
	if err != nil {
		_ = s.log.Error("Failed to check rate limit", "error", err)
		return
	}

	if !canMakeRequest {
		s.log.Info(
			"Rate limit reached, pausing sync",
			"sessionID",
			syncSession.SessionID,
			"delay",
			delay,
		)
		_ = s.PauseSync(ctx, syncSession.SessionID)

		// Schedule resume
		time.AfterFunc(delay, func() {
			_ = s.ResumeSync(context.Background(), syncSession.SessionID)
		})
		return
	}

	// Send the next request
	s.sendApiRequestToClient(ctx, syncSession.UserID, &pendingRequests[0])
}

func (s *discogsOrchestrationService) markSyncAsFailed(
	ctx context.Context,
	sessionID string,
	errorMessage string,
) {
	syncSession, err := s.syncRepo.GetBySessionID(ctx, sessionID)
	if err != nil {
		return
	}

	syncSession.MarkAsFailed(errorMessage)
	_ = s.syncRepo.Update(ctx, syncSession)

	message := &types.WebSocketMessageImpl{
		ID:        uuid.New().String(),
		Type:      "sync_error",
		Channel:   "sync",
		Action:    "sync_failed",
		Data:      map[string]interface{}{"sessionId": sessionID, "error": errorMessage},
		Timestamp: time.Now(),
	}

	s.websocketSender.SendMessageToUser(syncSession.UserID, message)
}

func (s *discogsOrchestrationService) getFirstPageURL(apiRequest *models.DiscogsApiRequest) string {
	// This would need to be implemented based on the URL pattern
	// For now, just return the URL if it contains page=1
	return apiRequest.URL
}

func (s *discogsOrchestrationService) handlePagination(
	ctx context.Context,
	apiRequest *models.DiscogsApiRequest,
	responseBody json.RawMessage,
) error {
	// Parse response to get pagination info
	var response struct {
		Pagination struct {
			Page    int `json:"page"`
			Pages   int `json:"pages"`
			PerPage int `json:"per_page"`
		} `json:"pagination"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return err
	}

	syncSession, err := s.syncRepo.GetBySessionID(ctx, apiRequest.SyncSession.SessionID)
	if err != nil {
		return err
	}

	// Update total pages
	syncSession.TotalPages = &response.Pagination.Pages
	syncSession.LastPage = &response.Pagination.Page

	// Create additional requests for remaining pages
	if response.Pagination.Pages > 1 {
		additionalRequests := response.Pagination.Pages - 1

		// Apply page limit if set
		if syncSession.PageLimit != nil && *syncSession.PageLimit < response.Pagination.Pages {
			additionalRequests = *syncSession.PageLimit - 1
		}

		syncSession.TotalRequests = 1 + additionalRequests

		// Create requests for pages 2 onwards
		for page := 2; page <= 1+additionalRequests; page++ {
			_ = s.createPageRequest(ctx, apiRequest, page)
		}
	}

	return s.syncRepo.Update(ctx, syncSession)
}

func (s *discogsOrchestrationService) createPageRequest(
	ctx context.Context,
	originalRequest *models.DiscogsApiRequest,
	page int,
) error {
	// Parse original URL and replace page parameter
	url := fmt.Sprintf("%s&page=%d", originalRequest.URL, page) // Simplified URL manipulation

	requestID := uuid.New().String()
	pageRequest := &models.DiscogsApiRequest{
		UserID:        originalRequest.UserID,
		SyncSessionID: originalRequest.SyncSessionID,
		RequestID:     requestID,
		URL:           url,
		Method:        originalRequest.Method,
		Headers:       originalRequest.Headers,
		Status:        models.RequestStatusPending,
	}

	return s.requestRepo.Create(ctx, pageRequest)
}

func (s *discogsOrchestrationService) processResponseData(
	ctx context.Context,
	apiRequest *models.DiscogsApiRequest,
	responseBody json.RawMessage,
) error {
	// This would implement the actual data processing logic
	// For now, we'll just log that we received data
	s.log.Info(
		"Processing response data",
		"requestID",
		apiRequest.RequestID,
		"dataSize",
		len(responseBody),
	)
	return nil
}

func (s *discogsOrchestrationService) getCurrentAction(
	syncSession *models.DiscogsCollectionSync,
) string {
	if syncSession.LastPage != nil && syncSession.TotalPages != nil {
		return fmt.Sprintf(
			"Processing page %d of %d",
			*syncSession.LastPage,
			*syncSession.TotalPages,
		)
	}
	return "Processing requests..."
}

