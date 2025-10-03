package handlers

import (
	"waugzee/internal/app"
	userStylusController "waugzee/internal/controllers/userStylus"
	"waugzee/internal/handlers/middleware"
	"waugzee/internal/logger"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type UserStylusHandler struct {
	Handler
	userStylusController userStylusController.UserStylusControllerInterface
}

func NewUserStylusHandler(app app.App, router fiber.Router) *UserStylusHandler {
	log := logger.New("handlers").File("userStylus_handler")
	return &UserStylusHandler{
		userStylusController: app.Controllers.UserStylus,
		Handler: Handler{
			log:        log,
			router:     router,
			middleware: app.Middleware,
		},
	}
}

func (h *UserStylusHandler) Register() {
	styluses := h.router.Group("/styluses")

	styluses.Get("", h.getUserStyluses)
	styluses.Post("", h.createUserStylus)
	styluses.Patch("/:id", h.updateUserStylus)
	styluses.Delete("/:id", h.deleteUserStylus)
}

func (h *UserStylusHandler) getUserStyluses(c *fiber.Ctx) error {
	user := middleware.GetUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	styluses, err := h.userStylusController.GetUserStyluses(c.Context(), user)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve user styluses",
		})
	}

	return c.JSON(fiber.Map{
		"styluses": styluses,
	})
}

func (h *UserStylusHandler) createUserStylus(c *fiber.Ctx) error {
	user := middleware.GetUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	var req userStylusController.CreateUserStylusRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := h.userStylusController.CreateUserStylus(c.Context(), user, &req); err != nil {
		if err.Error() == "stylusId is required" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "stylusId is required",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create user stylus",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
	})
}

func (h *UserStylusHandler) updateUserStylus(c *fiber.Ctx) error {
	user := middleware.GetUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	stylusIDParam := c.Params("id")
	stylusID, err := uuid.Parse(stylusIDParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid stylus ID",
		})
	}

	var req userStylusController.UpdateUserStylusRequest
	if err = c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := h.userStylusController.UpdateUserStylus(c.Context(), user, stylusID, &req); err != nil {
		if err.Error() == "user stylus not found or not owned by user" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "User stylus not found or not owned by user",
			})
		}
		if err.Error() == "no fields to update" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "No fields to update",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update user stylus",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
	})
}

func (h *UserStylusHandler) deleteUserStylus(c *fiber.Ctx) error {
	user := middleware.GetUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	stylusIDParam := c.Params("id")
	stylusID, err := uuid.Parse(stylusIDParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid stylus ID",
		})
	}

	if err := h.userStylusController.DeleteUserStylus(c.Context(), user, stylusID); err != nil {
		if err.Error() == "user stylus not found or not owned by user" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "User stylus not found or not owned by user",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete user stylus",
		})
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}
