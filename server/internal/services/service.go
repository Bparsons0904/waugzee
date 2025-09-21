package services

import (
	"waugzee/config"
	"waugzee/internal/database"
)

type Service struct {
	Zitadel       *ZitadelService
	Discogs       *DiscogsService
	Transaction   *TransactionService
	Scheduler     *SchedulerService
	Orchestration *OrchestrationService
}

func New(db database.DB, config config.Config) (Service, error) {
	transactionService := NewTransactionService(db)

	zitadelService, err := NewZitadelService(config)
	if err != nil {
		return Service{}, err
	}

	discogsService := NewDiscogsService()
	schedulerService := NewSchedulerService()
	orchestrationService := NewOrchestrationService()

	return Service{
		Zitadel:       zitadelService,
		Discogs:       discogsService,
		Transaction:   transactionService,
		Scheduler:     schedulerService,
		Orchestration: orchestrationService,
	}, nil
}

