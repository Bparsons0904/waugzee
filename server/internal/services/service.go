package services

import (
	"waugzee/config"
	"waugzee/internal/database"
	"waugzee/internal/events"
	"waugzee/internal/repositories"
)

type Service struct {
	Zitadel              *ZitadelService
	Discogs              *DiscogsService
	Transaction          *TransactionService
	Scheduler            *SchedulerService
	Orchestration        *OrchestrationService
	FolderDataExtraction *FolderDataExtractionService
}

func New(db database.DB, config config.Config, eventBus *events.EventBus) (Service, error) {
	transactionService := NewTransactionService(db)
	repos := repositories.New(db)

	zitadelService, err := NewZitadelService(config)
	if err != nil {
		return Service{}, err
	}

	discogsService := NewDiscogsService()
	schedulerService := NewSchedulerService()
	orchestrationService := NewOrchestrationService(eventBus, repos, db, transactionService)
	folderDataExtractionService := NewFolderDataExtractionService(repos)

	return Service{
		Zitadel:              zitadelService,
		Discogs:              discogsService,
		Transaction:          transactionService,
		Scheduler:            schedulerService,
		Orchestration:        orchestrationService,
		FolderDataExtraction: folderDataExtractionService,
	}, nil
}
