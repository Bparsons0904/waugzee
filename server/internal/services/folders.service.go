package services

import (
	"bytes"
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
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	COLLECTION_SYNC_HASH = "collection_sync"
	RELEASE_QUEUE_HASH   = "release_queue"
)

type FoldersService struct {
	log                         logger.Logger
	eventBus                    *events.EventBus
	repos                       repositories.Repository
	db                          database.DB
	transactionService          *TransactionService
	folderDataExtractionService *FolderDataExtractionService
	discogsRateLimiter          *DiscogsRateLimiterService
}

func NewFoldersService(
	eventBus *events.EventBus,
	repos repositories.Repository,
	db database.DB,
	transactionService *TransactionService,
	folderDataExtractionService *FolderDataExtractionService,
	discogsRateLimiter *DiscogsRateLimiterService,
) *FoldersService {
	log := logger.New("FoldersService")
	return &FoldersService{
		log:                         log,
		eventBus:                    eventBus,
		repos:                       repos,
		db:                          db,
		transactionService:          transactionService,
		folderDataExtractionService: folderDataExtractionService,
		discogsRateLimiter:          discogsRateLimiter,
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

	// Check rate limit before making API request
	if err := f.discogsRateLimiter.CheckUserRateLimit(ctx, user.ID); err != nil {
		return "", log.Err("rate limit check failed", err)
	}

	requestID := uuid.New().String()

	metadata := RequestMetadata{
		UserID:       user.ID,
		RequestID:    requestID,
		RequestType:  "folders",
		Timestamp:    time.Now(),
		DiscogsToken: *user.Configuration.DiscogsToken,
	}

	if err := database.NewCacheBuilder(f.db.Cache.ClientAPI, requestID).
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
		_ = database.NewCacheBuilder(f.db.Cache.ClientAPI, requestID).
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
		folder := &Folder{
			DiscogID:    &discogsFolder.ID,
			UserID:      metadata.UserID,
			Name:        discogsFolder.Name,
			Count:       discogsFolder.Count,
			ResourceURL: discogsFolder.ResourceURL,
		}
		folders = append(folders, folder)
	}

	keepDiscogIDs, _ := f.extractFolderSyncData(folders)

	// Execute folder upsert
	err = f.transactionService.Execute(ctx, func(txCtx context.Context, tx *gorm.DB) error {
		return f.repos.Folder.UpsertFolders(txCtx, tx, metadata.UserID, folders)
	})
	if err != nil {
		return log.Err("failed to upsert folders", err,
			"userID", metadata.UserID,
			"requestID", metadata.RequestID)
	}

	// Execute orphan cleanup in separate transaction
	err = f.transactionService.Execute(ctx, func(txCtx context.Context, tx *gorm.DB) error {
		return f.repos.Folder.DeleteOrphanFolders(txCtx, tx, metadata.UserID, keepDiscogIDs)
	})
	if err != nil {
		return log.Err("failed to delete orphan folders", err,
			"userID", metadata.UserID,
			"requestID", metadata.RequestID)
	}

	// Update user config in separate transaction
	err = f.transactionService.Execute(ctx, func(txCtx context.Context, tx *gorm.DB) error {
		return f.updateUserConfigWithUncategorizedFolderIfNotSet(txCtx, tx, metadata.UserID)
	})
	if err != nil {
		return log.Err("failed to update user config with default folder", err,
			"userID", metadata.UserID,
			"requestID", metadata.RequestID)
	}

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
		return nil
	}

	// Set to folder 1 (Uncategorized) as default instead of folder 0 (All)
	uncategorizedFolderID := 1
	userConfig.SelectedFolderID = &uncategorizedFolderID
	if err = f.repos.UserConfiguration.Update(ctx, tx, userConfig, f.repos.User); err != nil {
		return log.Err("failed to update user configuration with selected folder", err)
	}

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

	// Check rate limit before making API request
	if err := f.discogsRateLimiter.CheckUserRateLimit(ctx, user.ID); err != nil {
		return "", log.Err("rate limit check failed", err)
	}

	requestID := uuid.New().String()

	metadata := RequestMetadata{
		UserID:       user.ID,
		RequestID:    requestID,
		RequestType:  "folder_releases",
		Timestamp:    time.Now(),
		DiscogsToken: *user.Configuration.DiscogsToken,
	}

	if err := database.NewCacheBuilder(f.db.Cache.ClientAPI, requestID).
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
		_ = database.NewCacheBuilder(f.db.Cache.ClientAPI, requestID).
			WithHashPattern(API_HASH).
			WithContext(ctx).
			Delete()
		return "", log.Err("failed to publish API request event", err)
	}

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

	// Try to get folderID from request data first
	folderIDRaw, exists := responseData["folderID"]
	if !exists {
		// Fallback: extract folder_id from first release in the response
		if discogsFolderReleasesResponse != nil &&
			len(discogsFolderReleasesResponse.Data.Releases) > 0 {
			folderIDRaw = discogsFolderReleasesResponse.Data.Releases[0].FolderID
		} else {
			return log.ErrMsg("missing folderID in response data and no releases to extract from")
		}
	}

	folderID, ok := folderIDRaw.(int)
	if !ok {
		return log.ErrMsg("folderID is not an integer")
	}

	missingReleaseIDs := make([]int64, 0)

	// Get current sync state from cache
	var syncState CollectionSyncState
	found, err := database.NewCacheBuilder(f.db.Cache.ClientAPI, metadata.UserID.String()).
		WithHashPattern(COLLECTION_SYNC_HASH).
		WithContext(ctx).
		Get(&syncState)
	if err != nil {
		return log.Err("failed to get sync state from cache", err)
	}
	if !found {
		// No active sync state - process as individual folder (legacy mode)
		return f.processIndividualFolder(
			ctx,
			metadata,
			folderID,
			discogsFolderReleasesResponse,
		)
	}

	// Accumulate releases to sync state instead of immediate DB writes
	for _, discogsRelease := range discogsFolderReleasesResponse.Data.Releases {

		releaseID := discogsRelease.ID
		if releaseID == 0 && discogsRelease.BasicInformation.ID != 0 {
			releaseID = discogsRelease.BasicInformation.ID
		}

		if releaseID == 0 {
			log.Warn("Skipping release with no valid ID", "instanceID", discogsRelease.InstanceID)
			continue
		}

		// Convert notes to JSON
		notesJSON, _ := json.Marshal(discogsRelease.Notes)

		// Parse DateAdded from Discogs API response
		var dateAdded time.Time
		if discogsRelease.DateAdded != "" {
			// Try to parse the date_added field from Discogs API
			var parsedDate time.Time
			if parsedDate, err = time.Parse(time.RFC3339, discogsRelease.DateAdded); err == nil {
				dateAdded = parsedDate
			} else {
				// Fallback to current time if parsing fails
				log.Warn("Failed to parse DateAdded from Discogs API, using current time",
					"dateAdded", discogsRelease.DateAdded,
					"instanceID", discogsRelease.InstanceID,
					"error", err)
				dateAdded = time.Now()
			}
		} else {
			// If no date_added in response, use current time
			dateAdded = time.Now()
		}

		userRelease := &UserRelease{
			UserID:     metadata.UserID,
			ReleaseID:  releaseID, // Use the Discogs release ID directly
			InstanceID: discogsRelease.InstanceID,
			FolderID:   folderID,
			Rating:     discogsRelease.Rating,
			Notes:      datatypes.JSON(notesJSON),
			DateAdded:  dateAdded,
			Active:     true,
		}

		// Add to merged releases (overwrite if exists - latest folder wins)
		syncState.MergedReleases[discogsRelease.InstanceID] = userRelease
		// Store original data for data extraction
		syncState.OriginalReleases[discogsRelease.InstanceID] = discogsRelease
		missingReleaseIDs = append(missingReleaseIDs, releaseID)
	}

	// Check if this folder has more pages to process
	currentPage := discogsFolderReleasesResponse.Data.Pagination.Page
	totalPages := discogsFolderReleasesResponse.Data.Pagination.Pages

	if currentPage < totalPages {
		// Check if there's a next URL in pagination
		if nextURL, exists := discogsFolderReleasesResponse.Data.Pagination.URLs["next"]; exists &&
			nextURL != "" {

			// Check rate limit before making pagination API request
			if err = f.discogsRateLimiter.CheckUserRateLimit(ctx, metadata.UserID); err != nil {
				log.Warn("Rate limit check failed for pagination request", "error", err)
				return nil
			}

			// Create API request using the next URL from pagination
			requestID := uuid.New().String()

			requestMetadata := RequestMetadata{
				UserID:       metadata.UserID,
				RequestID:    requestID,
				RequestType:  "folder_releases",
				Timestamp:    time.Now(),
				DiscogsToken: metadata.DiscogsToken,
			}

			if err = database.NewCacheBuilder(f.db.Cache.ClientAPI, requestID).
				WithHashPattern(API_HASH).
				WithStruct(requestMetadata).
				WithTTL(APIRequestTTL).
				WithContext(ctx).
				Set(); err != nil {
				log.Warn("Failed to store pagination request metadata", "error", err)
				return nil
			}

			message := events.Message{
				ID:      requestID,
				Service: events.API,
				Event:   "api_request",
				UserID:  metadata.UserID.String(),
				Payload: map[string]any{
					"requestId":   requestID,
					"requestType": "folder_releases",
					"folderID":    folderID,
					"page":        currentPage + 1,
					"url":         nextURL,
					"method":      "GET",
					"headers": map[string]string{
						"Authorization": fmt.Sprintf("Discogs token=%s", metadata.DiscogsToken),
					},
					"callbackService": "orchestration",
					"callbackEvent":   "api_response",
				},
				Timestamp: time.Now(),
			}

			if err = f.eventBus.Publish(events.WEBSOCKET, "user", message); err != nil {
				log.Warn("Failed to publish pagination request", "error", err)
			}
		} else {
			log.Warn("Next page expected but no next URL found in pagination",
				"userID", metadata.UserID,
				"folderID", folderID,
				"currentPage", currentPage)
		}
	} else {
		// Mark this folder as completed - all pages processed
		syncState.CompletedFolders[folderID] = true
		syncState.ProcessedFolders = len(syncState.CompletedFolders)

	}

	// Check if all folders are complete
	if syncState.ProcessedFolders >= syncState.TotalFolders {
		syncState.SyncComplete = true

		// Perform release validation if not done yet
		if !syncState.ReleaseValidationDone {
			err = f.performReleaseValidation(ctx, &syncState, metadata.UserID)
			if err != nil {
				return log.Err("failed to perform release validation", err)
			}

			// Update sync state in cache
			err = database.NewCacheBuilder(f.db.Cache.ClientAPI, metadata.UserID.String()).
				WithHashPattern(COLLECTION_SYNC_HASH).
				WithStruct(syncState).
				WithTTL(30 * time.Minute).
				WithContext(ctx).
				Set()
			if err != nil {
				log.Warn("Failed to update sync state after validation", "error", err)
			}

			// If we have missing releases, return early to wait for API responses
			if len(syncState.MissingReleaseIDs) > 0 && !syncState.AllReleasesReady {
				return nil
			}
		}

		// Check if all releases are ready before proceeding
		if !syncState.AllReleasesReady && len(syncState.PendingReleaseRequests) > 0 {
			// Still waiting for release API responses
			return nil
		}

		// Analyze what changes need to be made (business logic - outside transaction)
		var operations *SyncCollectionOperations
		operations, err = f.analyzeDifferentialSync(
			ctx,
			metadata.UserID,
			syncState.MergedReleases,
		)
		if err != nil {
			return log.Err("failed to analyze differential sync", err)
		}

		// Execute sync operations in focused transaction
		err = f.transactionService.Execute(ctx, func(txCtx context.Context, tx *gorm.DB) error {
			return f.executeSyncOperations(txCtx, tx, metadata.UserID, operations)
		})
		if err != nil {
			return log.Err("failed to execute differential sync", err)
		}

		// Extract basic information in separate transaction to avoid long locks
		folderReleases := make([]DiscogsFolderReleaseItem, 0, len(syncState.OriginalReleases))
		for _, originalRelease := range syncState.OriginalReleases {
			folderReleases = append(folderReleases, originalRelease)
		}

		if len(folderReleases) > 0 {
			err = f.transactionService.Execute(ctx, func(txCtx context.Context, tx *gorm.DB) error {
				return f.folderDataExtractionService.ExtractBasicInformation(txCtx, tx, folderReleases)
			})
			if err != nil {
				log.Warn("Failed to extract basic information", "error", err)
				// Don't fail the sync for data extraction errors
			}
		}

		// Clean up sync state
		_ = database.NewCacheBuilder(f.db.Cache.ClientAPI, metadata.UserID.String()).
			WithHashPattern(COLLECTION_SYNC_HASH).
			WithContext(ctx).
			Delete()

		// TODO: TESTING - Early escape before full data sync to test basic data extraction
		// Remove this early return after testing to enable full data sync
		// log.Info("Collection sync completed successfully (TESTING MODE - full data sync disabled)",
		// 	"userID", metadata.UserID,
		// 	"totalReleases", len(syncState.MergedReleases))
		// return nil
		//
		// // After sync completion, identify records needing full data
		// releaseIDs := make([]int64, 0, len(syncState.MergedReleases))
		// for _, userRelease := range syncState.MergedReleases {
		// 	releaseIDs = append(releaseIDs, userRelease.ReleaseID)
		// }
		//
		// // Get records needing full data sync
		// needingFullData, err := f.folderDataExtractionService.GetRecordsNeedingFullData(
		// 	ctx,
		// 	f.db.SQLWithContext(ctx),
		// 	releaseIDs,
		// )
		// if err != nil {
		// 	log.Warn("Failed to identify records needing full data", "error", err)
		// } else if len(needingFullData) > 0 {
		// 	// Send WebSocket message to client with list of IDs to fetch
		// 	message := events.Message{
		// 		ID:      uuid.New().String(),
		// 		Service: events.SYSTEM,
		// 		Event:   "sync_trigger_full_data",
		// 		UserID:  metadata.UserID.String(),
		// 		Payload: map[string]any{
		// 			"releaseIDs": needingFullData,
		// 			"message":    "Additional release data needs to be fetched",
		// 		},
		// 		Timestamp: time.Now(),
		// 	}
		//
		// 	if err = f.eventBus.Publish(events.WEBSOCKET, "user", message); err != nil {
		// 		log.Warn("Failed to publish sync trigger message", "error", err)
		// 	} else {
		// 		log.Info("Triggered client-side full data sync",
		// 			"userID", metadata.UserID,
		// 			"releaseCount", len(needingFullData))
		// 	}
		// }
		//
		// log.Info("Collection sync completed successfully",
		// 	"userID", metadata.UserID,
		// 	"totalReleases", len(syncState.MergedReleases),
		// 	"needingFullData", len(needingFullData))
	} else {
		// Update sync state in cache
		err = database.NewCacheBuilder(f.db.Cache.ClientAPI, metadata.UserID.String()).
			WithHashPattern(COLLECTION_SYNC_HASH).
			WithStruct(syncState).
			WithTTL(30 * time.Minute).
			WithContext(ctx).
			Set()
		if err != nil {
			log.Warn("Failed to update sync state", "error", err)
		}

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
	folderID int,
	discogsFolderReleasesResponse *DiscogsFolderReleasesResponse,
) error {
	log := f.log.Function("processIndividualFolder")

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
	queueData := map[string]any{
		"releaseIDs": releaseIDs,
		"queued_at":  time.Now().Unix(),
	}

	err := database.NewCacheBuilder(f.db.Cache.ClientAPI, userID.String()).
		WithHashPattern(RELEASE_QUEUE_HASH).
		WithStruct(queueData).
		WithTTL(24 * time.Hour). // 24 hour TTL
		WithContext(ctx).
		Set()
	if err != nil {
		return log.Err("failed to queue releases for processing", err)
	}

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
	MergedReleases   map[int]*UserRelease             // key: InstanceID
	OriginalReleases map[int]DiscogsFolderReleaseItem // key: InstanceID - for data extraction
	CompletedFolders map[int]bool                     // key: FolderID
	SyncComplete     bool
	// Release validation tracking
	PendingReleaseRequests map[string]bool // key: requestID - tracks pending API requests
	MissingReleaseIDs      []int64         // release IDs that need to be fetched
	ExistingReleaseIDs     []int64         // release IDs that already exist
	ReleaseValidationDone  bool            // whether release validation is complete
	AllReleasesReady       bool            // whether all missing releases have been fetched
}

func (f *FoldersService) SyncAllUserFolders(
	ctx context.Context,
	user *User,
) error {
	log := f.log.Function("SyncAllUserFolders")

	// First, get user's folders to determine which ones to sync (skip folder 0)
	folders, err := f.repos.Folder.GetUserFolders(ctx, f.db.SQLWithContext(ctx), user.ID)
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

	// Initialize sync state
	syncState := &CollectionSyncState{
		UserID:                 user.ID,
		TotalFolders:           len(syncFolders),
		ProcessedFolders:       0,
		MergedReleases:         make(map[int]*UserRelease),
		OriginalReleases:       make(map[int]DiscogsFolderReleaseItem),
		CompletedFolders:       make(map[int]bool),
		SyncComplete:           false,
		PendingReleaseRequests: make(map[string]bool),
		MissingReleaseIDs:      make([]int64, 0),
		ExistingReleaseIDs:     make([]int64, 0),
		ReleaseValidationDone:  false,
		AllReleasesReady:       false,
	}

	// Store sync state in cache for tracking across API responses
	err = database.NewCacheBuilder(f.db.Cache.ClientAPI, user.ID).
		WithHashPattern(COLLECTION_SYNC_HASH).
		WithStruct(syncState).
		WithTTL(30 * time.Minute). // 30 min timeout for sync
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

	return nil
}

func (f *FoldersService) analyzeDifferentialSync(
	ctx context.Context,
	userID uuid.UUID,
	mergedReleases map[int]*UserRelease,
) (*SyncCollectionOperations, error) {
	log := f.log.Function("analyzeDifferentialSync")

	// Get current state from database
	currentReleases, err := f.repos.UserRelease.GetExistingByUser(
		ctx,
		f.db.SQLWithContext(ctx),
		userID,
	)
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
			// Check if folder, rating, notes, or dateAdded changed (update needed)
			if currentRelease.FolderID != mergedRelease.FolderID ||
				currentRelease.Rating != mergedRelease.Rating ||
				!bytes.Equal(currentRelease.Notes, mergedRelease.Notes) ||
				!currentRelease.DateAdded.Equal(mergedRelease.DateAdded) {
				// Update existing record with new folder/rating/notes/dateAdded
				updatedRelease := *currentRelease
				updatedRelease.FolderID = mergedRelease.FolderID
				updatedRelease.Rating = mergedRelease.Rating
				updatedRelease.Notes = mergedRelease.Notes
				updatedRelease.DateAdded = mergedRelease.DateAdded
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

	return operations, nil
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

	return nil
}

// performReleaseValidation validates that releases exist and handles missing ones
func (f *FoldersService) performReleaseValidation(
	ctx context.Context,
	syncState *CollectionSyncState,
	userID uuid.UUID,
) error {
	log := f.log.Function("performReleaseValidation")

	// Extract all release IDs from merged releases
	releaseIDs := make([]int64, 0, len(syncState.MergedReleases))
	for _, userRelease := range syncState.MergedReleases {
		releaseIDs = append(releaseIDs, userRelease.ReleaseID)
	}

	if len(releaseIDs) == 0 {
		log.Info("No releases to validate")
		syncState.ReleaseValidationDone = true
		syncState.AllReleasesReady = true
		return nil
	}

	// Check which releases exist in our database
	existingReleases, missingReleases, err := f.repos.Release.CheckReleaseExistence(
		ctx,
		f.db.SQLWithContext(ctx),
		releaseIDs,
	)
	if err != nil {
		return log.Err("failed to check release existence", err)
	}

	syncState.ExistingReleaseIDs = existingReleases
	syncState.MissingReleaseIDs = missingReleases
	syncState.ReleaseValidationDone = true

	log.Info("Release validation completed",
		"totalReleases", len(releaseIDs),
		"existing", len(existingReleases),
		"missing", len(missingReleases))

	// Update images for existing releases
	if len(existingReleases) > 0 {
		err = f.updateExistingReleaseImages(ctx, syncState, existingReleases)
		if err != nil {
			log.Warn("Failed to update release images", "error", err)
			// Don't fail the sync for image update errors
		}
	}

	// If we have missing releases, initiate API requests to fetch them
	if len(missingReleases) > 0 {
		// Get user for API requests
		user, err := f.repos.User.GetByID(ctx, f.db.SQLWithContext(ctx), userID)
		if err != nil {
			return log.Err("failed to get user for release sync", err)
		}

		// Request missing releases through release sync service
		releaseSyncService := NewReleaseSyncService(
			f.eventBus,
			f.repos,
			f.db,
			f.discogsRateLimiter,
		)

		syncStateID := userID.String() // Use user ID as sync state ID
		err = releaseSyncService.RequestMissingReleases(
			ctx,
			user,
			missingReleases,
			syncStateID,
		)
		if err != nil {
			log.Warn("Failed to request missing releases", "error", err)
			// Continue with existing releases only
			syncState.AllReleasesReady = true
		} else {
			// Mark that we're waiting for release API responses
			syncState.AllReleasesReady = false
			// We'll need to track the request IDs somehow - for now assume the release sync service
			// will manage this through the API response callbacks
			if syncState.PendingReleaseRequests == nil {
				syncState.PendingReleaseRequests = make(map[string]bool)
			}
		}
	} else {
		// No missing releases, we're ready to proceed
		syncState.AllReleasesReady = true
	}

	return nil
}

// updateExistingReleaseImages updates thumb and cover image URLs for existing releases
func (f *FoldersService) updateExistingReleaseImages(
	ctx context.Context,
	syncState *CollectionSyncState,
	existingReleaseIDs []int64,
) error {
	log := f.log.Function("updateExistingReleaseImages")

	// Build image updates from original release data
	imageUpdates := make([]repositories.ReleaseImageUpdate, 0)
	existingMap := make(map[int64]bool)
	for _, id := range existingReleaseIDs {
		existingMap[id] = true
	}

	for _, originalRelease := range syncState.OriginalReleases {
		releaseID := originalRelease.ID
		if releaseID == 0 && originalRelease.BasicInformation.ID != 0 {
			releaseID = originalRelease.BasicInformation.ID
		}

		// Only update images for existing releases
		if existingMap[releaseID] {
			update := repositories.ReleaseImageUpdate{
				ReleaseID: releaseID,
			}

			// Set thumb if available
			if originalRelease.BasicInformation.Thumb != "" {
				update.Thumb = &originalRelease.BasicInformation.Thumb
			}

			// Set cover image if available
			if originalRelease.BasicInformation.CoverImage != "" {
				update.CoverImage = &originalRelease.BasicInformation.CoverImage
			}

			// Only add update if we have at least one image to update
			if update.Thumb != nil || update.CoverImage != nil {
				imageUpdates = append(imageUpdates, update)
			}
		}
	}

	// Execute image updates
	if len(imageUpdates) > 0 {
		err := f.repos.Release.UpdateReleaseImages(
			ctx,
			f.db.SQLWithContext(ctx),
			imageUpdates,
		)
		if err != nil {
			return log.Err("failed to update release images", err)
		}

		log.Info("Updated release images", "count", len(imageUpdates))
	}

	return nil
}

// TriggerSyncCompletion directly triggers completion of a collection sync
func (f *FoldersService) TriggerSyncCompletion(ctx context.Context, userID uuid.UUID) error {
	log := f.log.Function("TriggerSyncCompletion")

	// Get current sync state from cache
	var syncState CollectionSyncState
	found, err := database.NewCacheBuilder(f.db.Cache.ClientAPI, userID.String()).
		WithHashPattern(COLLECTION_SYNC_HASH).
		WithContext(ctx).
		Get(&syncState)
	if err != nil {
		return log.Err("failed to get sync state from cache", err)
	}
	if !found {
		log.Warn("No active sync state found, completion already processed", "userID", userID)
		return nil
	}

	// Check if sync is ready for completion
	if !syncState.SyncComplete || !syncState.AllReleasesReady {
		log.Warn("Sync not ready for completion",
			"userID", userID,
			"syncComplete", syncState.SyncComplete,
			"allReleasesReady", syncState.AllReleasesReady)
		return nil
	}

	// Analyze what changes need to be made (business logic - outside transaction)
	var operations *SyncCollectionOperations
	operations, err = f.analyzeDifferentialSync(
		ctx,
		userID,
		syncState.MergedReleases,
	)
	if err != nil {
		return log.Err("failed to analyze differential sync", err)
	}

	// Execute sync operations in focused transaction
	err = f.transactionService.Execute(ctx, func(txCtx context.Context, tx *gorm.DB) error {
		return f.executeSyncOperations(txCtx, tx, userID, operations)
	})
	if err != nil {
		return log.Err("failed to execute differential sync", err)
	}

	// Extract basic information in separate transaction to avoid long locks
	folderReleases := make([]DiscogsFolderReleaseItem, 0, len(syncState.OriginalReleases))
	for _, originalRelease := range syncState.OriginalReleases {
		folderReleases = append(folderReleases, originalRelease)
	}

	if len(folderReleases) > 0 {
		err = f.transactionService.Execute(ctx, func(txCtx context.Context, tx *gorm.DB) error {
			return f.folderDataExtractionService.ExtractBasicInformation(txCtx, tx, folderReleases)
		})
		if err != nil {
			log.Warn("Failed to extract basic information", "error", err)
			// Don't fail the sync for data extraction errors
		}
	}

	// Clean up sync state
	_ = database.NewCacheBuilder(f.db.Cache.ClientAPI, userID.String()).
		WithHashPattern(COLLECTION_SYNC_HASH).
		WithContext(ctx).
		Delete()

	log.Info("Collection sync completed successfully",
		"userID", userID,
		"totalReleases", len(syncState.MergedReleases))

	return nil
}
