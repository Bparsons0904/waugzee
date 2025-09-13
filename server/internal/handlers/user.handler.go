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

	protected := users.Group("/", h.middleware.RequireAuth(h.zitadelService))
	protected.Get("/me", h.getCurrentUser)
}

func (h *UserHandler) getCurrentUser(c *fiber.Ctx) error {
	user := middleware.GetUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	return c.JSON(fiber.Map{"user": user})
}
