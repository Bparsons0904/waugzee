package userController

import (
	"context"
	"waugzee/config"
	"waugzee/internal/logger"
	. "waugzee/internal/models"
	"waugzee/internal/repositories"
	"waugzee/internal/services"
)

type UserController struct {
	userRepo       repositories.UserRepository
	userConfigRepo repositories.UserConfigurationRepository
	discogsService *services.DiscogsService
	Config         config.Config
	log            logger.Logger
}

type UserControllerInterface interface {
	UpdateDiscogsToken(ctx context.Context, user *User, req UpdateDiscogsTokenRequest) (*User, error)
}

func New(
	repos repositories.Repository,
	services services.Service,
	config config.Config,
) UserControllerInterface {
	return &UserController{
		userRepo:       repos.User,
		userConfigRepo: repos.UserConfiguration,
		discogsService: services.Discogs,
		Config:         config,
		log:            logger.New("userController"),
	}
}

type UpdateDiscogsTokenRequest struct {
	Token string `json:"token"`
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

	if err := uc.userConfigRepo.CreateOrUpdate(ctx, config); err != nil {
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
