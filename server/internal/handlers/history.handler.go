package handlers

import (
	"errors"
	"waugzee/internal/app"
	historyController "waugzee/internal/controllers/history"
	"waugzee/internal/handlers/middleware"
	logger "github.com/Bparsons0904/goLogger"

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
	plays.Put("/:id", h.updatePlayHistory)
	plays.Delete("/:id", h.deletePlayHistory)

	cleanings := h.router.Group("/cleanings")
	cleanings.Post("", h.logCleaning)
	cleanings.Put("/:id", h.updateCleaningHistory)
	cleanings.Delete("/:id", h.deleteCleaningHistory)

	h.router.Post("/logBoth", h.logBoth)
}

func (h *HistoryHandler) logPlay(c *fiber.Ctx) error {
	log := logger.New("handlers").TraceFromContext(c.UserContext()).File("history_handler").Function("logPlay")

	user := middleware.GetUser(c)
	if user == nil {
		log.Warn("Unauthorized access attempt")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	var req historyController.LogPlayRequest
	if err := c.BodyParser(&req); err != nil {
		log.Warn("Invalid request body", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	playHistory, err := h.historyController.LogPlay(c.Context(), user, &req)
	if err != nil {
		if errors.Is(err, historyController.ErrValidation) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		if errors.Is(err, historyController.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		_ = log.Err("Failed to log play", err, "userID", user.ID)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to log play",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"playHistory": playHistory,
	})
}

func (h *HistoryHandler) updatePlayHistory(c *fiber.Ctx) error {
	log := logger.New("handlers").TraceFromContext(c.UserContext()).File("history_handler").Function("updatePlayHistory")

	user := middleware.GetUser(c)

	playHistoryIDParam := c.Params("id")
	playHistoryID, err := uuid.Parse(playHistoryIDParam)
	if err != nil {
		log.Warn("Invalid play history ID", "id", playHistoryIDParam)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid play history ID",
		})
	}

	var req historyController.UpdatePlayHistoryRequest
	if err = c.BodyParser(&req); err != nil {
		log.Warn("Invalid request body", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	playHistory, err := h.historyController.UpdatePlayHistory(
		c.Context(),
		user,
		playHistoryID,
		&req,
	)
	if err != nil {
		if errors.Is(err, historyController.ErrValidation) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		if errors.Is(err, historyController.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		_ = log.Err("Failed to update play history", err, "playHistoryID", playHistoryID)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update play history",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"playHistory": playHistory,
	})
}

func (h *HistoryHandler) deletePlayHistory(c *fiber.Ctx) error {
	log := logger.New("handlers").TraceFromContext(c.UserContext()).File("history_handler").Function("deletePlayHistory")

	user := middleware.GetUser(c)
	if user == nil {
		log.Warn("Unauthorized access attempt")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	playHistoryIDParam := c.Params("id")
	playHistoryID, err := uuid.Parse(playHistoryIDParam)
	if err != nil {
		log.Warn("Invalid play history ID", "id", playHistoryIDParam)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid play history ID",
		})
	}

	if err := h.historyController.DeletePlayHistory(c.Context(), user, playHistoryID); err != nil {
		if errors.Is(err, historyController.ErrValidation) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		if errors.Is(err, historyController.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		_ = log.Err("Failed to delete play history", err, "playHistoryID", playHistoryID)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete play history",
		})
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

func (h *HistoryHandler) logCleaning(c *fiber.Ctx) error {
	log := logger.New("handlers").TraceFromContext(c.UserContext()).File("history_handler").Function("logCleaning")

	user := middleware.GetUser(c)
	if user == nil {
		log.Warn("Unauthorized access attempt")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	var req historyController.LogCleaningRequest
	if err := c.BodyParser(&req); err != nil {
		log.Warn("Invalid request body", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	cleaningHistory, err := h.historyController.LogCleaning(c.Context(), user, &req)
	if err != nil {
		if errors.Is(err, historyController.ErrValidation) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		if errors.Is(err, historyController.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		_ = log.Err("Failed to log cleaning", err, "userID", user.ID)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to log cleaning",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"cleaningHistory": cleaningHistory,
	})
}

func (h *HistoryHandler) updateCleaningHistory(c *fiber.Ctx) error {
	log := logger.New("handlers").TraceFromContext(c.UserContext()).File("history_handler").Function("updateCleaningHistory")

	user := middleware.GetUser(c)
	if user == nil {
		log.Warn("Unauthorized access attempt")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	cleaningHistoryIDParam := c.Params("id")
	cleaningHistoryID, err := uuid.Parse(cleaningHistoryIDParam)
	if err != nil {
		log.Warn("Invalid cleaning history ID", "id", cleaningHistoryIDParam)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid cleaning history ID",
		})
	}

	var req historyController.UpdateCleaningHistoryRequest
	if err = c.BodyParser(&req); err != nil {
		log.Warn("Invalid request body", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	cleaningHistory, err := h.historyController.UpdateCleaningHistory(
		c.Context(),
		user,
		cleaningHistoryID,
		&req,
	)
	if err != nil {
		if errors.Is(err, historyController.ErrValidation) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		if errors.Is(err, historyController.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		_ = log.Err("Failed to update cleaning history", err, "cleaningHistoryID", cleaningHistoryID)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update cleaning history",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"cleaningHistory": cleaningHistory,
	})
}

func (h *HistoryHandler) deleteCleaningHistory(c *fiber.Ctx) error {
	log := logger.New("handlers").TraceFromContext(c.UserContext()).File("history_handler").Function("deleteCleaningHistory")

	user := middleware.GetUser(c)
	if user == nil {
		log.Warn("Unauthorized access attempt")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	cleaningHistoryIDParam := c.Params("id")
	cleaningHistoryID, err := uuid.Parse(cleaningHistoryIDParam)
	if err != nil {
		log.Warn("Invalid cleaning history ID", "id", cleaningHistoryIDParam)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid cleaning history ID",
		})
	}

	if err := h.historyController.DeleteCleaningHistory(c.Context(), user, cleaningHistoryID); err != nil {
		if errors.Is(err, historyController.ErrValidation) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		if errors.Is(err, historyController.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		_ = log.Err("Failed to delete cleaning history", err, "cleaningHistoryID", cleaningHistoryID)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete cleaning history",
		})
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

func (h *HistoryHandler) logBoth(c *fiber.Ctx) error {
	log := logger.New("handlers").TraceFromContext(c.UserContext()).File("history_handler").Function("logBoth")

	user := middleware.GetUser(c)
	if user == nil {
		log.Warn("Unauthorized access attempt")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	var req historyController.LogBothRequest
	if err := c.BodyParser(&req); err != nil {
		log.Warn("Invalid request body", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	response, err := h.historyController.LogBoth(c.Context(), user, &req)
	if err != nil {
		if errors.Is(err, historyController.ErrValidation) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		if errors.Is(err, historyController.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		_ = log.Err("Failed to log play and cleaning", err, "userID", user.ID)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to log play and cleaning",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(response)
}
