package app

import (
	"context"
	"waugzee/config"
	"waugzee/internal/controllers"
	"waugzee/internal/database"
	"waugzee/internal/events"
	"waugzee/internal/handlers/middleware"
	"waugzee/internal/jobs"
	"waugzee/internal/logger"
	"waugzee/internal/repositories"
	"waugzee/internal/services"
	"waugzee/internal/websockets"
)

type App struct {
	Database   database.DB
	Middleware middleware.Middleware
	Websocket  *websockets.Manager
	EventBus   *events.EventBus
	Config     config.Config

	Services    services.Service
	Repos       repositories.Repository
	Controllers controllers.Controllers
}

func New() (*App, error) {
	log := logger.New("app").Function("New")

	config, err := config.New()
	if err != nil {
		return &App{}, log.Err("failed to initialize config", err)
	}

	db, err := database.New(config)
	if err != nil {
		return &App{}, log.Err("failed to create database", err)
	}

	eventBus := events.New(db.Cache.Events, config)
	repos := repositories.New(db)

	servicesComposite, err := services.New(db, config)
	if err != nil {
		return &App{}, log.Err("failed to initialize services", err)
	}

	websocket, err := websockets.New(
		db,
		eventBus,
		config,
		servicesComposite,
		repos,
	)
	if err != nil {
		return &App{}, log.Err("failed to create websocket manager", err)
	}

	middleware := middleware.New(db, eventBus, config, repos)
	controllersComposite := controllers.New(servicesComposite, repos, config)

	if err := jobs.RegisterAllJobs(servicesComposite.Scheduler, config); err != nil {
		return &App{}, log.Err("failed to register jobs", err)
	}

	app := &App{
		Database:    db,
		Config:      config,
		Middleware:  middleware,
		Services:    servicesComposite,
		Repos:       repos,
		Controllers: controllersComposite,
		Websocket:   websocket,
		EventBus:    eventBus,
	}

	return app, nil
}

func (a *App) Close() (err error) {
	if a.EventBus != nil {
		if closeErr := a.EventBus.Close(); closeErr != nil {
			err = closeErr
		}
	}

	if a.Services.Scheduler != nil {
		if closeErr := a.Services.Scheduler.Stop(context.Background()); closeErr != nil {
			err = closeErr
		}
	}

	if a.Services.Zitadel != nil {
		if closeErr := a.Services.Zitadel.Close(); closeErr != nil {
			err = closeErr
		}
	}

	if dbErr := a.Database.Close(); dbErr != nil {
		err = dbErr
	}

	return err
}
