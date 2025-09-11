package userController

import (
	"context"
	"waugzee/config"
	"waugzee/internal/events"
	"waugzee/internal/logger"
	. "waugzee/internal/models"
	"waugzee/internal/repositories"
)

type UserController struct {
	userRepo  repositories.UserRepository
	Config    config.Config
	log       logger.Logger
	wsManager WebSocketManager
	eventBus  *events.EventBus
}

type WebSocketManager interface {
	BroadcastUserLogin(userID string, userData map[string]any)
}

func New(
	eventBus *events.EventBus,
	userRepo repositories.UserRepository,
	config config.Config,
) *UserController {
	return &UserController{
		userRepo:  userRepo,
		Config:    config,
		log:       logger.New("userController"),
		wsManager: nil,
		eventBus:  eventBus,
	}
}

func (c *UserController) SetWebSocketManager(wsManager WebSocketManager) {
	c.wsManager = wsManager
}

func (c *UserController) Login(
	ctx context.Context,
	loginRequest LoginRequest,
) (user User, err error) {
	userPtr, err := c.userRepo.GetByLogin(ctx, loginRequest.Login)
	if err != nil {
		return user, err
	}
	user = *userPtr

	return user, err
}

// TODO: implement
func (c *UserController) Logout(sessionID string) (err error) {
	return err
}
