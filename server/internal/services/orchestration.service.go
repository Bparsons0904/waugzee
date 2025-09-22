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
	"gorm.io/gorm"
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
}

func NewOrchestrationService(
	eventBus *events.EventBus,
	repos repositories.Repository,
	transactionService *TransactionService,
) *OrchestrationService {
	log := logger.New("OrchestrationService")
	return &OrchestrationService{
		log:                log,
		eventBus:           eventBus,
		cache:              transactionService.db.Cache.General,
		repos:              repos,
		transactionService: transactionService,
	}
}

func (o *OrchestrationService) GetUserFolders(
	ctx context.Context,
	user *User,
) (string, error) {
	log := o.log.Function("GetUserFolders")

	if user == nil {
		return "", log.ErrMsg("user cannot be nil")
	}

	if user.Configuration == nil || user.Configuration.DiscogsToken == nil ||
		*user.Configuration.DiscogsToken == "" {
		return "", log.ErrMsg("user does not have a Discogs token")
	}

	if user.Configuration.DiscogsUsername == nil || *user.Configuration.DiscogsUsername == "" {
		return "", log.ErrMsg("user does not have a Discogs username")
	}

	requestID := uuid.New().String()

	metadata := RequestMetadata{
		UserID:       user.ID,
		RequestID:    requestID,
		RequestType:  "folders",
		Timestamp:    time.Now(),
		DiscogsToken: *user.Configuration.DiscogsToken,
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
		*user.Configuration.DiscogsUsername,
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
				"Authorization": fmt.Sprintf("Discogs token=%s", *user.Configuration.DiscogsToken),
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
		log.Er("API request failed", fmt.Errorf("%v", errorMsg),
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

// extractFolderSyncData extracts business logic for processing folder sync data
func (o *OrchestrationService) extractFolderSyncData(
	folders []*Folder,
) (keepDiscogIDs []int, allFolderDiscogID *int) {
	keepDiscogIDs = make([]int, 0, len(folders))
	for _, folder := range folders {
		if folder.DiscogID != nil {
			keepDiscogIDs = append(keepDiscogIDs, *folder.DiscogID)
			// Find the "All" folder and remember its DiscogID for later retrieval
			if folder.Name == "All" {
				allFolderDiscogID = folder.DiscogID
			}
		}
	}
	return keepDiscogIDs, allFolderDiscogID
}

// updateUserConfigWithAllFolder updates user configuration with the "All" folder as selected
func (o *OrchestrationService) updateUserConfigWithAllFolder(
	ctx context.Context,
	tx *gorm.DB,
	userID uuid.UUID,
	allFolderDiscogID int,
) error {
	log := o.log.Function("updateUserConfigWithAllFolder")

	// Retrieve the "All" folder from database with its proper ID
	allFolder, err := o.repos.Folder.GetFolderByDiscogID(
		ctx,
		tx,
		userID,
		allFolderDiscogID,
	)
	if err != nil {
		return log.Err("failed to retrieve All folder from database", err)
	}

	userConfig, err := o.repos.UserConfiguration.GetByUserID(ctx, tx, userID)
	if err != nil {
		return log.Err("failed to get user configuration", err)
	}

	userConfig.SelectedFolderID = &allFolder.ID
	if err = o.repos.UserConfiguration.Update(ctx, tx, userConfig); err != nil {
		return log.Err("failed to update user configuration with selected folder", err)
	}

	log.Info("Updated user configuration with All folder as selected",
		"userID", userID,
		"selectedFolderID", allFolder.ID)

	return nil
}

// processFoldersResponse handles the response for user folders requests
func (o *OrchestrationService) processFoldersResponse(
	ctx context.Context,
	metadata RequestMetadata,
	responseData map[string]any,
) error {
	log := o.log.Function("processFoldersResponse")

	// Use generic response processor
	discogsFoldersResponse, err := processDiscogsAPIResponse[DiscogsFoldersResponse](
		log, responseData, metadata, "folders")
	if err != nil {
		return nil // Don't return error as this is an expected API failure
	}

	// Convert Discogs folder items to our Folder model
	folders := make([]*Folder, 0, len(discogsFoldersResponse.Data.Folders))
	for _, discogsFolder := range discogsFoldersResponse.Data.Folders {
		log.Info("Processing folder", "discogID", discogsFolder)
		folder := &Folder{
			DiscogID:    &discogsFolder.ID,
			UserID:      metadata.UserID,
			Name:        discogsFolder.Name,
			Count:       discogsFolder.Count,
			ResourceURL: discogsFolder.ResourceURL,
		}
		folders = append(folders, folder)
	}

	log.Info("Successfully parsed folders data",
		"userID", metadata.UserID,
		"requestID", metadata.RequestID,
		"foldersCount", len(folders))

	// Extract sync data BEFORE transaction
	keepDiscogIDs, allFolderDiscogID := o.extractFolderSyncData(folders)

	// Execute transaction with minimal scope - only database operations
	err = o.transactionService.Execute(ctx, func(txCtx context.Context, tx *gorm.DB) error {
		log.Info("Upserting folders to database",
			"userID", metadata.UserID,
			"folderCount", len(folders))

		// Upsert all folders
		if err = o.repos.Folder.UpsertFolders(txCtx, tx, metadata.UserID, folders); err != nil {
			return log.Err("failed to upsert folders", err)
		}

		// Delete orphan folders not in the current sync
		if err = o.repos.Folder.DeleteOrphanFolders(txCtx, tx, metadata.UserID, keepDiscogIDs); err != nil {
			return log.Err("failed to delete orphan folders", err)
		}

		// Update user configuration with "All" folder as selected folder if found
		if allFolderDiscogID != nil {
			return o.updateUserConfigWithAllFolder(txCtx, tx, metadata.UserID, *allFolderDiscogID)
		}

		return nil
	})
	if err != nil {
		return log.Err("failed to save folders to database", err,
			"userID", metadata.UserID,
			"requestID", metadata.RequestID)
	}

	log.Info("Successfully saved folders to database",
		"userID", metadata.UserID,
		"requestID", metadata.RequestID,
		"foldersCount", len(folders))

	return nil
}
