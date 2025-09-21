package controllers

import (
	"waugzee/config"
	"waugzee/internal/events"
	"waugzee/internal/repositories"
	"waugzee/internal/services"

	authController "waugzee/internal/controllers/auth"
	syncController "waugzee/internal/controllers/sync"
	userController "waugzee/internal/controllers/users"
)

type Controllers struct {
	User userController.UserControllerInterface
	Auth authController.AuthControllerInterface
	Sync syncController.SyncControllerInterface
}

func New(
	services services.Service,
	repos repositories.Repository,
	eventBus *events.EventBus,
	config config.Config,
) Controllers {
	return Controllers{
		User: userController.New(repos, services, config),
		Auth: authController.New(services, repos),
		Sync: syncController.New(repos, services, eventBus, config),
	}
}
