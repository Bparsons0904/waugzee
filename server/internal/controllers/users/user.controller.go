package userController

import (
	"context"
	"waugzee/config"
	"waugzee/internal/database"
	"waugzee/internal/logger"
	. "waugzee/internal/models"
	"waugzee/internal/repositories"
	"waugzee/internal/services"

	"github.com/google/uuid"
)

type UserController struct {
	userRepo        repositories.UserRepository
	userConfigRepo  repositories.UserConfigurationRepository
	folderRepo      repositories.FolderRepository
	userReleaseRepo repositories.UserReleaseRepository
	discogsService  *services.DiscogsService
	db              database.DB
	Config          config.Config
	log             logger.Logger
}

type GetUserResponse struct {
	Folders  []*Folder      `json:"folders"`
	Releases []*UserRelease `json:"releases"`
}

type UserControllerInterface interface {
	UpdateDiscogsToken(
		ctx context.Context,
		user *User,
		token string,
	) (*User, error)
	GetUser(ctx context.Context, user *User) (*GetUserResponse, error)
	UpdateSelectedFolder(
		ctx context.Context,
		user *User,
		folderID int,
	) (*User, error)
}

func New(
	repos repositories.Repository,
	services services.Service,
	config config.Config,
	db database.DB,
) UserControllerInterface {
	return &UserController{
		userRepo:        repos.User,
		userConfigRepo:  repos.UserConfiguration,
		folderRepo:      repos.Folder,
		userReleaseRepo: repos.UserRelease,
		discogsService:  services.Discogs,
		db:              db,
		Config:          config,
		log:             logger.New("userController"),
	}
}

func (uc *UserController) UpdateDiscogsToken(
	ctx context.Context,
	user *User,
	token string,
) (*User, error) {
	log := uc.log.Function("UpdateDiscogsToken")

	if token == "" {
		return nil, log.ErrMsg("token is required")
	}

	identity, err := uc.discogsService.GetUserIdentity(token)
	if err != nil {
		log.Warn("Invalid Discogs token provided", "userID", user.ID, "error", err)
		return nil, log.Err("invalid discogs token", err)
	}

	config := &UserConfiguration{
		UserID:          user.ID,
		DiscogsToken:    &token,
		DiscogsUsername: &identity.Username,
	}

	if err := uc.userConfigRepo.CreateOrUpdate(ctx, uc.db.SQL, config, uc.userRepo); err != nil {
		return nil, log.Err("failed to update user configuration with discogs credentials", err)
	}

	user.Configuration = config

	log.Info(
		"Discogs credentials updated successfully",
		"userID",
		user.ID,
		"username",
		identity.Username,
	)

	return user, nil
}

func (uc *UserController) GetUser(
	ctx context.Context,
	user *User,
) (*GetUserResponse, error) {
	log := uc.log.Function("GetUser")

	// Get user folders
	folders, err := uc.folderRepo.GetUserFolders(ctx, uc.db.SQL, user.ID)
	if err != nil {
		return nil, log.Err("failed to get user folders", err, "userID", user.ID)
	}

	// Get user releases for selected folder
	var releases []*UserRelease
	if user.Configuration.SelectedFolderID != nil {
		// Get the folder using the composite key lookup
		selectedFolder, err := uc.folderRepo.GetFolderByID(ctx, uc.db.SQL, user.ID, *user.Configuration.SelectedFolderID)
		if err != nil {
			return nil, log.Err(
				"failed to get selected folder",
				err,
				"userID",
				user.ID,
				"folderID",
				*user.Configuration.SelectedFolderID,
			)
		}

		// Use the folder's ID to query user releases
		releases, err = uc.userReleaseRepo.GetUserReleasesByFolderID(
			ctx,
			uc.db.SQL,
			user.ID,
			*selectedFolder.ID,
		)
		if err != nil {
			return nil, log.Err(
				"failed to get user releases",
				err,
				"userID",
				user.ID,
				"folderID",
				*user.Configuration.SelectedFolderID,
				"folderID",
				*selectedFolder.ID,
			)
		}
	}

	return &GetUserResponse{
		Folders:  folders,
		Releases: releases,
	}, nil
}

func (uc *UserController) UpdateSelectedFolder(
	ctx context.Context,
	user *User,
	folderID int,
) (*User, error) {
	log := uc.log.Function("UpdateSelectedFolder")

	// Validate that the folder exists and belongs to the user
	_, err := uc.folderRepo.GetFolderByID(ctx, uc.db.SQL, user.ID, folderID)
	if err != nil {
		return nil, log.Err("folder not found or not owned by user", err)
	}

	user.Configuration.SelectedFolderID = &folderID

	if err := uc.userConfigRepo.Update(ctx, uc.db.SQL, user.Configuration, uc.userRepo); err != nil {
		return nil, log.Err("failed to update user configuration with selected folder", err)
	}

	// Clear the user folders cache since the configuration changed
	if err := uc.clearUserFoldersCache(ctx, user.ID); err != nil {
		log.Warn("failed to clear user folders cache", "userID", user.ID, "error", err)
	}

	log.Info(
		"Selected folder updated successfully",
		"userID",
		user.ID,
		"folderID",
		folderID,
	)

	return user, nil
}

func (uc *UserController) clearUserFoldersCache(ctx context.Context, userID uuid.UUID) error {
	return uc.folderRepo.ClearUserFoldersCache(ctx, userID)
}
