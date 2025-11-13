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
	DiscogsRateLimiter   *DiscogsRateLimiterService
	Download             *DownloadService
	DiscogsXMLParser     *DiscogsXMLParserService
	ReleaseSync          *ReleaseSyncService
	FileCleanup          *FileCleanupService
	KleioImport          *KleioImportService       // TODO: REMOVE_AFTER_MIGRATION
	CacheInvalidation    *CacheInvalidationService
	Recommendation       *RecommendationService
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
	discogsRateLimiterService := NewDiscogsRateLimiterService(db.Cache.ClientAPI)
	recommendationService := NewRecommendationService(repos, db.SQL, db.Cache.ClientAPI)
	orchestrationService := NewOrchestrationService(
		eventBus,
		repos,
		db,
		transactionService,
		discogsRateLimiterService,
		recommendationService,
	)
	folderDataExtractionService := NewFolderDataExtractionService(repos)
	downloadService := NewDownloadService(config, eventBus)
	discogsXMLParserService := NewDiscogsXMLParserService(repos, db, eventBus)
	releaseSyncService := NewReleaseSyncService(eventBus, repos, db, discogsRateLimiterService)
	fileCleanupService := NewFileCleanupService(config)
	cacheInvalidationService := NewCacheInvalidationService(eventBus)
	// TODO: REMOVE_AFTER_MIGRATION - One-time Kleio data import service
	kleioImportService := NewKleioImportService(
		db,
		repos.Stylus,
		repos.UserRelease,
		repos.History,
		transactionService,
	)

	return Service{
		Zitadel:              zitadelService,
		Discogs:              discogsService,
		Transaction:          transactionService,
		Scheduler:            schedulerService,
		Orchestration:        orchestrationService,
		FolderDataExtraction: folderDataExtractionService,
		DiscogsRateLimiter:   discogsRateLimiterService,
		Download:             downloadService,
		DiscogsXMLParser:     discogsXMLParserService,
		ReleaseSync:          releaseSyncService,
		FileCleanup:          fileCleanupService,
		CacheInvalidation:    cacheInvalidationService,
		KleioImport:          kleioImportService, // TODO: REMOVE_AFTER_MIGRATION
		Recommendation:       recommendationService,
	}, nil
}
