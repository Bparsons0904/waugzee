package handlers

import (
	"waugzee/internal/app"
	"waugzee/internal/handlers/middleware"
	"waugzee/internal/logger"
	"waugzee/internal/services"

	"github.com/gofiber/fiber/v2"
)

type UserHandler struct {
	Handler
	zitadelService *services.ZitadelService
}

func NewUserHandler(app app.App, router fiber.Router) *UserHandler {
	log := logger.New("handlers").File("user_handler")
	return &UserHandler{
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

	// Protected user endpoints - require valid OIDC token
	protected := users.Group("/", h.middleware.RequireAuth(h.zitadelService))
	protected.Get("/me", h.getCurrentUser)
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
