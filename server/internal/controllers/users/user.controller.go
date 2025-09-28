package userController

import (
	"context"
	"waugzee/config"
	"waugzee/internal/constants"
	"waugzee/internal/database"
	"waugzee/internal/logger"
	. "waugzee/internal/models"
	"waugzee/internal/repositories"
	"waugzee/internal/services"
)

type UserController struct {
	userRepo       repositories.UserRepository
	userConfigRepo repositories.UserConfigurationRepository
	folderRepo     repositories.FolderRepository
	discogsService *services.DiscogsService
	db             database.DB
	Config         config.Config
	log            logger.Logger
}

type UserControllerInterface interface {
	UpdateDiscogsToken(ctx context.Context, user *User, req UpdateDiscogsTokenRequest) (*User, error)
	GetUserWithFolders(ctx context.Context, user *User) (*UserWithFoldersResponse, error)
	UpdateSelectedFolder(ctx context.Context, user *User, req UpdateSelectedFolderRequest) (*User, error)
}

func New(
	repos repositories.Repository,
	services services.Service,
	config config.Config,
	db database.DB,
) UserControllerInterface {
	return &UserController{
		userRepo:       repos.User,
		userConfigRepo: repos.UserConfiguration,
		folderRepo:     repos.Folder,
		discogsService: services.Discogs,
		db:             db,
		Config:         config,
		log:            logger.New("userController"),
	}
}

type UpdateDiscogsTokenRequest struct {
	Token string `json:"token"`
}

type UpdateSelectedFolderRequest struct {
	FolderID int `json:"folderId"`
}

type UserWithFoldersResponse struct {
	User    *User     `json:"user"`
	Folders []*Folder `json:"folders"`
}

func (uc *UserController) UpdateDiscogsToken(
	ctx context.Context,
	user *User,
	req UpdateDiscogsTokenRequest,
) (*User, error) {
	log := uc.log.Function("UpdateDiscogsToken")

	if req.Token == "" {
		return nil, log.ErrMsg("token is required")
	}

	identity, err := uc.discogsService.GetUserIdentity(req.Token)
	if err != nil {
		log.Warn("Invalid Discogs token provided", "userID", user.ID, "error", err)
		return nil, log.Err("invalid discogs token", err)
	}

	// Create or update user configuration
	config := &UserConfiguration{
		UserID:          user.ID,
		DiscogsToken:    &req.Token,
		DiscogsUsername: &identity.Username,
	}

	if err := uc.userConfigRepo.CreateOrUpdate(ctx, uc.db.SQL, config, uc.userRepo); err != nil {
		return nil, log.Err("failed to update user configuration with discogs credentials", err)
	}

	// Update user's configuration relationship
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

func (uc *UserController) GetUserWithFolders(
	ctx context.Context,
	user *User,
) (*UserWithFoldersResponse, error) {
	log := uc.log.Function("GetUserWithFolders")

	// Try to get from cache first
	cacheKey := constants.UserWithFoldersCachePrefix + user.OIDCUserID
	var cachedResponse UserWithFoldersResponse
	found, err := database.NewCacheBuilder(uc.db.Cache.User, cacheKey).WithContext(ctx).Get(&cachedResponse)
	if err == nil && found {
		log.Info("user with folders found in cache", "oidcUserID", user.OIDCUserID)
		return &cachedResponse, nil
	}

	// Get user's folders from database
	folders, err := uc.folderRepo.GetUserFolders(ctx, uc.db.SQL, user.ID)
	if err != nil {
		return nil, log.Err("failed to get user folders", err, "userID", user.ID)
	}

	response := &UserWithFoldersResponse{
		User:    user,
		Folders: folders,
	}

	// Cache the response
	if err := database.NewCacheBuilder(uc.db.Cache.User, cacheKey).
		WithStruct(response).
		WithTTL(constants.UserCacheExpiry).
		WithContext(ctx).
		Set(); err != nil {
		log.Warn("failed to cache user with folders", "oidcUserID", user.OIDCUserID, "error", err)
	}

	log.Info("Retrieved user with folders", "userID", user.ID, "folderCount", len(folders))

	return response, nil
}

func (uc *UserController) UpdateSelectedFolder(
	ctx context.Context,
	user *User,
	req UpdateSelectedFolderRequest,
) (*User, error) {
	log := uc.log.Function("UpdateSelectedFolder")

	// Validate that the folder belongs to the user
	_, err := uc.folderRepo.GetFolderByDiscogID(ctx, uc.db.SQL, user.ID, req.FolderID)
	if err != nil {
		log.Warn("Attempted to select folder not owned by user", "userID", user.ID, "folderID", req.FolderID)
		return nil, log.Err("folder not found or not owned by user", err)
	}

	// Create or update user configuration with the selected folder
	config := user.Configuration
	if config == nil {
		config = &UserConfiguration{
			UserID:           user.ID,
			SelectedFolderID: &req.FolderID,
		}
	} else {
		config.SelectedFolderID = &req.FolderID
	}

	if err := uc.userConfigRepo.CreateOrUpdate(ctx, uc.db.SQL, config, uc.userRepo); err != nil {
		return nil, log.Err("failed to update user configuration with selected folder", err)
	}

	// Update user's configuration relationship
	user.Configuration = config

	// Clear the user with folders cache since the configuration changed
	userWithFoldersCacheKey := constants.UserWithFoldersCachePrefix + user.OIDCUserID
	if err := database.NewCacheBuilder(uc.db.Cache.User, userWithFoldersCacheKey).WithContext(ctx).Delete(); err != nil {
		log.Warn("failed to clear user with folders cache", "oidcUserID", user.OIDCUserID, "error", err)
	}

	log.Info(
		"Selected folder updated successfully",
		"userID",
		user.ID,
		"folderID",
		req.FolderID,
	)

	return user, nil
}
