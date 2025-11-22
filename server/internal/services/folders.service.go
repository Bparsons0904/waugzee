package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"
	"waugzee/internal/database"
	"waugzee/internal/events"
	logger "github.com/Bparsons0904/goLogger"
	. "waugzee/internal/models"
	"waugzee/internal/repositories"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type FoldersService struct {
	log                         logger.Logger
	eventBus                    *events.EventBus
	repos                       repositories.Repository
	db                          database.DB
	transactionService          *TransactionService
	folderDataExtractionService *FolderDataExtractionService
	folderValidationService     *FolderValidationService
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
	folderValidationService := NewFolderValidationService(repos, db)
	return &FoldersService{
		log:                         log,
		eventBus:                    eventBus,
		repos:                       repos,
		db:                          db,
		transactionService:          transactionService,
		folderDataExtractionService: folderDataExtractionService,
		folderValidationService:     folderValidationService,
		discogsRateLimiter:          discogsRateLimiter,
	}
}

func (f *FoldersService) RequestUserFolders(
	ctx context.Context,
	user *User,
) (string, error) {
	log := f.log.Function("RequestUserFolders")

	if err := f.folderValidationService.ValidateUserForFolderOperation(user); err != nil {
		return "", err
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
		// Create a copy of the ID to avoid taking address of loop variable
		folderID := discogsFolder.ID
		folder := &Folder{
			ID:          &folderID,
			UserID:      metadata.UserID,
			Name:        discogsFolder.Name,
			Count:       discogsFolder.Count,
			ResourceURL: discogsFolder.ResourceURL,
		}
		folders = append(folders, folder)
	}

	keepFolderIDs, _ := f.extractFolderSyncData(folders)

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
		return f.repos.Folder.DeleteOrphanFolders(txCtx, tx, metadata.UserID, keepFolderIDs)
	})
	if err != nil {
		return log.Err("failed to delete orphan folders", err,
			"userID", metadata.UserID,
			"requestID", metadata.RequestID)
	}

	// Clear folder cache after upsert and cleanup
	if err := f.repos.Folder.ClearUserFoldersCache(ctx, metadata.UserID); err != nil {
		log.Warn("failed to clear folder cache after sync", "error", err)
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

	// Trigger collection sync now that folders are saved
	user, err := f.repos.User.GetByID(ctx, f.db.SQLWithContext(ctx), metadata.UserID)
	if err != nil {
		log.Warn("Failed to fetch user for collection sync trigger", "error", err)
		return nil
	}

	err = f.SyncAllUserFolders(ctx, user)
	if err != nil {
		log.Warn("Failed to trigger collection sync after folder discovery",
			"userID", metadata.UserID,
			"error", err)
	}

	return nil
}

// processReleasesToSyncState converts Discogs release data to UserRelease models
// and accumulates them in the sync state for later processing.
func (f *FoldersService) processReleasesToSyncState(
	syncState *CollectionSyncState,
	releases []DiscogsFolderReleaseItem,
	userID uuid.UUID,
	folderID int,
) []int64 {
	log := f.log.Function("processReleasesToSyncState")

	if syncState == nil {
		log.Warn("syncState is nil, cannot process releases")
		return []int64{}
	}

	if len(releases) == 0 {
		log.Debug("No releases to process")
		return []int64{}
	}

	missingReleaseIDs := make([]int64, 0)

	for _, discogsRelease := range releases {
		releaseID := discogsRelease.ID
		if releaseID == 0 && discogsRelease.BasicInformation.ID != 0 {
			releaseID = discogsRelease.BasicInformation.ID
		}

		if releaseID == 0 {
			log.Warn("Skipping release with no valid ID", "instanceID", discogsRelease.InstanceID)
			continue
		}

		notesJSON, _ := json.Marshal(discogsRelease.Notes)

		// Parse DateAdded from Discogs API response
		var dateAdded time.Time
		if discogsRelease.DateAdded != "" {
			if parsedDate, err := time.Parse(time.RFC3339, discogsRelease.DateAdded); err == nil {
				dateAdded = parsedDate
			} else {
				log.Warn("Failed to parse DateAdded from Discogs API, using current time",
					"dateAdded", discogsRelease.DateAdded,
					"instanceID", discogsRelease.InstanceID,
					"error", err)
				dateAdded = time.Now()
			}
		} else {
			dateAdded = time.Now()
		}

		userRelease := &UserRelease{
			UserID:     userID,
			ReleaseID:  releaseID,
			InstanceID: discogsRelease.InstanceID,
			FolderID:   folderID,
			Rating:     discogsRelease.Rating,
			Notes:      datatypes.JSON(notesJSON),
			DateAdded:  dateAdded,
			Active:     true,
		}

		// Check collection size limits to prevent memory issues
		if len(syncState.MergedReleases) >= MaxReleasesPerSync {
			log.Warn("Reached maximum releases per sync, skipping additional releases",
				"maxReleases", MaxReleasesPerSync,
				"instanceID", discogsRelease.InstanceID)
			break
		}

		// Add to merged releases (overwrite if exists - latest folder wins)
		syncState.MergedReleases[discogsRelease.InstanceID] = userRelease
		// Store original data for data extraction
		syncState.OriginalReleases[discogsRelease.InstanceID] = discogsRelease
		missingReleaseIDs = append(missingReleaseIDs, releaseID)
	}

	return missingReleaseIDs
}

func (f *FoldersService) extractFolderSyncData(
	folders []*Folder,
) (keepFolderIDs []int, allFolderID *int) {
	keepFolderIDs = make([]int, 0, len(folders))
	for _, folder := range folders {
		if folder.ID != nil {
			keepFolderIDs = append(keepFolderIDs, *folder.ID)
			if folder.Name == "All" {
				allFolderID = folder.ID
			}
		}
	}
	return keepFolderIDs, allFolderID
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

	// Set to Uncategorized folder as default instead of All folder
	defaultFolderID := UncategorizedFolderID
	userConfig.SelectedFolderID = &defaultFolderID
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

	if err := f.folderValidationService.ValidateUserForFolderOperation(user); err != nil {
		return "", err
	}

	validatedPage, err := f.folderValidationService.ValidatePaginationParams(folderID, page)
	if err != nil {
		return "", log.Err("invalid pagination parameters", err)
	}
	page = validatedPage

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
		FolderID:     &folderID,
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
		f.clearSyncStateOnError(ctx, metadata.UserID, "failed to process API response")
		return nil // Don't return error as this is an expected API failure
	}

	if metadata.FolderID == nil {
		f.clearSyncStateOnError(ctx, metadata.UserID, "missing folder ID in request metadata")
		return log.ErrMsg("missing folder ID in request metadata")
	}
	folderID := *metadata.FolderID

	// Get current sync state from cache
	var syncState CollectionSyncState
	var missingReleaseIDs []int64
	found, err := database.NewCacheBuilder(f.db.Cache.ClientAPI, metadata.UserID.String()).
		WithHashPattern(COLLECTION_SYNC_HASH).
		WithContext(ctx).
		Get(&syncState)
	if err != nil {
		f.clearSyncStateOnError(ctx, metadata.UserID, "failed to get sync state from cache")
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

	// Process releases and accumulate to sync state
	missingReleaseIDs = f.processReleasesToSyncState(
		&syncState,
		discogsFolderReleasesResponse.Data.Releases,
		metadata.UserID,
		folderID,
	)

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
				FolderID:     &folderID,
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
				f.clearSyncStateOnError(ctx, metadata.UserID, "failed to perform release validation")
				return log.Err("failed to perform release validation", err)
			}

			log.Info("Release validation complete",
				"existingReleases", len(syncState.ExistingReleaseIDs),
				"missingReleases", len(syncState.MissingReleaseIDs),
				"allReleasesReady", syncState.AllReleasesReady)

			// Update sync state in cache
			err = database.NewCacheBuilder(f.db.Cache.ClientAPI, metadata.UserID.String()).
				WithHashPattern(COLLECTION_SYNC_HASH).
				WithStruct(syncState).
				WithTTL(SyncStateTTL).
				WithContext(ctx).
				Set()
			if err != nil {
				f.clearSyncStateOnError(ctx, metadata.UserID, "failed to update sync state after validation")
				log.Warn("Failed to update sync state after validation", "error", err)
			}

			// If we have missing releases, return early to wait for API responses
			if len(syncState.MissingReleaseIDs) > 0 && !syncState.AllReleasesReady {
				return nil
			}
		}

		// Check if all releases are ready before proceeding
		if !syncState.AllReleasesReady && len(syncState.PendingReleaseRequests) > 0 {
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
			f.clearSyncStateOnError(ctx, metadata.UserID, "failed to analyze differential sync")
			return log.Err("failed to analyze differential sync", err)
		}


		// Execute sync operations in focused transaction
		err = f.transactionService.Execute(ctx, func(txCtx context.Context, tx *gorm.DB) error {
			return f.executeSyncOperations(txCtx, tx, metadata.UserID, operations)
		})
		if err != nil {
			f.clearSyncStateOnError(ctx, metadata.UserID, "failed to execute differential sync")
			return log.Err("failed to execute differential sync", err)
		}

		// Extract basic information in separate transaction to avoid long locks
		folderReleases := make([]DiscogsFolderReleaseItem, 0, len(syncState.OriginalReleases))
		for _, originalRelease := range syncState.OriginalReleases {
			folderReleases = append(folderReleases, originalRelease)
		}

		if len(folderReleases) > 0 {
			err = f.transactionService.Execute(ctx, func(txCtx context.Context, tx *gorm.DB) error {
				return f.folderDataExtractionService.ExtractBasicInformation(
					txCtx,
					tx,
					folderReleases,
				)
			})
			if err != nil {
				log.Warn("Failed to extract basic information", "error", err)
				// Don't fail the sync for data extraction errors
			}
		}

		// Clean up sync state and release queue
		_ = database.NewCacheBuilder(f.db.Cache.ClientAPI, metadata.UserID.String()).
			WithHashPattern(COLLECTION_SYNC_HASH).
			WithContext(ctx).
			Delete()

		_ = database.NewCacheBuilder(f.db.Cache.ClientAPI, metadata.UserID.String()).
			WithHashPattern(RELEASE_QUEUE_HASH).
			WithContext(ctx).
			Delete()

		log.Info("Collection sync completed successfully",
			"userID", metadata.UserID,
			"totalReleases", len(syncState.MergedReleases))

		// Clear user cache to ensure frontend gets fresh data
		if err = f.repos.User.ClearUserCacheByUserID(ctx, f.db.SQLWithContext(ctx), metadata.UserID.String()); err != nil {
			log.Warn("Failed to clear user cache after sync completion", "error", err)
		}

		log.Info("Sync completed - user will receive recommendation on next login", "userID", metadata.UserID)

		// Send sync_complete event to notify client
		completeMessage := events.Message{
			ID:      metadata.UserID.String(),
			Service: events.USER,
			Event:   "sync_complete",
			UserID:  metadata.UserID.String(),
			Payload: map[string]any{
				"message":       "Collection sync completed successfully",
				"totalReleases": len(syncState.MergedReleases),
			},
			Timestamp: time.Now(),
		}
		if err = f.eventBus.Publish(events.WEBSOCKET, "user", completeMessage); err != nil {
			log.Warn("Failed to send sync_complete event", "error", err)
		}
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

	// Queue missing releases for processing only if sync is not complete
	// If sync is complete, we've already processed everything we can
	if len(missingReleaseIDs) > 0 && !syncState.SyncComplete {
		err = f.queueMissingReleases(ctx, metadata.UserID, missingReleaseIDs)
		if err != nil {
			log.Warn("Failed to queue missing releases",
				"error", err,
				"releaseCount", len(missingReleaseIDs))
		}
	}

	return nil
}

// processIndividualFolder handles processing a single folder when not part of collection sync.
// This is used for standalone folder processing outside of the full sync workflow.
func (f *FoldersService) processIndividualFolder(
	ctx context.Context,
	metadata RequestMetadata,
	folderID int,
	discogsFolderReleasesResponse *DiscogsFolderReleasesResponse,
) error {
	log := f.log.Function("processIndividualFolder")

	// Process individual folder when not part of collection sync
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
	SyncOperationID  string    // Unique ID for this sync operation (idempotency)
	StartedAt        time.Time // When sync operation started
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

	// Check if sync is already in progress (idempotency check)
	var existingSyncState CollectionSyncState
	found, err := database.NewCacheBuilder(f.db.Cache.ClientAPI, user.ID.String()).
		WithHashPattern(COLLECTION_SYNC_HASH).
		WithContext(ctx).
		Get(&existingSyncState)
	if err != nil {
		log.Warn("Failed to check for existing sync state", "error", err)
	} else if found && !existingSyncState.SyncComplete {
		// Check if sync is stale (older than MaxCollectionSyncTTL)
		syncAge := time.Since(existingSyncState.StartedAt)
		if syncAge > MaxCollectionSyncTTL {
			log.Warn("Clearing stale sync state",
				"userID", user.ID,
				"syncOperationID", existingSyncState.SyncOperationID,
				"age", syncAge)
			// Clear stale sync state
			_ = database.NewCacheBuilder(f.db.Cache.ClientAPI, user.ID.String()).
				WithHashPattern(COLLECTION_SYNC_HASH).
				WithContext(ctx).
				Delete()
		} else {
			log.Info("Sync already in progress for user",
				"userID", user.ID,
				"syncOperationID", existingSyncState.SyncOperationID,
				"startedAt", existingSyncState.StartedAt)
			return nil // Don't start duplicate sync
		}
	}

	folders, err := f.repos.Folder.GetUserFolders(ctx, f.db.SQLWithContext(ctx), user.ID)
	if err != nil {
		return log.Err("failed to get user folders", err)
	}

	if len(folders) == 0 {
		return log.ErrMsg("no folders to sync")
	}

	// Count folders with valid IDs (folders with nil ID or ID=0 cannot be synced)
	// Folder 0 is the "All" folder which is a virtual aggregation of other folders
	validFolderCount := 0
	for _, folder := range folders {
		if folder.ID != nil && *folder.ID != 0 {
			validFolderCount++
		}
	}

	if validFolderCount == 0 {
		return log.ErrMsg("no valid folders to sync (all folders have nil ID or are folder 0)")
	}

	// Generate unique sync operation ID for idempotency
	syncOperationID := uuid.New().String()

	// Initialize sync state
	syncState := &CollectionSyncState{
		UserID:                 user.ID,
		SyncOperationID:        syncOperationID,
		StartedAt:              time.Now(),
		TotalFolders:           validFolderCount,
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
		WithTTL(MaxCollectionSyncTTL).
		WithContext(ctx).
		Set()
	if err != nil {
		return log.Err("failed to store sync state", err)
	}

	// Start sync for each folder (page 1)
	foldersRequested := 0
	for _, folder := range folders {
		if folder.ID == nil {
			log.Warn("Skipping folder with nil ID", "folderName", folder.Name)
			continue
		}

		// Skip folder 0 (All folder) - it's a virtual folder that aggregates all other folders
		// and doesn't support the folder releases API endpoint
		if *folder.ID == 0 {
			log.Info("Skipping folder 0 (All folder) - virtual aggregation folder",
				"folderName", folder.Name)
			continue
		}

		_, err = f.RequestFolderReleases(ctx, user, *folder.ID, 1)
		if err != nil {
			log.Warn("Failed to start sync for folder",
				"folderID", *folder.ID,
				"error", err)
		} else {
			foldersRequested++
		}
	}

	log.Info("Started folder sync requests",
		"totalFolders", len(folders),
		"foldersRequested", foldersRequested,
		"syncOperationID", syncOperationID)

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

	// Get existing release IDs to validate foreign key constraints
	releaseIDs := make([]int64, 0, len(mergedReleases))
	for _, userRelease := range mergedReleases {
		releaseIDs = append(releaseIDs, userRelease.ReleaseID)
	}

	existingReleaseIDs, _, err := f.folderValidationService.ValidateReleases(ctx, releaseIDs)
	if err != nil {
		return nil, log.Err("failed to validate release IDs", err)
	}

	// Create a map for fast lookup of existing release IDs
	existingReleaseMap := make(map[int64]bool, len(existingReleaseIDs))
	for _, releaseID := range existingReleaseIDs {
		existingReleaseMap[releaseID] = true
	}

	operations := &SyncCollectionOperations{
		Create: make([]UserRelease, 0),
		Update: make([]UserRelease, 0),
		Delete: make([]int, 0),
	}

	skippedCount := 0

	// Find creates and updates
	for instanceID, mergedRelease := range mergedReleases {
		// Skip if the release doesn't exist in the database (foreign key constraint)
		if !existingReleaseMap[mergedRelease.ReleaseID] {
			skippedCount++
			continue
		}

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

	if skippedCount > 0 {
		log.Warn("Skipped user releases with missing release records",
			"skippedCount", skippedCount,
			"totalMerged", len(mergedReleases))
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
		log.Debug("No releases to validate")
		syncState.ReleaseValidationDone = true
		syncState.AllReleasesReady = true
		return nil
	}

	// Use validation service to check release existence
	existingReleases, missingReleases, err := f.folderValidationService.ValidateReleases(
		ctx,
		releaseIDs,
	)
	if err != nil {
		return log.Err("failed to validate releases", err)
	}

	syncState.ExistingReleaseIDs = existingReleases
	syncState.MissingReleaseIDs = missingReleases
	syncState.ReleaseValidationDone = true

	// Update images for existing releases
	if len(existingReleases) > 0 {
		err = f.folderValidationService.UpdateReleaseImages(
			ctx,
			syncState.OriginalReleases,
			existingReleases,
		)
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
		f.clearSyncStateOnError(ctx, userID, "failed to analyze differential sync in completion")
		return log.Err("failed to analyze differential sync", err)
	}

	// Execute sync operations in focused transaction
	err = f.transactionService.Execute(ctx, func(txCtx context.Context, tx *gorm.DB) error {
		return f.executeSyncOperations(txCtx, tx, userID, operations)
	})
	if err != nil {
		f.clearSyncStateOnError(ctx, userID, "failed to execute differential sync in completion")
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

	// Clean up sync state and release queue
	_ = database.NewCacheBuilder(f.db.Cache.ClientAPI, userID.String()).
		WithHashPattern(COLLECTION_SYNC_HASH).
		WithContext(ctx).
		Delete()

	_ = database.NewCacheBuilder(f.db.Cache.ClientAPI, userID.String()).
		WithHashPattern(RELEASE_QUEUE_HASH).
		WithContext(ctx).
		Delete()

	log.Info("Collection sync completed successfully",
		"userID", userID,
		"totalReleases", len(syncState.MergedReleases))

	// Clear user cache to ensure frontend gets fresh data
	if err = f.repos.User.ClearUserCacheByUserID(ctx, f.db.SQLWithContext(ctx), userID.String()); err != nil {
		log.Warn("Failed to clear user cache after sync completion", "error", err)
	}

	log.Info("Sync completed - user will receive recommendation on next login", "userID", userID)

	// Send sync_complete event to notify client
	completeMessage := events.Message{
		ID:      userID.String(),
		Service: events.USER,
		Event:   "sync_complete",
		UserID:  userID.String(),
		Payload: map[string]any{
			"message":       "Collection sync completed successfully",
			"totalReleases": len(syncState.MergedReleases),
		},
		Timestamp: time.Now(),
	}
	if err = f.eventBus.Publish(events.WEBSOCKET, "user", completeMessage); err != nil {
		log.Warn("Failed to send sync_complete event", "error", err)
	}

	return nil
}

func (f *FoldersService) ClearSyncState(ctx context.Context, userID uuid.UUID) error {
	log := f.log.Function("ClearSyncState")

	err := database.NewCacheBuilder(f.db.Cache.ClientAPI, userID.String()).
		WithHashPattern(COLLECTION_SYNC_HASH).
		WithContext(ctx).
		Delete()

	if err != nil {
		return log.Err("failed to clear sync state", err)
	}

	log.Info("Sync state cleared successfully", "userID", userID)
	return nil
}

// clearSyncStateOnError clears the sync state cache when an error occurs during processing
func (f *FoldersService) clearSyncStateOnError(ctx context.Context, userID uuid.UUID, reason string) {
	log := f.log.Function("clearSyncStateOnError")

	err := database.NewCacheBuilder(f.db.Cache.ClientAPI, userID.String()).
		WithHashPattern(COLLECTION_SYNC_HASH).
		WithContext(ctx).
		Delete()

	if err != nil {
		log.Warn("Failed to clear sync state on error",
			"userID", userID,
			"reason", reason,
			"error", err)
	} else {
		log.Info("Cleared sync state due to error",
			"userID", userID,
			"reason", reason)
	}

	// Send sync_error event to notify client
	errorMessage := events.Message{
		ID:      userID.String(),
		Service: events.USER,
		Event:   "sync_error",
		UserID:  userID.String(),
		Payload: map[string]any{
			"error":   "Sync failed",
			"message": reason,
		},
		Timestamp: time.Now(),
	}
	if err = f.eventBus.Publish(events.WEBSOCKET, "user", errorMessage); err != nil {
		log.Warn("Failed to send sync_error event", "error", err)
	}
}
