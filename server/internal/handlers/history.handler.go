package handlers

import (
	"strings"
	"waugzee/internal/app"
	historyController "waugzee/internal/controllers/history"
	"waugzee/internal/handlers/middleware"
	"waugzee/internal/logger"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type HistoryHandler struct {
	Handler
	historyController historyController.HistoryControllerInterface
}

func NewHistoryHandler(app app.App, router fiber.Router) *HistoryHandler {
	log := logger.New("handlers").File("history_handler")
	return &HistoryHandler{
		historyController: app.Controllers.History,
		Handler: Handler{
			log:        log,
			router:     router,
			middleware: app.Middleware,
		},
	}
}

func (h *HistoryHandler) Register() {
	plays := h.router.Group("/plays")
	plays.Post("", h.logPlay)
	plays.Delete("/:id", h.deletePlayHistory)

	cleanings := h.router.Group("/cleanings")
	cleanings.Post("", h.logCleaning)
	cleanings.Delete("/:id", h.deleteCleaningHistory)
}

func (h *HistoryHandler) logPlay(c *fiber.Ctx) error {
	user := middleware.GetUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	var req historyController.LogPlayRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	playHistory, err := h.historyController.LogPlay(c.Context(), user, &req)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "required") ||
			strings.Contains(errMsg, "not found") ||
			strings.Contains(errMsg, "invalid") ||
			strings.Contains(errMsg, "cannot be") ||
			strings.Contains(errMsg, "exceed") {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": errMsg,
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to log play",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"playHistory": playHistory,
	})
}

func (h *HistoryHandler) deletePlayHistory(c *fiber.Ctx) error {
	user := middleware.GetUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	playHistoryIDParam := c.Params("id")
	playHistoryID, err := uuid.Parse(playHistoryIDParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid play history ID",
		})
	}

	if err := h.historyController.DeletePlayHistory(c.Context(), user, playHistoryID); err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "required") {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": errMsg,
			})
		}
		if strings.Contains(errMsg, "not found") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": errMsg,
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete play history",
		})
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

func (h *HistoryHandler) logCleaning(c *fiber.Ctx) error {
	user := middleware.GetUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	var req historyController.LogCleaningRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	cleaningHistory, err := h.historyController.LogCleaning(c.Context(), user, &req)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "required") ||
			strings.Contains(errMsg, "invalid") ||
			strings.Contains(errMsg, "cannot be") ||
			strings.Contains(errMsg, "exceed") {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": errMsg,
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to log cleaning",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"cleaningHistory": cleaningHistory,
	})
}

func (h *HistoryHandler) deleteCleaningHistory(c *fiber.Ctx) error {
	user := middleware.GetUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	cleaningHistoryIDParam := c.Params("id")
	cleaningHistoryID, err := uuid.Parse(cleaningHistoryIDParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid cleaning history ID",
		})
	}

	if err := h.historyController.DeleteCleaningHistory(c.Context(), user, cleaningHistoryID); err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "required") {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": errMsg,
			})
		}
		if strings.Contains(errMsg, "not found") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": errMsg,
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete cleaning history",
		})
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}


