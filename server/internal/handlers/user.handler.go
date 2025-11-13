package handlers

import (
	"waugzee/internal/app"
	userController "waugzee/internal/controllers/users"
	"waugzee/internal/handlers/middleware"
	"waugzee/internal/logger"
	"waugzee/internal/services"

	"github.com/gofiber/fiber/v2"
)

type UpdateDiscogsTokenRequest struct {
	Token string `json:"token"`
}

type UpdateSelectedFolderRequest struct {
	FolderID int `json:"folderId"`
}

type UserHandler struct {
	Handler
	zitadelService *services.ZitadelService
	userController userController.UserControllerInterface
}

func NewUserHandler(app app.App, router fiber.Router) *UserHandler {
	log := logger.New("handlers").File("user_handler")
	return &UserHandler{
		zitadelService: app.Services.Zitadel,
		userController: app.Controllers.User,
		Handler: Handler{
			log:        log,
			router:     router,
			middleware: app.Middleware,
		},
	}
}

func (h *UserHandler) Register() {
	users := h.router.Group("/users")

	users.Get("/me", h.getCurrentUser)
	users.Put("/me/discogs", h.updateDiscogsToken)
	users.Put("/me/folder", h.updateSelectedFolder)
	users.Put("/me/preferences", h.updateUserPreferences)
}

func (h *UserHandler) getCurrentUser(c *fiber.Ctx) error {
	user := middleware.GetUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	userData, err := h.userController.GetUser(c.Context(), user)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve user data",
		})
	}

	return c.JSON(fiber.Map{
		"user":                user,
		"folders":             userData.Folders,
		"releases":            userData.Releases,
		"styluses":            userData.Styluses,
		"playHistory":         userData.PlayHistory,
		"dailyRecommendation": userData.DailyRecommendation,
	})
}

func (h *UserHandler) updateDiscogsToken(c *fiber.Ctx) error {
	user := middleware.GetUser(c)

	var req UpdateDiscogsTokenRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	user, err := h.userController.UpdateDiscogsToken(c.Context(), user, req.Token)
	if err != nil {
		if err.Error() == "token is required" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Token is required",
			})
		}
		if err.Error() == "invalid discogs token" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid Discogs token",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to save token",
		})
	}

	return c.JSON(fiber.Map{"user": user})
}

func (h *UserHandler) updateSelectedFolder(c *fiber.Ctx) error {
	user := middleware.GetUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	var req UpdateSelectedFolderRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	updatedUser, err := h.userController.UpdateSelectedFolder(c.Context(), user, req.FolderID)
	if err != nil {
		if err.Error() == "folder not found or not owned by user" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Folder not found or not owned by user",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update selected folder",
		})
	}

	return c.JSON(fiber.Map{"user": updatedUser})
}

func (h *UserHandler) updateUserPreferences(c *fiber.Ctx) error {
	user := middleware.GetUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	var req userController.UpdateUserPreferencesRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	updatedUser, err := h.userController.UpdateUserPreferences(c.Context(), user, &req)
	if err != nil {
		errMsg := err.Error()
		if errMsg == "user configuration not found, please set up Discogs integration first" ||
			errMsg == "recentlyPlayedThresholdDays must be between 1 and 365" ||
			errMsg == "cleaningFrequencyPlays must be between 1 and 50" ||
			errMsg == "neglectedRecordsThresholdDays must be between 1 and 730" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": errMsg,
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update user preferences",
		})
	}

	return c.JSON(fiber.Map{"user": updatedUser})
}
