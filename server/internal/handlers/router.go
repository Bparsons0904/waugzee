package handlers

import (
	"waugzee/internal/app"
	"waugzee/internal/handlers/middleware"
	"waugzee/internal/logger"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

type Handler struct {
	middleware middleware.Middleware
	log        logger.Logger
	router     fiber.Router
}

func Router(router fiber.Router, app *app.App) (err error) {
	setupWebSocketRoute(router, app)

	api := router.Group("/api")
	HealthHandler(api, app.Config)
	NewAuthHandler(*app, api).Register()
	NewUserHandler(*app, api).Register()
	NewAdminHandler(*app, api).Register()

	return nil
}

func setupWebSocketRoute(router fiber.Router, app *app.App) {
	router.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})
	router.Get("/ws", websocket.New(func(c *websocket.Conn) {
		app.Websocket.HandleWebSocket(c)
	}))
}
