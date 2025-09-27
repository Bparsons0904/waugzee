package services

import (
	"context"
	"encoding/json"
	"time"
	"waugzee/internal/database"
	"waugzee/internal/events"
	"waugzee/internal/logger"
	. "waugzee/internal/models"
	"waugzee/internal/repositories"

	"github.com/google/uuid"
	"github.com/valkey-io/valkey-go"
)

type RequestMetadata struct {
	UserID       uuid.UUID `json:"userId"`
	RequestID    string    `json:"requestId"`
	RequestType  string    `json:"requestType"`
	Timestamp    time.Time `json:"timestamp"`
	DiscogsToken string    `json:"discogsToken,omitempty"`
}

type OrchestrationService struct {
	log                logger.Logger
	eventBus           *events.EventBus
	cache              valkey.Client
	repos              repositories.Repository
	transactionService *TransactionService
	foldersService     *FoldersService
	releaseSyncService *ReleaseSyncService
	discogsRateLimiter *DiscogsRateLimiterService
}

func NewOrchestrationService(
	eventBus *events.EventBus,
	repos repositories.Repository,
	db database.DB,
	transactionService *TransactionService,
	discogsRateLimiter *DiscogsRateLimiterService,
) *OrchestrationService {
	log := logger.New("OrchestrationService")
	folderDataExtractionService := NewFolderDataExtractionService(repos)
	foldersService := NewFoldersService(
		eventBus,
		repos,
		db,
		transactionService,
		folderDataExtractionService,
		discogsRateLimiter,
	)
	releaseSyncService := NewReleaseSyncService(eventBus, repos, db, discogsRateLimiter)
	return &OrchestrationService{
		log:                log,
		eventBus:           eventBus,
		cache:              transactionService.db.Cache.ClientAPI,
		repos:              repos,
		transactionService: transactionService,
		foldersService:     foldersService,
		releaseSyncService: releaseSyncService,
		discogsRateLimiter: discogsRateLimiter,
	}
}

func (o *OrchestrationService) GetUserFolders(
	ctx context.Context,
	user *User,
) (string, error) {
	return o.foldersService.RequestUserFolders(ctx, user)
}

// SyncUserFoldersAndCollection performs comprehensive sync: discovers folders then syncs collection.
// This is the main entry point for collection synchronization.
func (o *OrchestrationService) SyncUserFoldersAndCollection(
	ctx context.Context,
	user *User,
) error {
	log := o.log.Function("SyncUserFoldersAndCollection")

	if user == nil {
		return log.ErrMsg("user cannot be nil")
	}

	if user.Configuration == nil || user.Configuration.DiscogsToken == nil ||
		*user.Configuration.DiscogsToken == "" {
		return log.ErrMsg("user does not have a Discogs token configured")
	}

	// Step 1: Request folder discovery (async - will trigger folder processing)
	_, err := o.foldersService.RequestUserFolders(ctx, user)
	if err != nil {
		return log.Err("failed to initiate folder discovery", err)
	}

	// Step 2: Start collection sync for all user folders
	// This will process folders 1+ and coordinate with folder responses
	err = o.foldersService.SyncAllUserFolders(ctx, user)
	if err != nil {
		return log.Err("failed to initiate collection sync", err)
	}

	return nil
}

// HandleAPIResponse processes API responses from the client-as-proxy pattern.
// It retrieves request metadata from cache, routes responses to appropriate services,
// and handles request cleanup.
func (o *OrchestrationService) HandleAPIResponse(
	ctx context.Context,
	responseData map[string]any,
) error {
	log := o.log.Function("HandleAPIResponse")

	if responseData == nil {
		return log.ErrMsg("responseData cannot be nil")
	}

	requestID, ok := responseData["requestId"].(string)
	if !ok || requestID == "" {
		return log.ErrMsg("missing or invalid requestId in response")
	}

	var metadata RequestMetadata
	found, err := database.NewCacheBuilder(o.cache, requestID).
		WithHashPattern(API_HASH).
		WithContext(ctx).
		Get(&metadata)
	if err != nil {
		return log.Err("failed to retrieve request metadata from cache", err)
	}

	if !found {
		return log.ErrMsg("request metadata not found or expired for requestID: " + requestID)
	}

	if metadata.RequestID != requestID {
		return log.ErrMsg(
			"request ID mismatch: expected " + metadata.RequestID + ", received " + requestID,
		)
	}

	err = database.NewCacheBuilder(o.cache, requestID).
		WithHashPattern(API_HASH).
		WithContext(ctx).
		Delete()
	if err != nil {
		log.Err("failed to cleanup cache entry", err, "requestID", requestID)
	}

	switch metadata.RequestType {
	case "folders":
		err = o.foldersService.ProcessFoldersResponse(ctx, metadata, responseData)
	case "folder_releases":
		err = o.foldersService.ProcessFolderReleasesResponse(ctx, metadata, responseData)
	case "release":
		err = o.handleReleaseResponse(ctx, metadata, responseData)
	default:
		return log.ErrMsg("unknown request type: " + metadata.RequestType)
	}

	if err != nil {
		return log.Err("failed to process response", err, "requestType", metadata.RequestType)
	}

	return nil
}

// processDiscogsAPIResponse is a generic function to handle common Discogs API response patterns.
// It validates the response, extracts data, and unmarshals to the target type.
func processDiscogsAPIResponse[T any](
	log logger.Logger,
	responseData map[string]any,
	metadata RequestMetadata,
	responseType string,
) (*T, error) {
	// Check if response contains error
	if errorMsg, exists := responseData["error"]; exists {
		return nil, log.Error("API request failed",
			"responseType", responseType,
			"userID", metadata.UserID,
			"requestID", metadata.RequestID,
			"error", errorMsg)
	}

	data, exists := responseData["data"]
	if !exists {
		return nil, log.ErrMsg("missing data field in " + responseType + " response")
	}

	// Marshal and unmarshal to target type
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return nil, log.Err("failed to marshal response data", err, "responseType", responseType)
	}

	var result T
	if err := json.Unmarshal(dataJSON, &result); err != nil {
		return nil, log.Err("failed to unmarshal response", err, "responseType", responseType)
	}

	return &result, nil
}

// handleReleaseResponse processes individual release API responses and updates sync state.
// It delegates to ReleaseSyncService for actual release processing and manages collection sync state.
func (o *OrchestrationService) handleReleaseResponse(
	ctx context.Context,
	metadata RequestMetadata,
	responseData map[string]any,
) error {
	log := o.log.Function("handleReleaseResponse")

	// Process the release through the release sync service
	err := o.releaseSyncService.ProcessReleaseResponse(ctx, metadata, responseData)
	if err != nil {
		log.Warn("Failed to process release response", "error", err)
		// Don't fail completely - just log and continue
	}

	// Extract syncStateId from response data to update collection sync state
	syncStateID, exists := responseData["syncStateId"]
	if !exists {
		log.Warn("No syncStateId in release response - cannot update collection sync state")
		return nil
	}

	syncStateIDStr, ok := syncStateID.(string)
	if !ok {
		log.Warn("Invalid syncStateId type in release response")
		return nil
	}

	// Get current collection sync state as raw data to avoid import cycles
	var syncStateData map[string]any
	found, err := database.NewCacheBuilder(o.cache, syncStateIDStr).
		WithHashPattern("collection_sync").
		WithContext(ctx).
		Get(&syncStateData)
	if err != nil {
		return log.Err("failed to get collection sync state", err)
	}

	if !found {
		// Sync state expired or doesn't exist - this is OK, sync might be complete
		log.Info("Collection sync state not found - sync may be complete",
			"syncStateId", syncStateIDStr)
		return nil
	}

	// Extract pending requests from sync state data
	pendingRequestsRaw, exists := syncStateData["PendingReleaseRequests"]
	if exists {
		if pendingRequestsMap, ok := pendingRequestsRaw.(map[string]any); ok {
			pendingRequests := make(map[string]bool)
			for k := range pendingRequestsMap {
				pendingRequests[k] = true
			}

			// Remove this request from pending requests
			delete(pendingRequests, metadata.RequestID)

			// Check if all release requests are complete
			if len(pendingRequests) == 0 {
				// Update sync state to mark all releases ready
				syncStateData["AllReleasesReady"] = true
				syncStateData["PendingReleaseRequests"] = make(map[string]bool)

				// Get missing releases count for logging
				missingReleases := 0
				if missingRaw, exists := syncStateData["MissingReleaseIDs"]; exists {
					if missingSlice, ok := missingRaw.([]any); ok {
						missingReleases = len(missingSlice)
					}
				}

				log.Info("All release requests completed, sync can proceed",
					"syncStateId", syncStateIDStr,
					"totalMissingReleases", missingReleases)

				// Update sync state and trigger completion directly
				err = database.NewCacheBuilder(o.cache, syncStateIDStr).
					WithHashPattern("collection_sync").
					WithStruct(syncStateData).
					WithTTL(SyncStateTTL).
					WithContext(ctx).
					Set()
				if err != nil {
					log.Warn("Failed to update sync state", "error", err)
					return nil
				}

				// Trigger sync completion directly via folders service
				err = o.foldersService.TriggerSyncCompletion(ctx, metadata.UserID)
				if err != nil {
					log.Warn("Failed to trigger sync completion", "error", err)
				}
			} else {
				// Update sync state with remaining pending requests
				syncStateData["PendingReleaseRequests"] = pendingRequests

				err = database.NewCacheBuilder(o.cache, syncStateIDStr).
					WithHashPattern("collection_sync").
					WithStruct(syncStateData).
					WithTTL(SyncStateTTL).
					WithContext(ctx).
					Set()
				if err != nil {
					log.Warn("Failed to update sync state", "error", err)
				}

				log.Debug("Release request completed, waiting for more",
					"syncStateId", syncStateIDStr,
					"remainingRequests", len(pendingRequests))
			}
		}
	}

	return nil
}
