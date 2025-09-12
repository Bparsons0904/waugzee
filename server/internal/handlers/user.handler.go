package handlers

import (
	"waugzee/internal/app"
	userController "waugzee/internal/controllers/users"
	"waugzee/internal/handlers/middleware"
	"waugzee/internal/logger"
	. "waugzee/internal/models"
	"waugzee/internal/services"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type UserHandler struct {
	Handler
	controller     userController.UserController
	zitadelService *services.ZitadelService
}

func NewUserHandler(app app.App, router fiber.Router) *UserHandler {
	log := logger.New("handlers").File("user_handler")
	return &UserHandler{
		controller:     *app.UserController,
		zitadelService: app.ZitadelService,
		Handler: Handler{
			log:        log,
			router:     router,
			middleware: app.Middleware,
		},
	}
}

func (h *UserHandler) Register() {
	users := h.router.Group("/users")
	users.Post("/login", h.login)

	users.Get("/", h.getUser)
	users.Post("/logout", h.logout)

	// Protected user endpoints - require valid OIDC token
	protected := users.Group("/", h.middleware.RequireAuth(h.zitadelService))
	protected.Get("/me", h.getCurrentUser)
}

func (h *UserHandler) getUser(c *fiber.Ctx) error {
	userID := uuid.MustParse("0198ca62-4fff-7923-9adb-f3a93a37fee2")
	user := User{
		BaseUUIDModel: BaseUUIDModel{
			ID: userID,
		},
		FirstName: "John",
		LastName:  "Doe",
		IsAdmin:   true,
	}

	return c.JSON(fiber.Map{"message": "success", "user": user})
}

func (h *UserHandler) logout(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"message": "success"})
}

func (h *UserHandler) login(c *fiber.Ctx) error {
	log := h.log.Function("login")

	var loginRequest LoginRequest
	if err := c.BodyParser(&loginRequest); err != nil {
		log.Er("failed to parse login request", err)
		return c.Status(fiber.StatusBadRequest).
			JSON(fiber.Map{"message": "failed to parse login request"})
	}

	return c.JSON(fiber.Map{"message": "success", "user": "user"})
}

// getCurrentUser returns information about the currently authenticated user
func (h *UserHandler) getCurrentUser(c *fiber.Ctx) error {
	// Get user directly from middleware context
	user := middleware.GetUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	// Convert to user profile for response
	response := fiber.Map{
		"user": user.ToProfile(),
	}

	return c.JSON(response)
}
