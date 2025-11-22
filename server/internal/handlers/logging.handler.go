package handlers

import (
	"waugzee/internal/app"
	"waugzee/internal/handlers/middleware"
	"waugzee/internal/types"

	loggingController "waugzee/internal/controllers/logging"

	logger "github.com/Bparsons0904/goLogger"

	"github.com/gofiber/fiber/v2"
)

type LoggingHandler struct {
	Handler
	loggingController loggingController.LoggingControllerInterface
}

func NewLoggingHandler(app app.App, router fiber.Router) *LoggingHandler {
	log := logger.New("handlers").File("logging_handler")
	return &LoggingHandler{
		loggingController: app.Controllers.Logging,
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
	log := logger.New("handlers").TraceFromContext(c.Context()).File("logging_handler").Function("handleLogBatch")

	user := middleware.GetUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Authentication required"})
	}

	var req types.LogBatchRequest
	if err := c.BodyParser(&req); err != nil {
		log.Er("Failed to parse log batch request", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	response, err := h.loggingController.ProcessLogBatch(c.Context(), req, user.ID.String())
	if err != nil {
		log.Er("Failed to process log batch", err,
			"userID", user.ID,
			"sessionID", req.SessionID,
			"logCount", len(req.Logs))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to process logs"})
	}

	return c.Status(fiber.StatusOK).JSON(response)
}
