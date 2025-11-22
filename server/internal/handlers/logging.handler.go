package handlers

import (
	"waugzee/internal/app"
	"waugzee/internal/handlers/middleware"
	"waugzee/internal/logger"
	"waugzee/internal/services"
	"waugzee/internal/types"

	"github.com/gofiber/fiber/v2"
)

type LoggingHandler struct {
	Handler
	loggingService *services.LoggingService
}

func NewLoggingHandler(app app.App, router fiber.Router) *LoggingHandler {
	log := logger.New("handlers").File("logging_handler")
	return &LoggingHandler{
		loggingService: app.Services.Logging,
		Handler: Handler{
			log:        log,
			router:     router,
			middleware: app.Middleware,
		},
	}
}

func (h *LoggingHandler) Register() {
	h.router.Post("/logs", h.handleLogBatch)
}

func (h *LoggingHandler) handleLogBatch(c *fiber.Ctx) error {
	log := h.log.Function("handleLogBatch")

	user := middleware.GetUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Authentication required"})
	}

	var req types.LogBatchRequest
	if err := c.BodyParser(&req); err != nil {
		log.Er("Failed to parse log batch request", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if len(req.Logs) == 0 {
		return c.Status(fiber.StatusOK).JSON(types.LogBatchResponse{
			Success:   true,
			Processed: 0,
		})
	}

	if req.SessionID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Session ID is required"})
	}

	// Process the log batch
	response, err := h.loggingService.ProcessLogBatch(c.Context(), req, user.ID.String())
	if err != nil {
		log.Er("Failed to process log batch", err,
			"userID", user.ID,
			"sessionID", req.SessionID,
			"logCount", len(req.Logs))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to process logs"})
	}

	return c.Status(fiber.StatusOK).JSON(response)
}
