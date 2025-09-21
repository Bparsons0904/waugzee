package controllers

import (
	"waugzee/config"
	"waugzee/internal/repositories"
	"waugzee/internal/services"

	authController "waugzee/internal/controllers/auth"
	userController "waugzee/internal/controllers/users"
)

type Controllers struct {
	User *userController.UserController
	Auth authController.AuthControllerInterface
}

func New(
	services services.Service,
	repos repositories.Repository,
	config config.Config,
) Controllers {
	return Controllers{
		User: userController.New(repos, services, config),
		Auth: authController.New(services, repos),
	}
}
