package services

import (
	"context"
	"fmt"
	"time"
	"waugzee/internal/database"
	"waugzee/internal/events"
	"waugzee/internal/logger"
	. "waugzee/internal/models"

	"github.com/google/uuid"
	"github.com/valkey-io/valkey-go"
)

const (
	API_HASH          = "api_request"
	APIRequestTTL     = 10 * time.Minute
	DiscogsAPIBaseURL = "https://api.discogs.com"
)

type RequestMetadata struct {
	UserID       uuid.UUID `json:"userId"`
	RequestID    string    `json:"requestId"`
	RequestType  string    `json:"requestType"`
	Timestamp    time.Time `json:"timestamp"`
	DiscogsToken string    `json:"discogsToken,omitempty"`
}

type OrchestrationService struct {
	log      logger.Logger
	eventBus *events.EventBus
	cache    valkey.Client
}

func NewOrchestrationService(eventBus *events.EventBus, db database.DB) *OrchestrationService {
	log := logger.New("OrchestrationService")
	return &OrchestrationService{
		log:      log,
		eventBus: eventBus,
		cache:    db.Cache.General,
	}
}

func (o *OrchestrationService) GetUserFolders(
	ctx context.Context,
	user *User,
) (string, error) {
	log := o.log.Function("GetUserFolders")

	if user == nil {
		return "", fmt.Errorf("user cannot be nil")
	}

	if user.DiscogsToken == nil || *user.DiscogsToken == "" {
		return "", fmt.Errorf("user does not have a Discogs token")
	}

	if user.DiscogsUsername == nil || *user.DiscogsUsername == "" {
		return "", fmt.Errorf("user does not have a Discogs username")
	}

	requestID := uuid.New().String()

	metadata := RequestMetadata{
		UserID:       user.ID,
		RequestID:    requestID,
		RequestType:  "folders",
		Timestamp:    time.Now(),
		DiscogsToken: *user.DiscogsToken,
	}

	if err := database.NewCacheBuilder(o.cache, requestID).
		WithHashPattern(API_HASH).
		WithStruct(metadata).
		WithTTL(APIRequestTTL).
		WithContext(ctx).
		Set(); err != nil {
		return "", log.Err("failed to store request metadata in cache", err)
	}

	fullURL := fmt.Sprintf(
		"%s/users/%s/collection/folders",
		DiscogsAPIBaseURL,
		*user.DiscogsUsername,
	)
	message := events.Message{
		ID:      requestID,
		Service: events.API,
		Event:   "api_request",
		UserID:  user.ID.String(),
		Payload: map[string]any{
			"requestId":   requestID,
			"requestType": "folders",
			"url":         fullURL,
			"method":      "GET",
			"headers": map[string]string{
				"Authorization": fmt.Sprintf("Discogs token=%s", *user.DiscogsToken),
			},
			"callbackService": "orchestration",
			"callbackEvent":   "api_response",
		},
		Timestamp: time.Now(),
	}

	if err := o.eventBus.Publish(events.WEBSOCKET, "user", message); err != nil {
		_ = database.NewCacheBuilder(o.cache, requestID).
			WithHashPattern(API_HASH).
			WithContext(ctx).
			Delete()
		return "", log.Err("failed to publish API request event", err)
	}

	return requestID, nil
}

func (o *OrchestrationService) HandleAPIResponse(
	ctx context.Context,
	responseData map[string]any,
) error {
	log := o.log.Function("HandleAPIResponse")

	requestID, ok := responseData["requestId"].(string)
	if !ok || requestID == "" {
		return log.ErrMsg("missing or invalid requestId in response")
	}

	log.Info("Processing API response", "requestID", requestID)

	var metadata RequestMetadata
	found, err := database.NewCacheBuilder(o.cache, requestID).
		WithHashPattern(API_HASH).
		WithContext(ctx).
		Get(&metadata)
	if err != nil {
		return log.Err("failed to retrieve request metadata from cache", err)
	}

	if !found {
		log.Er("Request metadata not found in cache",
			fmt.Errorf("cache miss"),
			"requestID", requestID)
		return fmt.Errorf("request metadata not found or expired for requestID: %s", requestID)
	}

	if metadata.RequestID != requestID {
		return fmt.Errorf(
			"request ID mismatch: expected %s, received %s",
			metadata.RequestID,
			requestID,
		)
	}

	err = database.NewCacheBuilder(o.cache, requestID).
		WithHashPattern(API_HASH).
		WithContext(ctx).
		Delete()
	if err != nil {
		log.Er("failed to cleanup cache entry", err, "requestID", requestID)
	}

	log.Info("API response processed successfully",
		"requestID", requestID,
		"requestType", metadata.RequestType,
		"userID", metadata.UserID)

	switch metadata.RequestType {
	case "folders":
		err = o.processFoldersResponse(ctx, metadata, responseData)
	default:
		return fmt.Errorf("unknown request type: %s", metadata.RequestType)
	}

	if err != nil {
		return log.Err("failed to process response", err, "requestType", metadata.RequestType)
	}

	return nil
}

// processFoldersResponse handles the response for user folders requests
func (o *OrchestrationService) processFoldersResponse(
	ctx context.Context,
	metadata RequestMetadata,
	responseData map[string]any,
) error {
	log := o.log.Function("processFoldersResponse")

	// Check if response contains error
	if errorMsg, exists := responseData["error"]; exists {
		log.Er("Folders API request failed", fmt.Errorf("%v", errorMsg),
			"userID", metadata.UserID,
			"requestID", metadata.RequestID)
		return nil // Don't return error as this is an expected API failure
	}

	// Extract folders data from response
	foldersData, exists := responseData["data"]
	if !exists {
		return log.ErrMsg("missing data field in folders response")
	}

	log.Info("Successfully received folders data",
		"userID", metadata.UserID,
		"requestID", metadata.RequestID,
		"foldersData", foldersData)

	// TODO: Process and store folder data when folder sync feature is implemented

	return nil
}
