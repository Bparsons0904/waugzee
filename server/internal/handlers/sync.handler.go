package handlers

import (
	"waugzee/internal/app"
	syncController "waugzee/internal/controllers/sync"
	"waugzee/internal/handlers/middleware"
	"waugzee/internal/logger"
	"waugzee/internal/services"

	"github.com/gofiber/fiber/v2"
)

type SyncHandler struct {
	Handler
	syncController syncController.SyncControllerInterface
	zitadelService *services.ZitadelService
}

func NewSyncHandler(app app.App, router fiber.Router) *SyncHandler {
	log := logger.New("handlers").File("sync_handler")
	return &SyncHandler{
		syncController: app.Controllers.Sync,
		zitadelService: app.Services.Zitadel,
		Handler: Handler{
			log:        log,
			router:     router,
			middleware: app.Middleware,
		},
	}
}

func (h *SyncHandler) Register() {
	sync := h.router.Group("/sync")

	sync.Post("/syncCollection", h.InitiateCollectionSync)
}

func (h *SyncHandler) InitiateCollectionSync(c *fiber.Ctx) error {
	log := h.log.Function("InitiateCollectionSync")

	user := middleware.GetUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	err := h.syncController.HandleSyncRequest(c.Context(), user)
	if err != nil {
		_ = log.Error("Failed to handle sync request", "error", err, "userID", user.ID)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to initiate sync",
		})
	}

	log.Info("Collection sync initiated successfully", "userID", user.ID)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": "Collection sync initiated successfully",
	})
}
