package loggingController

import (
	"context"

	logger "github.com/Bparsons0904/goLogger"

	"waugzee/internal/services"
	"waugzee/internal/types"
)

type LoggingController struct {
	loggingService *services.LoggingService
}

type LoggingControllerInterface interface {
	ProcessLogBatch(ctx context.Context, req types.LogBatchRequest, userID string) (*types.LogBatchResponse, error)
}

func New(services services.Service) LoggingControllerInterface {
	return &LoggingController{
		loggingService: services.Logging,
	}
}

// ProcessLogBatch validates and processes a batch of client logs
func (c *LoggingController) ProcessLogBatch(
	ctx context.Context,
	req types.LogBatchRequest,
	userID string,
) (*types.LogBatchResponse, error) {
	log := logger.New("loggingController").TraceFromContext(ctx).Function("ProcessLogBatch")

	// Handle empty batch
	if len(req.Logs) == 0 {
		return &types.LogBatchResponse{
			Success:   true,
			Processed: 0,
		}, nil
	}

	// Validate session ID
	if req.SessionID == "" {
		return nil, log.ErrMsg("session ID is required")
	}

	// Process the log batch via service
	response, err := c.loggingService.ProcessLogBatch(ctx, req, userID)
	if err != nil {
		log.Er("Failed to process log batch", err,
			"userID", userID,
			"sessionID", req.SessionID,
			"logCount", len(req.Logs))
		return nil, err
	}

	return response, nil
}
