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
	userRepo       repositories.UserRepository
	userConfigRepo repositories.UserConfigurationRepository
	folderRepo     repositories.FolderRepository
	discogsService *services.DiscogsService
	db             database.DB
	Config         config.Config
	log            logger.Logger
}

type UserControllerInterface interface {
	UpdateDiscogsToken(
		ctx context.Context,
		user *User,
		token string,
	) (*User, error)
	GetUser(ctx context.Context, userID uuid.UUID) ([]*Folder, error)
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
		userRepo:       repos.User,
		userConfigRepo: repos.UserConfiguration,
		folderRepo:     repos.Folder,
		discogsService: services.Discogs,
		db:             db,
		Config:         config,
		log:            logger.New("userController"),
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
	userID uuid.UUID,
) ([]*Folder, error) {
	return uc.folderRepo.GetUserFolders(ctx, uc.db.SQL, userID)
}

func (uc *UserController) UpdateSelectedFolder(
	ctx context.Context,
	user *User,
	folderID int,
) (*User, error) {
	log := uc.log.Function("UpdateSelectedFolder")

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
