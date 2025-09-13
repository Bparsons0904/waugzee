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
	config config.Config,
) *UserController {
	return &UserController{
		userRepo: userRepo,
		Config:   config,
		log:      logger.New("userController"),
	}
}
