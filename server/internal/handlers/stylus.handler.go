package handlers

import (
	"waugzee/internal/app"
	stylusController "waugzee/internal/controllers/stylus"
	"waugzee/internal/handlers/middleware"
	"waugzee/internal/logger"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type StylusHandler struct {
	Handler
	stylusController stylusController.StylusControllerInterface
}

func NewStylusHandler(app app.App, router fiber.Router) *StylusHandler {
	log := logger.New("handlers").File("stylus_handler")
	return &StylusHandler{
		stylusController: app.Controllers.Stylus,
		Handler: Handler{
			log:        log,
			router:     router,
			middleware: app.Middleware,
		},
	}
}

func (h *StylusHandler) Register() {
	styluses := h.router.Group("/styluses")

	styluses.Get("/available", h.getAvailableStyluses)
	styluses.Post("/custom", h.createCustomStylus)
	styluses.Get("", h.getUserStyluses)
	styluses.Post("", h.createUserStylus)
	styluses.Patch("/:id", h.updateUserStylus)
	styluses.Delete("/:id", h.deleteUserStylus)
}

func (h *StylusHandler) getAvailableStyluses(c *fiber.Ctx) error {
	user := middleware.GetUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	styluses, err := h.stylusController.GetAvailableStyluses(c.Context(), user)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve available styluses",
		})
	}

	return c.JSON(fiber.Map{
		"styluses": styluses,
	})
}

func (h *StylusHandler) createCustomStylus(c *fiber.Ctx) error {
	user := middleware.GetUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	var req stylusController.CreateCustomStylusRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	stylus, err := h.stylusController.CreateCustomStylus(c.Context(), user, &req)
	if err != nil {
		if err.Error() == "brand is required" || err.Error() == "model is required" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create custom stylus",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"stylus": stylus,
	})
}

func (h *StylusHandler) getUserStyluses(c *fiber.Ctx) error {
	user := middleware.GetUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	styluses, err := h.stylusController.GetUserStyluses(c.Context(), user)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve user styluses",
		})
	}

	return c.JSON(fiber.Map{
		"styluses": styluses,
	})
}

func (h *StylusHandler) createUserStylus(c *fiber.Ctx) error {
	user := middleware.GetUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	var req stylusController.CreateUserStylusRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := h.stylusController.CreateUserStylus(c.Context(), user, &req); err != nil {
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

func (h *StylusHandler) updateUserStylus(c *fiber.Ctx) error {
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

	var req stylusController.UpdateUserStylusRequest
	if err = c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := h.stylusController.UpdateUserStylus(c.Context(), user, stylusID, &req); err != nil {
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

func (h *StylusHandler) deleteUserStylus(c *fiber.Ctx) error {
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

	if err := h.stylusController.DeleteUserStylus(c.Context(), user, stylusID); err != nil {
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
