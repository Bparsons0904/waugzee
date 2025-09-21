package syncController

import (
	"context"
	"time"
	"waugzee/config"
	"waugzee/internal/events"
	"waugzee/internal/logger"
	. "waugzee/internal/models"
	"waugzee/internal/repositories"
	"waugzee/internal/services"

	"github.com/google/uuid"
)

type SyncController struct {
	userRepo       repositories.UserRepository
	discogsService *services.DiscogsService
	eventBus       *events.EventBus
	config         config.Config
	log            logger.Logger
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
		userRepo:       repos.User,
		discogsService: services.Discogs,
		eventBus:       eventBus,
		config:         config,
		log:            logger.New("syncController"),
	}
}

func (sc *SyncController) HandleSyncRequest(
	ctx context.Context,
	user *User,
) error {
	log := sc.log.Function("HandleSyncRequest")

	if user == nil {
		return log.ErrMsg("user is required")
	}

	log.Info("Processing sync request", "userID", user.ID)

	channelEvent := events.ChannelEvent{
		Event: "api",
		Message: events.Message{
			ID:        uuid.New().String(),
			Service:   events.API,
			Event:     "api",
			UserID:    user.ID.String(),
			Payload:   map[string]any{},
			Timestamp: time.Now(),
		},
	}

	if err := sc.eventBus.Publish(events.WEBSOCKET, channelEvent); err != nil {
		return log.Err("failed to publish sync event", err)
	}

	return nil
}
