package app

import (
	"context"
	"waugzee/config"
	"waugzee/internal/database"
	"waugzee/internal/events"
	"waugzee/internal/handlers/middleware"
	"waugzee/internal/jobs"
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
	TransactionService   *services.TransactionService
	ZitadelService       *services.ZitadelService
	DiscogsService       *services.DiscogsService
	DownloadService      *services.DownloadService
	SchedulerService     *services.SchedulerService
	XMLProcessingService *services.XMLProcessingService

	// Repositories
	UserRepo                  repositories.UserRepository
	DiscogsDataProcessingRepo repositories.DiscogsDataProcessingRepository
	LabelRepo                 repositories.LabelRepository

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
	// Initialize repositories
	userRepo := repositories.New(db)
	discogsDataProcessingRepo := repositories.NewDiscogsDataProcessingRepository(db)
	labelRepo := repositories.NewLabelRepository(db)

	// Initialize services
	discogsService := services.NewDiscogsService()
	downloadService := services.NewDownloadService(config)
	schedulerService := services.NewSchedulerService()
	xmlProcessingService := services.NewXMLProcessingService(
		labelRepo,
		discogsDataProcessingRepo,
		transactionService,
	)

	websocket, err := websockets.New(db, eventBus, config, zitadelService, userRepo)
	if err != nil {
		return &App{}, log.Err("failed to create websocket manager", err)
	}

	// Initialize controllers with repositories and services
	middleware := middleware.New(db, eventBus, config, userRepo)
	authController := authController.New(zitadelService, userRepo, db)
	userController := userController.New(userRepo, discogsService, config)

	// Register jobs with scheduler if enabled
	if config.SchedulerEnabled {
		// Download job runs at 2:00 AM UTC daily
		discogsDownloadJob := jobs.NewDiscogsDownloadJob(
			discogsDataProcessingRepo,
			transactionService,
			downloadService,
			services.Daily,
		)
		if err := schedulerService.AddJob(discogsDownloadJob); err != nil {
			return &App{}, log.Err("failed to register Discogs download job", err)
		}
		log.Info("Registered Discogs download job with scheduler")

		// Processing job runs at 3:00 AM UTC daily (1 hour after download)
		discogsProcessingJob := jobs.NewDiscogsProcessingJob(
			discogsDataProcessingRepo,
			transactionService,
			xmlProcessingService,
			services.DailyProcessing,
		)
		if err := schedulerService.AddJob(discogsProcessingJob); err != nil {
			return &App{}, log.Err("failed to register Discogs processing job", err)
		}
		log.Info("Registered Discogs processing job with scheduler")
	}

	app := &App{
		Database:                  db,
		Config:                    config,
		Middleware:                middleware,
		TransactionService:        transactionService,
		ZitadelService:            zitadelService,
		DiscogsService:            discogsService,
		DownloadService:           downloadService,
		SchedulerService:          schedulerService,
		XMLProcessingService:      xmlProcessingService,
		UserRepo:                  userRepo,
		DiscogsDataProcessingRepo: discogsDataProcessingRepo,
		LabelRepo:                 labelRepo,
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
		a.DownloadService,
		a.SchedulerService,
		a.XMLProcessingService,
		a.AuthController,
		a.UserController,
		a.Middleware,
		a.UserRepo,
		a.DiscogsDataProcessingRepo,
		a.LabelRepo,
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

	if a.SchedulerService != nil {
		if closeErr := a.SchedulerService.Stop(context.Background()); closeErr != nil {
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
