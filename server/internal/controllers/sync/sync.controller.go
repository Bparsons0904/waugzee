package syncController

import (
	"context"
	"waugzee/config"
	"waugzee/internal/events"
	"waugzee/internal/logger"
	. "waugzee/internal/models"
	"waugzee/internal/repositories"
	"waugzee/internal/services"
)

type SyncController struct {
	userRepo             repositories.UserRepository
	discogsService       *services.DiscogsService
	orchestrationService *services.OrchestrationService
	eventBus             *events.EventBus
	config               config.Config
	log                  logger.Logger
}

type SyncControllerInterface interface {
	HandleSyncRequest(ctx context.Context, user *User) error
}

func New(
	repos repositories.Repository,
	services services.Service,
	eventBus *events.EventBus,
	config config.Config,
) SyncControllerInterface {
	return &SyncController{
		userRepo:             repos.User,
		discogsService:       services.Discogs,
		orchestrationService: services.Orchestration,
		eventBus:             eventBus,
		config:               config,
		log:                  logger.New("syncController"),
	}
}

func (sc *SyncController) HandleSyncRequest(
	ctx context.Context,
	user *User,
) error {
	log := sc.log.Function("HandleSyncRequest")

	if user.Configuration == nil || user.Configuration.DiscogsToken == nil || *user.Configuration.DiscogsToken == "" {
		return log.ErrMsg("user does not have a Discogs token configured")
	}

	requestID, err := sc.orchestrationService.GetUserFolders(ctx, user)
	if err != nil {
		return log.Err("failed to initiate user folders request", err)
	}

	log.Info("Sync request initiated successfully",
		"userID", user.ID,
		"requestID", requestID)

	return nil
}
