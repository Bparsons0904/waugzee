package handlers

import (
	"waugzee/internal/app"
	"waugzee/internal/handlers/middleware"
	"waugzee/internal/logger"
	"waugzee/internal/services" // Claude these all should be controllers and not services. Controllers handle the business logic.

	"github.com/gofiber/fiber/v2"
)

type AdminHandler struct {
	Handler
	adminService services.AdminServiceInterface
}

func NewAdminHandler(app app.App, router fiber.Router) *AdminHandler {
	log := logger.New("handlers").File("admin_handler")
	return &AdminHandler{
		adminService: app.Services.Admin,
		Handler: Handler{
			log:        log,
			router:     router,
			middleware: app.Middleware,
		},
	}
}

func (h *AdminHandler) Register() {
	admin := h.router.Group("/admin", h.middleware.RequireAdmin())

	admin.Get("/downloads/status", h.getDownloadStatus)
	admin.Post("/downloads/trigger", h.triggerDownload)
	admin.Post("/downloads/reprocess", h.triggerReprocess)
}

func (h *AdminHandler) getDownloadStatus(c *fiber.Ctx) error {
	log := h.log.Function("getDownloadStatus")

	status, err := h.adminService.GetDownloadStatus(c.Context())
	if err != nil {
		_ = log.Err("Failed to get download status", err)
		return c.Status(fiber.StatusInternalServerError).
			JSON(fiber.Map{"error": "Failed to get download status"})
	}

	// Claude is this really necessary?  Just return the status and we would get the same result correct?
	if status == nil {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{})
	}

	return c.Status(fiber.StatusOK).JSON(status)
}

func (h *AdminHandler) triggerDownload(c *fiber.Ctx) error {
	log := h.log.Function("triggerDownload")

	user := middleware.GetUser(c)

	log.Info("Admin triggering download", "userID", user.ID, "email", user.Email)

	err := h.adminService.TriggerDownload(c.Context())
	if err != nil {
		if err.Error() == "Download or processing already in progress" {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
		}
		log.Er("Failed to trigger download", err)
		return c.Status(fiber.StatusInternalServerError).
			JSON(fiber.Map{"error": "Failed to trigger download"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Download triggered successfully"})
}

func (h *AdminHandler) triggerReprocess(c *fiber.Ctx) error {
	log := h.log.Function("triggerReprocess")

	user := middleware.GetUser(c)

	log.Info("Admin triggering reprocess", "userID", user.ID, "email", user.Email)

	err := h.adminService.TriggerReprocess(c.Context())
	if err != nil {
		if err.Error() == "No processing record found" ||
			err.Error() == "Files must be downloaded before reprocessing" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		log.Er("Failed to trigger reprocess", err)
		return c.Status(fiber.StatusInternalServerError).
			JSON(fiber.Map{"error": "Failed to trigger reprocess"})
	}

	return c.Status(fiber.StatusOK).
		JSON(fiber.Map{"message": "Reprocessing triggered successfully"})
}
