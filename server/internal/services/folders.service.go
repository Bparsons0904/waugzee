package services

import (
	"context"
	"fmt"
	"strconv"
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
	COLLECTION_SYNC_HASH = "collection_sync"
	RELEASE_QUEUE_HASH   = "release_queue"
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

	keepDiscogIDs, _ := f.extractFolderSyncData(folders)

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

		// Set default selected folder to Uncategorized (folder 1) if not set
		return f.updateUserConfigWithUncategorizedFolderIfNotSet(txCtx, tx, metadata.UserID)
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

func (f *FoldersService) updateUserConfigWithUncategorizedFolderIfNotSet(
	ctx context.Context,
	tx *gorm.DB,
	userID uuid.UUID,
) error {
	log := f.log.Function("updateUserConfigWithUncategorizedFolderIfNotSet")

	userConfig, err := f.repos.UserConfiguration.GetByUserID(ctx, tx, userID)
	if err != nil {
		return log.Err("failed to get user configuration", err)
	}

	// Only set the default folder if user doesn't already have a selected folder
	if userConfig.SelectedFolderID != nil {
		log.Info("User already has a selected folder, skipping update",
			"userID", userID,
			"existingSelectedFolderID", *userConfig.SelectedFolderID)
		return nil
	}

	// Set to folder 1 (Uncategorized) as default instead of folder 0 (All)
	uncategorizedFolderID := 1
	userConfig.SelectedFolderID = &uncategorizedFolderID
	if err = f.repos.UserConfiguration.Update(ctx, tx, userConfig); err != nil {
		return log.Err("failed to update user configuration with selected folder", err)
	}

	log.Info("Updated user configuration with Uncategorized folder as selected",
		"userID", userID,
		"selectedFolderID", uncategorizedFolderID)

	return nil
}

func (f *FoldersService) RequestFolderReleases(
	ctx context.Context,
	user *User,
	folderID int,
	page int,
) (string, error) {
	log := f.log.Function("RequestFolderReleases")

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

	if folderID < 0 {
		return "", log.ErrMsg("folderID must be non-negative")
	}

	if page < 1 {
		page = 1
	}

	if page > 10000 { // Reasonable upper limit to prevent abuse
		return "", log.ErrMsg("page number too large (max: 10000)")
	}

	requestID := uuid.New().String()

	metadata := RequestMetadata{
		UserID:       user.ID,
		RequestID:    requestID,
		RequestType:  "folder_releases",
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
		"%s/users/%s/collection/folders/%d/releases?page=%d&per_page=100",
		DiscogsAPIBaseURL,
		*user.Configuration.DiscogsUsername,
		folderID,
		page,
	)

	message := events.Message{
		ID:      requestID,
		Service: events.API,
		Event:   "api_request",
		UserID:  user.ID.String(),
		Payload: map[string]any{
			"requestId":   requestID,
			"requestType": "folder_releases",
			"folderID":    folderID,
			"page":        page,
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

	log.Info("Requested folder releases",
		"userID", user.ID,
		"folderID", folderID,
		"page", page,
		"requestID", requestID)

	return requestID, nil
}

func (f *FoldersService) ProcessFolderReleasesResponse(
	ctx context.Context,
	metadata RequestMetadata,
	responseData map[string]any,
) error {
	log := f.log.Function("ProcessFolderReleasesResponse")

	discogsFolderReleasesResponse, err := processDiscogsAPIResponse[DiscogsFolderReleasesResponse](
		log, responseData, metadata, "folder_releases")
	if err != nil {
		return nil // Don't return error as this is an expected API failure
	}

	folderIDRaw, exists := responseData["folderID"]
	if !exists {
		return log.ErrMsg("missing folderID in response data")
	}

	var folderID int
	switch v := folderIDRaw.(type) {
	case int:
		folderID = v
	case float64:
		folderID = int(v)
	case string:
		folderID, err = strconv.Atoi(v)
		if err != nil {
			return log.Err("invalid folderID format", err, "folderID", v)
		}
	default:
		return log.ErrMsg("invalid folderID type")
	}

	missingReleaseIDs := make([]int64, 0)

	// Get current sync state from cache
	var syncState CollectionSyncState
	found, err := database.NewCacheBuilder(f.cache, metadata.UserID.String()).
		WithHashPattern(COLLECTION_SYNC_HASH).
		WithContext(ctx).
		Get(&syncState)
	if err != nil {
		return log.Err("failed to get sync state from cache", err)
	}
	if !found {
		// No active sync state - process as individual folder (legacy mode)
		return f.processIndividualFolder(ctx, metadata, responseData, folderID, discogsFolderReleasesResponse)
	}

	// Accumulate releases to sync state instead of immediate DB writes
	for _, discogsRelease := range discogsFolderReleasesResponse.Data.Releases {
		log.Info("Accumulating folder release to sync state",
			"releaseID", discogsRelease.ID,
			"instanceID", discogsRelease.InstanceID,
			"folderID", folderID)

		releaseID := discogsRelease.ID
		if releaseID == 0 && discogsRelease.BasicInformation.ID != 0 {
			releaseID = discogsRelease.BasicInformation.ID
		}

		if releaseID == 0 {
			log.Warn("Skipping release with no valid ID", "instanceID", discogsRelease.InstanceID)
			continue
		}

		userRelease := &UserRelease{
			UserID:     metadata.UserID,
			ReleaseID:  releaseID, // Use the Discogs release ID directly
			InstanceID: discogsRelease.InstanceID,
			FolderID:   folderID,
			Rating:     discogsRelease.Rating,
			Notes:      discogsRelease.Notes,
			Active:     true,
		}

		// Add to merged releases (overwrite if exists - latest folder wins)
		syncState.MergedReleases[discogsRelease.InstanceID] = userRelease
		missingReleaseIDs = append(missingReleaseIDs, releaseID)
	}

	// Mark this folder as processed if this was the last page
	if discogsFolderReleasesResponse.Data.Pagination.Page >= discogsFolderReleasesResponse.Data.Pagination.Pages {
		syncState.CompletedFolders[folderID] = true
		syncState.ProcessedFolders = len(syncState.CompletedFolders)
	}

	// Check if all folders are complete
	if syncState.ProcessedFolders >= syncState.TotalFolders {
		syncState.SyncComplete = true
		log.Info("All folders processed, executing differential sync",
			"userID", metadata.UserID,
			"totalReleases", len(syncState.MergedReleases))

		// Execute differential sync
		err = f.transactionService.Execute(ctx, func(txCtx context.Context, tx *gorm.DB) error {
			// First, ensure all Release records exist
			err := f.ensureReleaseRecordsExist(txCtx, tx, syncState.MergedReleases)
			if err != nil {
				return log.Err("failed to ensure release records exist", err)
			}

			operations, err := f.analyzeDifferentialSync(txCtx, tx, metadata.UserID, syncState.MergedReleases)
			if err != nil {
				return log.Err("failed to analyze differential sync", err)
			}

			return f.executeSyncOperations(txCtx, tx, metadata.UserID, operations)
		})
		if err != nil {
			return log.Err("failed to execute differential sync", err)
		}

		// Clean up sync state
		_ = database.NewCacheBuilder(f.cache, metadata.UserID.String()).
			WithHashPattern(COLLECTION_SYNC_HASH).
			WithContext(ctx).
			Delete()

		log.Info("Collection sync completed successfully",
			"userID", metadata.UserID,
			"totalReleases", len(syncState.MergedReleases))
	} else {
		// Update sync state in cache
		err = database.NewCacheBuilder(f.cache, metadata.UserID.String()).
			WithHashPattern(COLLECTION_SYNC_HASH).
			WithStruct(syncState).
			WithTTL(30*time.Minute).
			WithContext(ctx).
			Set()
		if err != nil {
			log.Warn("Failed to update sync state", "error", err)
		}

		log.Info("Folder processing accumulated",
			"userID", metadata.UserID,
			"folderID", folderID,
			"processedFolders", syncState.ProcessedFolders,
			"totalFolders", syncState.TotalFolders,
			"releasesInPage", len(discogsFolderReleasesResponse.Data.Releases))
	}

	// Queue missing releases for processing
	if len(missingReleaseIDs) > 0 {
		err = f.queueMissingReleases(ctx, metadata.UserID, missingReleaseIDs)
		if err != nil {
			log.Warn("Failed to queue missing releases",
				"error", err,
				"releaseCount", len(missingReleaseIDs))
		}
	}

	return nil
}

// processIndividualFolder handles processing a single folder when not part of collection sync
func (f *FoldersService) processIndividualFolder(
	ctx context.Context,
	metadata RequestMetadata,
	responseData map[string]any,
	folderID int,
	discogsFolderReleasesResponse *DiscogsFolderReleasesResponse,
) error {
	log := f.log.Function("processIndividualFolder")

	log.Info("Processing individual folder (not part of collection sync)",
		"userID", metadata.UserID,
		"folderID", folderID,
		"releasesCount", len(discogsFolderReleasesResponse.Data.Releases))

	// TODO: Implement individual folder processing
	// For now, just log that we received the data
	missingReleaseIDs := make([]int64, 0)
	for _, discogsRelease := range discogsFolderReleasesResponse.Data.Releases {
		releaseID := discogsRelease.ID
		if releaseID == 0 && discogsRelease.BasicInformation.ID != 0 {
			releaseID = discogsRelease.BasicInformation.ID
		}
		if releaseID > 0 {
			missingReleaseIDs = append(missingReleaseIDs, releaseID)
		}
	}

	// Queue missing releases for processing
	if len(missingReleaseIDs) > 0 {
		err := f.queueMissingReleases(ctx, metadata.UserID, missingReleaseIDs)
		if err != nil {
			log.Warn("Failed to queue missing releases",
				"error", err,
				"releaseCount", len(missingReleaseIDs))
		}
	}

	log.Info("Individual folder processed",
		"userID", metadata.UserID,
		"folderID", folderID,
		"queuedReleases", len(missingReleaseIDs))

	return nil
}

func (f *FoldersService) queueMissingReleases(
	ctx context.Context,
	userID uuid.UUID,
	releaseIDs []int64,
) error {
	log := f.log.Function("queueMissingReleases")

	if len(releaseIDs) == 0 {
		return nil
	}

	// Queue missing releases using CacheBuilder pattern
	queueData := map[string]interface{}{
		"releaseIDs": releaseIDs,
		"queued_at":  time.Now().Unix(),
	}

	err := database.NewCacheBuilder(f.cache, userID.String()).
		WithHashPattern(RELEASE_QUEUE_HASH).
		WithStruct(queueData).
		WithTTL(24*time.Hour). // 24 hour TTL
		WithContext(ctx).
		Set()
	if err != nil {
		return log.Err("failed to queue releases for processing", err)
	}

	log.Info("Queued missing releases for processing",
		"userID", userID,
		"releaseCount", len(releaseIDs))

	return nil
}

// SyncCollectionOperations represents the result of differential sync analysis
type SyncCollectionOperations struct {
	Create []UserRelease
	Update []UserRelease
	Delete []int // InstanceIDs to delete
}

// CollectionSyncState holds the state during folder collection sync
type CollectionSyncState struct {
	UserID           uuid.UUID
	TotalFolders     int
	ProcessedFolders int
	MergedReleases   map[int]*UserRelease // key: InstanceID
	CompletedFolders map[int]bool         // key: FolderID
	SyncComplete     bool
}

func (f *FoldersService) SyncAllUserFolders(
	ctx context.Context,
	user *User,
) error {
	log := f.log.Function("SyncAllUserFolders")

	// First, get user's folders to determine which ones to sync (skip folder 0)
	folders, err := f.repos.Folder.GetUserFolders(ctx, nil, user.ID)
	if err != nil {
		return log.Err("failed to get user folders", err)
	}

	// Filter to folders 1+ (skip folder 0 "All")
	syncFolders := make([]*Folder, 0)
	for _, folder := range folders {
		if folder.DiscogID != nil && *folder.DiscogID > 0 {
			syncFolders = append(syncFolders, folder)
		}
	}

	if len(syncFolders) == 0 {
		return log.ErrMsg("no folders to sync (only found folder 0 or no folders)")
	}

	log.Info("Starting collection sync for folders 1+",
		"userID", user.ID,
		"totalFolders", len(syncFolders))

	// Initialize sync state
	syncState := &CollectionSyncState{
		UserID:           user.ID,
		TotalFolders:     len(syncFolders),
		ProcessedFolders: 0,
		MergedReleases:   make(map[int]*UserRelease),
		CompletedFolders: make(map[int]bool),
		SyncComplete:     false,
	}

	// Store sync state in cache for tracking across API responses
	err = database.NewCacheBuilder(f.cache, user.ID.String()).
		WithHashPattern(COLLECTION_SYNC_HASH).
		WithStruct(syncState).
		WithTTL(30*time.Minute). // 30 min timeout for sync
		WithContext(ctx).
		Set()
	if err != nil {
		return log.Err("failed to store sync state", err)
	}

	// Start sync for each folder (page 1)
	for _, folder := range syncFolders {
		if folder.DiscogID == nil {
			continue
		}

		_, err = f.RequestFolderReleases(ctx, user, *folder.DiscogID, 1)
		if err != nil {
			log.Warn("Failed to start sync for folder",
				"folderID", *folder.DiscogID,
				"error", err)
		}
	}

	log.Info("Collection sync initiated",
		"userID", user.ID,
		"foldersRequested", len(syncFolders))

	return nil
}

func (f *FoldersService) analyzeDifferentialSync(
	ctx context.Context,
	tx *gorm.DB,
	userID uuid.UUID,
	mergedReleases map[int]*UserRelease,
) (*SyncCollectionOperations, error) {
	log := f.log.Function("analyzeDifferentialSync")

	// Get current state from database
	currentReleases, err := f.repos.UserRelease.GetExistingByUser(ctx, tx, userID)
	if err != nil {
		return nil, log.Err("failed to get existing user releases", err)
	}

	operations := &SyncCollectionOperations{
		Create: make([]UserRelease, 0),
		Update: make([]UserRelease, 0),
		Delete: make([]int, 0),
	}

	// Find creates and updates
	for instanceID, mergedRelease := range mergedReleases {
		if currentRelease, exists := currentReleases[instanceID]; exists {
			// Check if folder changed (update needed)
			if currentRelease.FolderID != mergedRelease.FolderID ||
				currentRelease.Rating != mergedRelease.Rating ||
				currentRelease.Notes != mergedRelease.Notes {
				// Update existing record with new folder/rating/notes
				updatedRelease := *currentRelease
				updatedRelease.FolderID = mergedRelease.FolderID
				updatedRelease.Rating = mergedRelease.Rating
				updatedRelease.Notes = mergedRelease.Notes
				operations.Update = append(operations.Update, updatedRelease)
			}
			// Remove from current map so we can find deletes
			delete(currentReleases, instanceID)
		} else {
			// New release (create)
			operations.Create = append(operations.Create, *mergedRelease)
		}
	}

	// Find deletes (remaining items in current that weren't in merged)
	for instanceID := range currentReleases {
		operations.Delete = append(operations.Delete, instanceID)
	}

	log.Info("Differential sync analysis complete",
		"userID", userID,
		"creates", len(operations.Create),
		"updates", len(operations.Update),
		"deletes", len(operations.Delete))

	return operations, nil
}

func (f *FoldersService) ensureReleaseRecordsExist(
	ctx context.Context,
	tx *gorm.DB,
	mergedReleases map[int]*UserRelease,
) error {
	log := f.log.Function("ensureReleaseRecordsExist")

	if len(mergedReleases) == 0 {
		return nil
	}

	// Collect all unique release IDs
	releaseIDSet := make(map[int64]bool)
	for _, userRelease := range mergedReleases {
		releaseIDSet[userRelease.ReleaseID] = true
	}

	releaseIDs := make([]int64, 0, len(releaseIDSet))
	for releaseID := range releaseIDSet {
		releaseIDs = append(releaseIDs, releaseID)
	}

	// Get existing releases
	existingReleases, err := f.repos.Release.GetBatchByDiscogsIDs(ctx, tx, releaseIDs)
	if err != nil {
		return log.Err("failed to get existing releases", err)
	}

	// Create placeholder releases for missing ones
	missingReleases := make([]*Release, 0)
	for _, releaseID := range releaseIDs {
		if _, exists := existingReleases[releaseID]; !exists {
			// Create minimal placeholder release
			release := &Release{
				ID:     releaseID,
				Title:  "Unknown Release", // Will be updated when full release data is processed
				Format: FormatVinyl,       // Default to vinyl
			}
			missingReleases = append(missingReleases, release)
		}
	}

	// Insert missing releases
	if len(missingReleases) > 0 {
		err = f.repos.Release.InsertBatch(ctx, tx, missingReleases)
		if err != nil {
			return log.Err("failed to insert missing releases", err, "count", len(missingReleases))
		}
		log.Info("Created placeholder releases", "count", len(missingReleases))
	}

	return nil
}

func (f *FoldersService) executeSyncOperations(
	ctx context.Context,
	tx *gorm.DB,
	userID uuid.UUID,
	operations *SyncCollectionOperations,
) error {
	log := f.log.Function("executeSyncOperations")

	// Execute creates
	if len(operations.Create) > 0 {
		// Convert slice to slice of pointers
		createPointers := make([]*UserRelease, len(operations.Create))
		for i := range operations.Create {
			createPointers[i] = &operations.Create[i]
		}
		if err := f.repos.UserRelease.CreateBatch(ctx, tx, createPointers); err != nil {
			return log.Err("failed to create user releases", err)
		}
	}

	// Execute updates
	if len(operations.Update) > 0 {
		// Convert slice to slice of pointers
		updatePointers := make([]*UserRelease, len(operations.Update))
		for i := range operations.Update {
			updatePointers[i] = &operations.Update[i]
		}
		if err := f.repos.UserRelease.UpdateBatch(ctx, tx, updatePointers); err != nil {
			return log.Err("failed to update user releases", err)
		}
	}

	// Execute deletes
	if len(operations.Delete) > 0 {
		if err := f.repos.UserRelease.DeleteBatch(ctx, tx, userID, operations.Delete); err != nil {
			return log.Err("failed to delete user releases", err)
		}
	}

	log.Info("Sync operations executed successfully",
		"userID", userID,
		"created", len(operations.Create),
		"updated", len(operations.Update),
		"deleted", len(operations.Delete))

	return nil
}