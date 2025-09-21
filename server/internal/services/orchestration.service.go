package services

import (
	"context"
	"fmt"
	"waugzee/internal/logger"

	"github.com/google/uuid"
)

// OrchestrationService coordinates API requests between the server and client
// following the client-as-proxy pattern for Discogs integration
type OrchestrationService struct {
	log logger.Logger
}

// NewOrchestrationService creates a new orchestration service instance
func NewOrchestrationService() *OrchestrationService {
	log := logger.New("OrchestrationService")
	return &OrchestrationService{
		log: log,
	}
}

// GetUserFolders initiates a request to get user folders from Discogs
// This method coordinates the API request through the client-as-proxy pattern
func (o *OrchestrationService) GetUserFolders(ctx context.Context, userID uuid.UUID, discogsToken string) (string, error) {
	log := o.log.Function("GetUserFolders")

	if userID == uuid.Nil {
		return "", fmt.Errorf("userID cannot be nil")
	}

	if discogsToken == "" {
		return "", fmt.Errorf("discogs token cannot be empty")
	}

	// Generate a unique request ID for tracking
	requestID := uuid.New().String()

	log.Info("Initiating user folders request",
		"userID", userID,
		"requestID", requestID)

	// TODO: Send API request message to client via WebSocket
	// This will be implemented when the WebSocket integration is wired up
	log.Info("User folders request initiated",
		"userID", userID,
		"requestID", requestID)

	return requestID, nil
}