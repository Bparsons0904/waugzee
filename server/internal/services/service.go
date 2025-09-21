package services

import (
	"waugzee/config"
	"waugzee/internal/database"
	"waugzee/internal/events"
)

type Service struct {
	Zitadel       *ZitadelService
	Discogs       *DiscogsService
	Transaction   *TransactionService
	Scheduler     *SchedulerService
	Orchestration *OrchestrationService
}

func New(db database.DB, config config.Config, eventBus *events.EventBus) (Service, error) {
	transactionService := NewTransactionService(db)

	zitadelService, err := NewZitadelService(config)
	if err != nil {
		return Service{}, err
	}

	discogsService := NewDiscogsService()
	schedulerService := NewSchedulerService()
	orchestrationService := NewOrchestrationService(eventBus, db)

	return Service{
		Zitadel:       zitadelService,
		Discogs:       discogsService,
		Transaction:   transactionService,
		Scheduler:     schedulerService,
		Orchestration: orchestrationService,
	}, nil
}

