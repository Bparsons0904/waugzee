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

	user.DiscogsToken = &req.Token
	user.DiscogsUsername = &identity.Username

	if err := uc.userRepo.Update(ctx, user); err != nil {
		return nil, log.Err("failed to update user with discogs credentials", err)
	}

	log.Info(
		"Discogs credentials updated successfully",
		"userID",
		user.ID,
		"username",
		identity.Username,
	)

	return user, nil
}
