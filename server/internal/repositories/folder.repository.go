package repositories

import (
	"context"
	"fmt"
	"waugzee/internal/database"
	logger "github.com/Bparsons0904/goLogger"
	. "waugzee/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	USER_FOLDERS_CACHE_PREFIX = "user_folders"
	USER_FOLDER_CACHE_PREFIX  = "user_folder"
	USER_FOLDER_CACHE_EXPIRY  = USER_CACHE_EXPIRY
)

type FolderRepository interface {
	UpsertFolders(ctx context.Context, tx *gorm.DB, userID uuid.UUID, folders []*Folder) error
	DeleteOrphanFolders(
		ctx context.Context,
		tx *gorm.DB,
		userID uuid.UUID,
		keepFolderIDs []int,
	) error
	GetUserFolders(ctx context.Context, tx *gorm.DB, userID uuid.UUID) ([]*Folder, error)
	GetFolderByID(
		ctx context.Context,
		tx *gorm.DB,
		userID uuid.UUID,
		folderID int,
	) (*Folder, error)
	ClearUserFoldersCache(ctx context.Context, userID uuid.UUID) error
}

type folderRepository struct {
	cache database.DB
}

func NewFolderRepository(cache database.DB) FolderRepository {
	return &folderRepository{
		cache: cache,
	}
}

func (r *folderRepository) UpsertFolders(
	ctx context.Context,
	tx *gorm.DB,
	userID uuid.UUID,
	folders []*Folder,
) error {
	log := logger.New("folderRepository").TraceFromContext(ctx).Function("UpsertFolders")

	log.Info("Upserting folders", "userID", userID, "folderCount", len(folders))
	if len(folders) == 0 {
		log.Info("No folders to upsert")
		return nil
	}

	// Ensure all folders have the correct UserID
	for _, folder := range folders {
		folder.UserID = userID
	}

	if err := tx.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "id"},
				{Name: "user_id"},
			},
			DoUpdates: clause.AssignmentColumns([]string{
				"name",
				"count",
				"resource_url",
				"updated_at",
			}),
		}).
		Create(folders).Error; err != nil {
		return log.Err(
			"failed to upsert folders",
			err,
			"userID",
			userID,
			"folderCount",
			len(folders),
		)
	}

	if err := r.ClearUserFoldersCache(ctx, userID); err != nil {
		log.Warn("failed to clear folders cache", "userID", userID, "error", err)
	}
	r.clearIndividualFolderCaches(ctx, userID, folders)

	return nil
}

func (r *folderRepository) DeleteOrphanFolders(
	ctx context.Context,
	tx *gorm.DB,
	userID uuid.UUID,
	keepFolderIDs []int,
) error {
	log := logger.New("folderRepository").TraceFromContext(ctx).Function("DeleteOrphanFolders")

	if len(keepFolderIDs) == 0 {
		// Delete all user folders if no IDs to keep
		result := tx.WithContext(ctx).
			Where("user_id = ?", userID).
			Delete(&Folder{})

		if result.Error != nil {
			return log.Err("failed to delete all user folders", result.Error, "userID", userID)
		}

		if err := r.ClearUserFoldersCache(ctx, userID); err != nil {
			log.Warn("failed to clear folders cache", "userID", userID, "error", err)
		}

		return nil
	}

	result := tx.WithContext(ctx).
		Where("user_id = ? AND id NOT IN ?", userID, keepFolderIDs).
		Delete(&Folder{})

	if result.Error != nil {
		return log.Err(
			"failed to delete orphan folders",
			result.Error,
			"userID",
			userID,
			"keepCount",
			len(keepFolderIDs),
		)
	}

	if err := r.ClearUserFoldersCache(ctx, userID); err != nil {
		log.Warn("failed to clear folders cache", "userID", userID, "error", err)
	}

	return nil
}

func (r *folderRepository) GetUserFolders(
	ctx context.Context,
	tx *gorm.DB,
	userID uuid.UUID,
) ([]*Folder, error) {
	log := logger.New("folderRepository").TraceFromContext(ctx).Function("GetUserFolders")

	var cachedFolders []*Folder
	found, err := database.NewCacheBuilder(r.cache.Cache.User, userID).
		WithContext(ctx).
		WithHash(USER_FOLDERS_CACHE_PREFIX).
		Get(&cachedFolders)
	if err == nil && found {
		log.Info("user folders found in cache", "userID", userID)
		return cachedFolders, nil
	}

	var folders []*Folder
	if err := tx.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("name ASC").
		Find(&folders).Error; err != nil {
		return nil, log.Err("failed to get user folders", err, "userID", userID)
	}

	if err := database.NewCacheBuilder(r.cache.Cache.User, userID).
		WithContext(ctx).
		WithHash(USER_FOLDERS_CACHE_PREFIX).
		WithStruct(folders).
		WithTTL(USER_CACHE_EXPIRY).
		Set(); err != nil {
		log.Warn("failed to cache user folders", "userID", userID, "error", err)
	}

	log.Info("Retrieved user folders", "userID", userID, "folderCount", len(folders))
	return folders, nil
}

func (r *folderRepository) GetFolderByID(
	ctx context.Context,
	tx *gorm.DB,
	userID uuid.UUID,
	folderID int,
) (*Folder, error) {
	log := logger.New("folderRepository").TraceFromContext(ctx).Function("GetFolderByID")

	cacheKey := fmt.Sprintf("%s:%d", userID.String(), folderID)
	var cachedFolder Folder
	found, err := database.NewCacheBuilder(r.cache.Cache.User, cacheKey).
		WithContext(ctx).
		WithHash(USER_FOLDER_CACHE_PREFIX).
		Get(&cachedFolder)
	if err == nil && found {
		log.Info("folder found in cache", "userID", userID, "folderID", folderID)
		return &cachedFolder, nil
	}

	var folder Folder
	if err := tx.WithContext(ctx).
		Where("user_id = ? AND id = ?", userID, folderID).
		First(&folder).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, log.Err("folder not found", err, "userID", userID, "folderID", folderID)
		}
		return nil, log.Err(
			"failed to get folder by ID",
			err,
			"userID",
			userID,
			"folderID",
			folderID,
		)
	}

	if err := database.NewCacheBuilder(r.cache.Cache.User, cacheKey).
		WithContext(ctx).
		WithHash(USER_FOLDER_CACHE_PREFIX).
		WithStruct(&folder).
		WithTTL(USER_FOLDER_CACHE_EXPIRY).
		Set(); err != nil {
		log.Warn("failed to cache folder", "userID", userID, "folderID", folderID, "error", err)
	}

	log.Info("Retrieved folder by ID from database and cached", "userID", userID, "folderID", folderID)
	return &folder, nil
}

func (r *folderRepository) ClearUserFoldersCache(ctx context.Context, userID uuid.UUID) error {
	log := logger.New("folderRepository").TraceFromContext(ctx).Function("ClearUserFoldersCache")

	if err := database.NewCacheBuilder(r.cache.Cache.User, userID.String()).
		WithContext(ctx).
		WithHash(USER_FOLDERS_CACHE_PREFIX).
		Delete(); err != nil {
		log.Warn("failed to clear user folders cache", "userID", userID, "error", err)
		return err
	}

	log.Info("cleared user folders cache", "userID", userID)
	return nil
}

func (r *folderRepository) clearIndividualFolderCaches(
	ctx context.Context,
	userID uuid.UUID,
	folders []*Folder,
) {
	log := logger.New("folderRepository").TraceFromContext(ctx).Function("clearIndividualFolderCaches")

	for _, folder := range folders {
		if folder.ID != nil {
			cacheKey := fmt.Sprintf("%s:%d", userID.String(), *folder.ID)
			err := database.NewCacheBuilder(r.cache.Cache.User, cacheKey).
				WithContext(ctx).
				WithHash(USER_FOLDER_CACHE_PREFIX).
				Delete()
			if err != nil {
				log.Warn("failed to clear individual folder cache", "userID", userID, "folderID", *folder.ID, "error", err)
			}
		}
	}
}
