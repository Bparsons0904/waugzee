package services

import (
	"waugzee/internal/events"
	"waugzee/internal/logger"
	// . "waugzee/internal/models"
)

type CacheInvalidationService struct {
	eventBus *events.EventBus
	log      logger.Logger
}

func NewCacheInvalidationService(
	eventBus *events.EventBus,
) *CacheInvalidationService {
	return &CacheInvalidationService{
		eventBus: eventBus,
		log:      logger.New("CacheInvalidationService"),
	}
}

