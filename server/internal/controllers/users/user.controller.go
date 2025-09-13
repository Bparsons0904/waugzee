package userController

import (
	"waugzee/config"
	"waugzee/internal/logger"
	"waugzee/internal/repositories"
)

type UserController struct {
	userRepo repositories.UserRepository
	Config   config.Config
	log      logger.Logger
}

func New(
	userRepo repositories.UserRepository,
	discogsService *services.DiscogsService,
	config config.Config,
) *UserController {
	return &UserController{
		userRepo:       userRepo,
		discogsService: discogsService,
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

	// Validate token with Discogs API
	if err := uc.discogsService.ValidateToken(req.Token); err != nil {
		log.Warn("Invalid Discogs token provided", "userID", user.ID, "error", err)
		return nil, log.Err("invalid discogs token", err)
	}

	user.DiscogsToken = &req.Token
	if err := uc.userRepo.Update(ctx, user); err != nil {
		return nil, log.Err("failed to update user with discogs token", err)
	}

	return user, nil
}
