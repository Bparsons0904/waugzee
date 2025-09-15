package handlers

import (
	"waugzee/internal/app"
	adminController "waugzee/internal/controllers/admin"
	"waugzee/internal/handlers/middleware"
	"waugzee/internal/logger"
	"waugzee/internal/services"

	"github.com/gofiber/fiber/v2"
)

type AdminHandler struct {
	Handler
	zitadelService  *services.ZitadelService
	adminController *adminController.AdminController
}

func NewAdminHandler(app app.App, router fiber.Router) *AdminHandler {
	log := logger.New("handlers").File("admin_handler")
	return &AdminHandler{
		zitadelService:  app.ZitadelService,
		adminController: app.AdminController,
		Handler: Handler{
			log:        log,
			router:     router,
			middleware: app.Middleware,
		},
	}
}

func (h *AdminHandler) Register() {
	admin := h.router.Group("/admin")

	// All admin endpoints require authentication
	protected := admin.Group("/", h.middleware.RequireAuth(h.zitadelService))

	// Discogs parsing endpoints
	discogs := protected.Group("/discogs")
	discogs.Post("/parse", h.parseFile)
	discogs.Post("/process", h.processWorkflow)
}

// parseFile handles direct parsing of XML files without database operations
func (h *AdminHandler) parseFile(c *fiber.Ctx) error {
	log := h.log.Function("parseFile")

	// Check if user is authenticated (additional security for admin endpoints)
	user := middleware.GetUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	var req adminController.SimplifiedRequest
	if err := c.BodyParser(&req); err != nil {
		log.Warn("Invalid request body", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
	}

	// Validate required fields
	if len(req.FileTypes) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "fileTypes is required and must contain at least one file type",
		})
	}

	log.Info("Processing simplified parse request",
		"userID", user.ID,
		"fileTypes", req.FileTypes,
		"limits", req.Limits)

	// Call controller to handle simplified parsing
	response, err := h.adminController.SimplifiedParse(c.Context(), req)
	if err != nil {
		_ = log.Error("Simplified parse failed", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to parse files",
			"details": err.Error(),
		})
	}

	return c.JSON(response)
}

// processWorkflow handles the full processing workflow
func (h *AdminHandler) processWorkflow(c *fiber.Ctx) error {
	log := h.log.Function("processWorkflow")

	// Check if user is authenticated (additional security for admin endpoints)
	user := middleware.GetUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	var req adminController.SimplifiedRequest
	if err := c.BodyParser(&req); err != nil {
		log.Warn("Invalid request body", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
	}

	// Validate required fields
	if len(req.FileTypes) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "fileTypes is required and must contain at least one file type",
		})
	}

	log.Info("Processing simplified workflow request",
		"userID", user.ID,
		"fileTypes", req.FileTypes,
		"limits", req.Limits)

	// Call controller to handle simplified workflow processing
	response, err := h.adminController.SimplifiedProcess(c.Context(), req)
	if err != nil {
		_ = log.Error("Simplified process workflow failed", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to process workflow",
			"details": err.Error(),
		})
	}

	return c.JSON(response)
}