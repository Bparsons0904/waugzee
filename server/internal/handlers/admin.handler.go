package handlers

import (
	"waugzee/internal/app"
	"waugzee/internal/logger"

	"github.com/gofiber/fiber/v2"
)

type AdminHandler struct {
	Handler
}

func NewAdminHandler(app app.App, router fiber.Router) *AdminHandler {
	log := logger.New("handlers").File("admin_handler")
	return &AdminHandler{
		Handler: Handler{
			log:        log,
			router:     router,
			middleware: app.Middleware,
		},
	}
}

func (h *AdminHandler) Register() {
	admin := h.router.Group("/admin")

	// Monthly Downloads
	admin.Get("/downloads/status", h.getDownloadStatus)
}

// Monthly Downloads Handlers
func (h *AdminHandler) getDownloadStatus(c *fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "TODO: implement"})
}
