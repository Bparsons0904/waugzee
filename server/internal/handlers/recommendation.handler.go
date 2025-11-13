package handlers

import (
	"errors"
	"waugzee/internal/app"
	recommendationController "waugzee/internal/controllers/recommendation"
	"waugzee/internal/handlers/middleware"
	"waugzee/internal/logger"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type RecommendationHandler struct {
	Handler
	recommendationController recommendationController.RecommendationControllerInterface
}

func NewRecommendationHandler(app app.App, router fiber.Router) *RecommendationHandler {
	log := logger.New("handlers").File("recommendation_handler")
	return &RecommendationHandler{
		recommendationController: app.Controllers.Recommendation,
		Handler: Handler{
			log:        log,
			router:     router,
			middleware: app.Middleware,
		},
	}
}

func (h *RecommendationHandler) Register() {
	recommendations := h.router.Group("/recommendations")
	recommendations.Post("/:id/listen", h.markAsListened)
}

func (h *RecommendationHandler) markAsListened(c *fiber.Ctx) error {
	user := middleware.GetUser(c)

	recommendationIDParam := c.Params("id")
	recommendationID, err := uuid.Parse(recommendationIDParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid recommendation ID",
		})
	}

	err = h.recommendationController.MarkAsListened(
		c.Context(),
		user,
		recommendationID,
	)
	if err != nil {
		if errors.Is(err, recommendationController.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Recommendation not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to mark recommendation as listened",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Recommendation marked as listened",
	})
}
