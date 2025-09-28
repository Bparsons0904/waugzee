package repositories

import (
	"context"
	"waugzee/internal/constants"
	"waugzee/internal/database"
	"waugzee/internal/logger"
	. "waugzee/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type FolderRepository interface {
	UpsertFolders(ctx context.Context, tx *gorm.DB, userID uuid.UUID, folders []*Folder) error
	DeleteOrphanFolders(
		ctx context.Context,
		tx *gorm.DB,
		userID uuid.UUID,
		keepDiscogIDs []int,
	) error
	GetUserFolders(ctx context.Context, tx *gorm.DB, userID uuid.UUID) ([]*Folder, error)
	GetFolderByDiscogID(
		ctx context.Context,
		tx *gorm.DB,
		userID uuid.UUID,
		discogID int,
	) (*Folder, error)
	ClearUserFoldersCache(ctx context.Context, userID uuid.UUID) error
}

type folderRepository struct {
	cache database.DB
	log   logger.Logger
}

func NewFolderRepository(cache database.DB) FolderRepository {
	return &folderRepository{
		cache: cache,
		log:   logger.New("folderRepository"),
	}
}

func (r *folderRepository) UpsertFolders(
	ctx context.Context,
	tx *gorm.DB,
	userID uuid.UUID,
	folders []*Folder,
) error {
	log := r.log.Function("UpsertFolders")

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
				{Name: "discog_id"},
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

	return nil
}

func (r *folderRepository) DeleteOrphanFolders(
	ctx context.Context,
	tx *gorm.DB,
	userID uuid.UUID,
	keepDiscogIDs []int,
) error {
	log := r.log.Function("DeleteOrphanFolders")

	if len(keepDiscogIDs) == 0 {
		// Delete all user folders if no IDs to keep
		result := tx.WithContext(ctx).
			Where("user_id = ?", userID).
			Delete(&Folder{})

		if result.Error != nil {
			return log.Err("failed to delete all user folders", result.Error, "userID", userID)
		}

		return nil
	}

	result := tx.WithContext(ctx).
		Where("user_id = ? AND discog_id NOT IN ?", userID, keepDiscogIDs).
		Delete(&Folder{})

	if result.Error != nil {
		return log.Err(
			"failed to delete orphan folders",
			result.Error,
			"userID",
			userID,
			"keepCount",
			len(keepDiscogIDs),
		)
	}

	return nil
}

func (r *folderRepository) GetUserFolders(
	ctx context.Context,
	tx *gorm.DB,
	userID uuid.UUID,
) ([]*Folder, error) {
	log := r.log.Function("GetUserFolders")

	// Try to get from cache first
	var cachedFolders []*Folder
	found, err := database.NewCacheBuilder(r.cache.Cache.User, userID.String()).
		WithContext(ctx).
		WithHash(constants.UserFoldersCachePrefix).
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

	// Cache the folders
	if err := database.NewCacheBuilder(r.cache.Cache.User, userID.String()).
		WithContext(ctx).
		WithHash(constants.UserFoldersCachePrefix).
		WithStruct(folders).
		WithTTL(constants.UserCacheExpiry).
		Set(); err != nil {
		log.Warn("failed to cache user folders", "userID", userID, "error", err)
	}

	log.Info("Retrieved user folders", "userID", userID, "folderCount", len(folders))
	return folders, nil
}

func (r *folderRepository) GetFolderByDiscogID(
	ctx context.Context,
	tx *gorm.DB,
	userID uuid.UUID,
	discogID int,
) (*Folder, error) {
	log := r.log.Function("GetFolderByDiscogID")

	var folder Folder
	if err := tx.WithContext(ctx).
		Where("user_id = ? AND discog_id = ?", userID, discogID).
		First(&folder).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, log.Err("folder not found", err, "userID", userID, "discogID", discogID)
		}
		return nil, log.Err(
			"failed to get folder by discog ID",
			err,
			"userID",
			userID,
			"discogID",
			discogID,
		)
	}

	return &folder, nil
}

func (r *folderRepository) ClearUserFoldersCache(ctx context.Context, userID uuid.UUID) error {
	log := r.log.Function("ClearUserFoldersCache")

	if err := database.NewCacheBuilder(r.cache.Cache.User, userID.String()).
		WithContext(ctx).
		WithHash(constants.UserFoldersCachePrefix).
		Delete(); err != nil {
		log.Warn("failed to clear user folders cache", "userID", userID, "error", err)
		return err
	}

	log.Info("cleared user folders cache", "userID", userID)
	return nil
}
