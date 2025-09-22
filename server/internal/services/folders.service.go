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
	"github.com/valkey-io/valkey-go"
	"gorm.io/gorm"
)

type FoldersService struct {
	log                logger.Logger
	eventBus           *events.EventBus
	cache              valkey.Client
	repos              repositories.Repository
	transactionService *TransactionService
}

func NewFoldersService(
	eventBus *events.EventBus,
	repos repositories.Repository,
	transactionService *TransactionService,
) *FoldersService {
	log := logger.New("FoldersService")
	return &FoldersService{
		log:                log,
		eventBus:           eventBus,
		cache:              transactionService.db.Cache.General,
		repos:              repos,
		transactionService: transactionService,
	}
}

func (f *FoldersService) RequestUserFolders(
	ctx context.Context,
	user *User,
) (string, error) {
	log := f.log.Function("RequestUserFolders")

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

	if err := database.NewCacheBuilder(f.cache, requestID).
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

	if err := f.eventBus.Publish(events.WEBSOCKET, "user", message); err != nil {
		_ = database.NewCacheBuilder(f.cache, requestID).
			WithHashPattern(API_HASH).
			WithContext(ctx).
			Delete()
		return "", log.Err("failed to publish API request event", err)
	}

	return requestID, nil
}

func (f *FoldersService) ProcessFoldersResponse(
	ctx context.Context,
	metadata RequestMetadata,
	responseData map[string]any,
) error {
	log := f.log.Function("ProcessFoldersResponse")

	discogsFoldersResponse, err := processDiscogsAPIResponse[DiscogsFoldersResponse](
		log, responseData, metadata, "folders")
	if err != nil {
		return nil // Don't return error as this is an expected API failure
	}

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

	keepDiscogIDs, allFolderDiscogID := f.extractFolderSyncData(folders)

	err = f.transactionService.Execute(ctx, func(txCtx context.Context, tx *gorm.DB) error {
		log.Info("Upserting folders to database",
			"userID", metadata.UserID,
			"folderCount", len(folders))

		if err = f.repos.Folder.UpsertFolders(txCtx, tx, metadata.UserID, folders); err != nil {
			return log.Err("failed to upsert folders", err)
		}

		if err = f.repos.Folder.DeleteOrphanFolders(txCtx, tx, metadata.UserID, keepDiscogIDs); err != nil {
			return log.Err("failed to delete orphan folders", err)
		}

		if allFolderDiscogID != nil {
			return f.updateUserConfigWithAllFolderIfNotSet(txCtx, tx, metadata.UserID, *allFolderDiscogID)
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

func (f *FoldersService) extractFolderSyncData(
	folders []*Folder,
) (keepDiscogIDs []int, allFolderDiscogID *int) {
	keepDiscogIDs = make([]int, 0, len(folders))
	for _, folder := range folders {
		if folder.DiscogID != nil {
			keepDiscogIDs = append(keepDiscogIDs, *folder.DiscogID)
			if folder.Name == "All" {
				allFolderDiscogID = folder.DiscogID
			}
		}
	}
	return keepDiscogIDs, allFolderDiscogID
}

func (f *FoldersService) updateUserConfigWithAllFolderIfNotSet(
	ctx context.Context,
	tx *gorm.DB,
	userID uuid.UUID,
	allFolderDiscogID int,
) error {
	log := f.log.Function("updateUserConfigWithAllFolderIfNotSet")

	userConfig, err := f.repos.UserConfiguration.GetByUserID(ctx, tx, userID)
	if err != nil {
		return log.Err("failed to get user configuration", err)
	}

	// Only set the "All" folder if user doesn't already have a selected folder
	if userConfig.SelectedFolderID != nil {
		log.Info("User already has a selected folder, skipping update",
			"userID", userID,
			"existingSelectedFolderID", *userConfig.SelectedFolderID)
		return nil
	}

	allFolder, err := f.repos.Folder.GetFolderByDiscogID(
		ctx,
		tx,
		userID,
		allFolderDiscogID,
	)
	if err != nil {
		return log.Err("failed to retrieve All folder from database", err)
	}

	userConfig.SelectedFolderID = &allFolder.ID
	if err = f.repos.UserConfiguration.Update(ctx, tx, userConfig); err != nil {
		return log.Err("failed to update user configuration with selected folder", err)
	}

	log.Info("Updated user configuration with All folder as selected",
		"userID", userID,
		"selectedFolderID", allFolder.ID)

	return nil
}