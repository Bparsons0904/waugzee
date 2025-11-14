package controllers

import (
	"waugzee/config"
	"waugzee/internal/database"
	"waugzee/internal/events"
	"waugzee/internal/repositories"
	"waugzee/internal/services"

	adminController "waugzee/internal/controllers/admin"
	authController "waugzee/internal/controllers/auth"
	historyController "waugzee/internal/controllers/history"
	recommendationController "waugzee/internal/controllers/recommendation"
	stylusController "waugzee/internal/controllers/stylus"
	syncController "waugzee/internal/controllers/sync"
	userController "waugzee/internal/controllers/users"
)

type Controllers struct {
	User           userController.UserControllerInterface
	Auth           authController.AuthControllerInterface
	Sync           syncController.SyncControllerInterface
	Stylus         stylusController.StylusControllerInterface
	History        historyController.HistoryControllerInterface
	Admin          adminController.AdminControllerInterface
	Recommendation recommendationController.RecommendationControllerInterface
}

func New(
	services services.Service,
	repos repositories.Repository,
	eventBus *events.EventBus,
	config config.Config,
	db database.DB,
) Controllers {
	return Controllers{
		User:    userController.New(repos, services, config, db),
		Auth:    authController.New(services, repos, db),
		Sync:    syncController.New(repos, services, eventBus, config),
		Stylus:  stylusController.New(repos, services, config, db),
		History: historyController.New(repos, services, config, db),
		Admin: adminController.NewAdminController(
			db.SQL,
			repos.DiscogsDataProcessing,
			services.Download,
			services.DiscogsXMLParser,
			services.Scheduler,
			services.FileCleanup,
			services.KleioImport, // TODO: REMOVE_AFTER_MIGRATION
		),
		Recommendation: recommendationController.New(repos, db.SQL, db.Cache.ClientAPI),
	}
}
