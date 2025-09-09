package handlers

import (
	"waugzee/internal/app"
	userController "waugzee/internal/controllers/users"
	"waugzee/internal/logger"
	. "waugzee/internal/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type UserHandler struct {
	Handler
	controller userController.UserController
}

func NewUserHandler(app app.App, router fiber.Router) *UserHandler {
	log := logger.New("handlers").File("user_handler")
	return &UserHandler{
		controller: *app.UserController,
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
}

func (h *UserHandler) getUser(c *fiber.Ctx) error {

	userID := uuid.MustParse("0198ca62-4fff-7923-9adb-f3a93a37fee2")
	user := User{
		BaseUUIDModel: BaseUUIDModel{
			ID: userID,
		},
		FirstName: "John",
		LastName:  "Doe",
		Login:     "johndoe",
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
