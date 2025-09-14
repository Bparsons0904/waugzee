package app

import (
	"waugzee/config"
	"waugzee/internal/database"
	"waugzee/internal/events"
	"waugzee/internal/handlers/middleware"
	"waugzee/internal/logger"
	"waugzee/internal/repositories"
	"waugzee/internal/services"
	"waugzee/internal/websockets"

	authController "waugzee/internal/controllers/auth"
	userController "waugzee/internal/controllers/users"
)

type App struct {
	Database   database.DB
	Middleware middleware.Middleware
	Websocket  *websockets.Manager
	EventBus   *events.EventBus
	Config     config.Config

	// Services
	TransactionService *services.TransactionService
	ZitadelService     *services.ZitadelService
	DiscogsService     *services.DiscogsService

	// Repositories
	UserRepo                  repositories.UserRepository
	DiscogsDataProcessingRepo repositories.DiscogsDataProcessingRepository

	// Controllers
	AuthController authController.AuthControllerInterface
	UserController *userController.UserController
}

func New() (*App, error) {
	log := logger.New("app").Function("New")

	config, err := config.InitConfig()
	if err != nil {
		return &App{}, log.Err("failed to initialize config", err)
	}

	db, err := database.New(config)
	if err != nil {
		return &App{}, log.Err("failed to create database", err)
	}

	eventBus := events.New(db.Cache.Events, config)

	// Initialize services
	transactionService := services.NewTransactionService(db)
	zitadelService, err := services.NewZitadelService(config)
	if err != nil {
		return &App{}, log.Err("failed to create Zitadel service", err)
	}
	discogsService := services.NewDiscogsService()

	// Initialize repositories
	userRepo := repositories.New(db)
	discogsDataProcessingRepo := repositories.NewDiscogsDataProcessingRepository(db)

	websocket, err := websockets.New(db, eventBus, config, zitadelService, userRepo)
	if err != nil {
		return &App{}, log.Err("failed to create websocket manager", err)
	}

	// Initialize controllers with repositories and services
	middleware := middleware.New(db, eventBus, config, userRepo)
	authController := authController.New(zitadelService, userRepo, db)
	userController := userController.New(userRepo, discogsService, config)

	app := &App{
		Database:                  db,
		Config:                    config,
		Middleware:                middleware,
		TransactionService:        transactionService,
		ZitadelService:            zitadelService,
		DiscogsService:            discogsService,
		UserRepo:                  userRepo,
		DiscogsDataProcessingRepo: discogsDataProcessingRepo,
		AuthController:            authController,
		UserController:            userController,
		Websocket:                 websocket,
		EventBus:                  eventBus,
	}

	if err := app.validate(); err != nil {
		return &App{}, log.Err("failed to validate app", err)
	}

	return app, nil
}

func (a *App) validate() error {
	log := logger.New("app").Function("validate")
	if a.Database.SQL == nil {
		return log.ErrMsg("database is nil")
	}

	if a.Config == (config.Config{}) {
		return log.ErrMsg("config is nil")
	}

	nilChecks := []any{
		a.Websocket,
		a.EventBus,
		a.TransactionService,
		a.ZitadelService,
		a.DiscogsService,
		a.AuthController,
		a.UserController,
		a.Middleware,
		a.UserRepo,
		a.DiscogsDataProcessingRepo,
	}

	for _, check := range nilChecks {
		if check == nil {
			return log.ErrMsg("nil check failed")
		}
	}

	return nil
}

func (a *App) Close() (err error) {
	if a.EventBus != nil {
		if closeErr := a.EventBus.Close(); closeErr != nil {
			err = closeErr
		}
	}

	if a.ZitadelService != nil {
		if closeErr := a.ZitadelService.Close(); closeErr != nil {
			err = closeErr
		}
	}

	if dbErr := a.Database.Close(); dbErr != nil {
		err = dbErr
	}

	return err
}

