package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"waugzee/internal/database"
	"waugzee/internal/events"
	"waugzee/internal/logger"
	. "waugzee/internal/models"
	"waugzee/internal/repositories"

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
	log                logger.Logger
	eventBus           *events.EventBus
	cache              valkey.Client
	repos              repositories.Repository
	transactionService *TransactionService
	foldersService     *FoldersService
}

func NewOrchestrationService(
	eventBus *events.EventBus,
	repos repositories.Repository,
	transactionService *TransactionService,
) *OrchestrationService {
	log := logger.New("OrchestrationService")
	foldersService := NewFoldersService(eventBus, repos, transactionService)
	return &OrchestrationService{
		log:                log,
		eventBus:           eventBus,
		cache:              transactionService.db.Cache.General,
		repos:              repos,
		transactionService: transactionService,
		foldersService:     foldersService,
	}
}

func (o *OrchestrationService) GetUserFolders(
	ctx context.Context,
	user *User,
) (string, error) {
	return o.foldersService.RequestUserFolders(ctx, user)
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
		return log.ErrMsg("request metadata not found or expired for requestID: " + requestID)
	}

	if metadata.RequestID != requestID {
		return log.ErrMsg(fmt.Sprintf(
			"request ID mismatch: expected %s, received %s",
			metadata.RequestID,
			requestID,
		))
	}

	err = database.NewCacheBuilder(o.cache, requestID).
		WithHashPattern(API_HASH).
		WithContext(ctx).
		Delete()
	if err != nil {
		log.Err("failed to cleanup cache entry", err, "requestID", requestID)
	}

	log.Info("API response processed successfully",
		"requestID", requestID,
		"requestType", metadata.RequestType,
		"userID", metadata.UserID)

	switch metadata.RequestType {
	case "folders":
		err = o.foldersService.ProcessFoldersResponse(ctx, metadata, responseData)
	case "folder_releases":
		err = o.foldersService.ProcessFolderReleasesResponse(ctx, metadata, responseData)
	default:
		return log.ErrMsg("unknown request type: " + metadata.RequestType)
	}

	if err != nil {
		return log.Err("failed to process response", err, "requestType", metadata.RequestType)
	}

	return nil
}

// processDiscogsAPIResponse is a generic function to handle common Discogs API response patterns
func processDiscogsAPIResponse[T any](
	log logger.Logger,
	responseData map[string]any,
	metadata RequestMetadata,
	responseType string,
) (*T, error) {
	// Check if response contains error
	if errorMsg, exists := responseData["error"]; exists {
		log.Err("API request failed", fmt.Errorf("%v", errorMsg),
			"responseType", responseType,
			"userID", metadata.UserID,
			"requestID", metadata.RequestID)
		return nil, fmt.Errorf("API request failed: %v", errorMsg)
	}

	// Extract data from response
	data, exists := responseData["data"]
	if !exists {
		return nil, log.ErrMsg(fmt.Sprintf("missing data field in %s response", responseType))
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

