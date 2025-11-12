package handlers

import (
	"waugzee/internal/app"
	"waugzee/internal/controllers/admin"
	"waugzee/internal/handlers/middleware"
	"waugzee/internal/logger"

	"github.com/gofiber/fiber/v2"
)

type AdminHandler struct {
	Handler
	adminController admin.AdminControllerInterface
}

func NewAdminHandler(app app.App, router fiber.Router) *AdminHandler {
	log := logger.New("handlers").File("admin_handler")
	return &AdminHandler{
		adminController: app.Controllers.Admin,
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
	admin.Post("/downloads/reset", h.resetStuckDownload)

	admin.Get("/files", h.listStoredFiles)
	admin.Delete("/files", h.cleanupAllFiles)
	admin.Delete("/files/:yearMonth", h.cleanupYearMonth)
}

func (h *AdminHandler) getDownloadStatus(c *fiber.Ctx) error {
	log := h.log.Function("getDownloadStatus")

	status, err := h.adminController.GetDownloadStatus(c.Context())
	if err != nil {
		_ = log.Err("Failed to get download status", err)
		return c.Status(fiber.StatusInternalServerError).
			JSON(fiber.Map{"error": "Failed to get download status"})
	}

	return c.Status(fiber.StatusOK).JSON(status)
}

func (h *AdminHandler) triggerDownload(c *fiber.Ctx) error {
	log := h.log.Function("triggerDownload")

	user := middleware.GetUser(c)

	log.Info("Admin triggering download", "userID", user.ID, "email", user.Email)

	err := h.adminController.TriggerDownload(c.Context())
	if err != nil {
		if err.Error() == "download or processing already in progress" {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
		}
		_ = log.Err("Failed to trigger download", err)
		return c.Status(fiber.StatusInternalServerError).
			JSON(fiber.Map{"error": "Failed to trigger download"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Download triggered successfully"})
}

func (h *AdminHandler) triggerReprocess(c *fiber.Ctx) error {
	log := h.log.Function("triggerReprocess")

	user := middleware.GetUser(c)

	log.Info("Admin triggering reprocess", "userID", user.ID, "email", user.Email)

	err := h.adminController.TriggerReprocess(c.Context())
	if err != nil {
		if err.Error() == "no processing record found" ||
			err.Error() == "files must be downloaded before reprocessing" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		_ = log.Err("Failed to trigger reprocess", err)
		return c.Status(fiber.StatusInternalServerError).
			JSON(fiber.Map{"error": "Failed to trigger reprocess"})
	}

	return c.Status(fiber.StatusOK).
		JSON(fiber.Map{"message": "Reprocessing triggered successfully"})
}

func (h *AdminHandler) resetStuckDownload(c *fiber.Ctx) error {
	log := h.log.Function("resetStuckDownload")

	user := middleware.GetUser(c)

	log.Info("Admin resetting stuck download", "userID", user.ID, "email", user.Email)

	err := h.adminController.ResetStuckDownload(c.Context())
	if err != nil {
		if err.Error() == "no processing record found" ||
			err.Error() == "cannot reset record in this state" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		_ = log.Err("Failed to reset download", err)
		return c.Status(fiber.StatusInternalServerError).
			JSON(fiber.Map{"error": "Failed to reset download"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Download reset successfully. You can now trigger a new download.",
	})
}

func (h *AdminHandler) listStoredFiles(c *fiber.Ctx) error {
	log := h.log.Function("listStoredFiles")

	response, err := h.adminController.ListStoredFiles(c.Context())
	if err != nil {
		_ = log.Err("Failed to list stored files", err)
		return c.Status(fiber.StatusInternalServerError).
			JSON(fiber.Map{"error": "Failed to list stored files"})
	}

	return c.Status(fiber.StatusOK).JSON(response)
}

func (h *AdminHandler) cleanupAllFiles(c *fiber.Ctx) error {
	log := h.log.Function("cleanupAllFiles")

	user := middleware.GetUser(c)

	log.Info("Admin cleaning up all files", "userID", user.ID, "email", user.Email)

	err := h.adminController.CleanupAllFiles(c.Context())
	if err != nil {
		_ = log.Err("Failed to cleanup all files", err)
		return c.Status(fiber.StatusInternalServerError).
			JSON(fiber.Map{"error": "Failed to cleanup all files"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "All files cleaned up successfully",
	})
}

func (h *AdminHandler) cleanupYearMonth(c *fiber.Ctx) error {
	log := h.log.Function("cleanupYearMonth")

	yearMonth := c.Params("yearMonth")
	if yearMonth == "" {
		return c.Status(fiber.StatusBadRequest).
			JSON(fiber.Map{"error": "yearMonth parameter is required"})
	}

	user := middleware.GetUser(c)

	log.Info("Admin cleaning up year-month files",
		"userID", user.ID,
		"email", user.Email,
		"yearMonth", yearMonth)

	err := h.adminController.CleanupYearMonth(c.Context(), yearMonth)
	if err != nil {
		_ = log.Err("Failed to cleanup year-month files", err, "yearMonth", yearMonth)
		return c.Status(fiber.StatusInternalServerError).
			JSON(fiber.Map{"error": "Failed to cleanup year-month files"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Year-month files cleaned up successfully",
		"yearMonth": yearMonth,
	})
}
