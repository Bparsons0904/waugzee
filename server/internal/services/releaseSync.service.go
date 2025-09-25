package services

import (
	"context"
	"fmt"
	"time"
	"waugzee/internal/database"
	"waugzee/internal/events"
	"waugzee/internal/logger"
	. "waugzee/internal/models"
	"waugzee/internal/repositories"

	"github.com/google/uuid"
)

type ReleaseSyncService struct {
	log                logger.Logger
	eventBus           *events.EventBus
	repos              repositories.Repository
	db                 database.DB
	discogsRateLimiter *DiscogsRateLimiterService
}

func NewReleaseSyncService(
	eventBus *events.EventBus,
	repos repositories.Repository,
	db database.DB,
	discogsRateLimiter *DiscogsRateLimiterService,
) *ReleaseSyncService {
	return &ReleaseSyncService{
		log:                logger.New("ReleaseSyncService"),
		eventBus:           eventBus,
		repos:              repos,
		db:                 db,
		discogsRateLimiter: discogsRateLimiter,
	}
}

// RequestMissingReleases initiates API requests for missing releases
func (rs *ReleaseSyncService) RequestMissingReleases(
	ctx context.Context,
	user *User,
	missingReleaseIDs []int64,
	syncStateID string,
) error {
	log := rs.log.Function("RequestMissingReleases")

	if len(missingReleaseIDs) == 0 {
		return nil
	}

	if user.Configuration == nil || user.Configuration.DiscogsToken == nil ||
		*user.Configuration.DiscogsToken == "" {
		return log.ErrMsg("user does not have a Discogs token")
	}

	log.Info("Requesting missing releases",
		"userID", user.ID,
		"missingCount", len(missingReleaseIDs),
		"syncStateID", syncStateID)

	// Track request IDs to update sync state
	requestIDs := make([]string, 0, len(missingReleaseIDs))

	// Request each missing release
	for _, releaseID := range missingReleaseIDs {
		requestID := uuid.New().String()

		metadata := RequestMetadata{
			UserID:       user.ID,
			RequestID:    requestID,
			RequestType:  "release",
			Timestamp:    time.Now(),
			DiscogsToken: *user.Configuration.DiscogsToken,
		}

		// Store request metadata in cache
		if err := database.NewCacheBuilder(rs.db.Cache.ClientAPI, requestID).
			WithHashPattern(API_HASH).
			WithStruct(metadata).
			WithTTL(APIRequestTTL).
			WithContext(ctx).
			Set(); err != nil {
			log.Warn("Failed to store request metadata", "error", err, "requestID", requestID)
			continue
		}

		// Check rate limit before making API request
		if err := rs.discogsRateLimiter.CheckUserRateLimit(ctx, user.ID); err != nil {
			// Clean up cache entry since we can't proceed
			_ = database.NewCacheBuilder(rs.db.Cache.ClientAPI, requestID).
				WithHashPattern(API_HASH).
				WithContext(ctx).
				Delete()
			log.Warn("Rate limit check failed, skipping release",
				"releaseID", releaseID, "error", err)
			continue
		}

		// Create API request message
		fullURL := fmt.Sprintf("%s/releases/%d", DiscogsAPIBaseURL, releaseID)
		message := events.Message{
			ID:      requestID,
			Service: events.API,
			Event:   "api_request",
			UserID:  user.ID.String(),
			Payload: map[string]any{
				"requestId":    requestID,
				"requestType":  "release",
				"releaseId":    releaseID,
				"syncStateId":  syncStateID,
				"url":          fullURL,
				"method":       "GET",
				"headers": map[string]string{
					"Authorization": fmt.Sprintf("Discogs token=%s", *user.Configuration.DiscogsToken),
				},
				"callbackService": "orchestration",
				"callbackEvent":   "api_response",
			},
			Timestamp: time.Now(),
		}

		// Publish API request
		if err := rs.eventBus.Publish(events.WEBSOCKET, "user", message); err != nil {
			// Clean up cache entry since we can't proceed
			_ = database.NewCacheBuilder(rs.db.Cache.ClientAPI, requestID).
				WithHashPattern(API_HASH).
				WithContext(ctx).
				Delete()
			log.Warn("Failed to publish API request",
				"releaseID", releaseID, "error", err)
			continue
		}

		requestIDs = append(requestIDs, requestID)
		log.Debug("Requested missing release",
			"releaseID", releaseID,
			"requestID", requestID)
	}

	// Update sync state with pending request IDs
	if len(requestIDs) > 0 {
		err := rs.updateSyncStateWithPendingRequests(ctx, syncStateID, requestIDs)
		if err != nil {
			log.Warn("Failed to update sync state with pending requests", "error", err)
		}
	}

	return nil
}

// ProcessReleaseResponse handles API response for individual release requests
func (rs *ReleaseSyncService) ProcessReleaseResponse(
	ctx context.Context,
	metadata RequestMetadata,
	responseData map[string]any,
) error {
	log := rs.log.Function("ProcessReleaseResponse")

	// Process the release API response using the same pattern as other services
	discogsReleaseResponse, err := processDiscogsAPIResponse[DiscogsReleaseResponse](
		log, responseData, metadata, "release")
	if err != nil {
		log.Warn("Failed to process release response", "error", err)
		return nil // Don't return error as this might be an expected API failure
	}

	if discogsReleaseResponse == nil || discogsReleaseResponse.Data == nil {
		log.Warn("Empty release response data")
		return nil
	}

	releaseData := discogsReleaseResponse.Data

	// Create Release model from API response
	release := &Release{
		BaseDiscogModel: BaseDiscogModel{
			ID: releaseData.ID,
		},
		Title:      releaseData.Title,
		LastSynced: &metadata.Timestamp,
	}

	// Set optional fields
	if releaseData.Year > 0 {
		release.Year = &releaseData.Year
	}
	if releaseData.Country != "" {
		release.Country = &releaseData.Country
	}
	if releaseData.Thumb != "" {
		release.Thumb = &releaseData.Thumb
	}
	if releaseData.ResourceURL != "" {
		release.ResourceURL = &releaseData.ResourceURL
	}
	if releaseData.URI != "" {
		release.URI = &releaseData.URI
	}
	if releaseData.MasterID > 0 {
		release.MasterID = &releaseData.MasterID
	}

	// Set format (default to vinyl)
	release.Format = FormatVinyl

	// TODO: Store additional data like tracks, images, videos in Data JSONB field
	// This would require defining the proper struct for the JSONB data

	// Save release to database
	err = rs.repos.Release.UpsertBatch(ctx, rs.db.SQLWithContext(ctx), []*Release{release})
	if err != nil {
		return log.Err("Failed to save release from API response", err,
			"releaseID", releaseData.ID)
	}

	log.Info("Successfully processed release from API response",
		"releaseID", releaseData.ID,
		"title", releaseData.Title)

	return nil
}

// DiscogsReleaseResponse represents the API response structure for a single release
type DiscogsReleaseResponse struct {
	Data *DiscogsReleaseData `json:"data"`
}

// DiscogsReleaseData represents the release data from Discogs API
type DiscogsReleaseData struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Year        int    `json:"year"`
	Country     string `json:"country"`
	Thumb       string `json:"thumb"`
	ResourceURL string `json:"resource_url"`
	URI         string `json:"uri"`
	MasterID    int64  `json:"master_id"`
	// Add other fields as needed based on Discogs API response
}

// updateSyncStateWithPendingRequests adds pending request IDs to the collection sync state
func (rs *ReleaseSyncService) updateSyncStateWithPendingRequests(
	ctx context.Context,
	syncStateID string,
	requestIDs []string,
) error {
	log := rs.log.Function("updateSyncStateWithPendingRequests")

	const COLLECTION_SYNC_HASH = "collection_sync"

	// Get current sync state as raw data to avoid import cycles
	var syncStateData map[string]any
	found, err := database.NewCacheBuilder(rs.db.Cache.ClientAPI, syncStateID).
		WithHashPattern(COLLECTION_SYNC_HASH).
		WithContext(ctx).
		Get(&syncStateData)
	if err != nil {
		return log.Err("failed to get sync state", err)
	}

	if !found {
		log.Warn("Sync state not found, cannot update pending requests", "syncStateID", syncStateID)
		return nil
	}

	// Initialize or update pending requests
	pendingRequests := make(map[string]bool)

	// Get existing pending requests
	if existingRaw, exists := syncStateData["PendingReleaseRequests"]; exists {
		if existingMap, ok := existingRaw.(map[string]any); ok {
			for k := range existingMap {
				pendingRequests[k] = true
			}
		}
	}

	// Add new request IDs
	for _, requestID := range requestIDs {
		pendingRequests[requestID] = true
	}

	// Update sync state
	syncStateData["PendingReleaseRequests"] = pendingRequests

	err = database.NewCacheBuilder(rs.db.Cache.ClientAPI, syncStateID).
		WithHashPattern(COLLECTION_SYNC_HASH).
		WithStruct(syncStateData).
		WithTTL(30*time.Minute).
		WithContext(ctx).
		Set()
	if err != nil {
		return log.Err("failed to update sync state with pending requests", err)
	}

	log.Info("Updated sync state with pending release requests",
		"syncStateID", syncStateID,
		"newRequests", len(requestIDs),
		"totalPending", len(pendingRequests))

	return nil
}