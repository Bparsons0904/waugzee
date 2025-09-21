package handlers

import (
	"strings"
	"time"
	"waugzee/internal/app"
	"waugzee/internal/handlers/middleware"
	"waugzee/internal/logger"
	"waugzee/internal/services"

	authController "waugzee/internal/controllers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

type AuthHandler struct {
	Handler
	authController authController.AuthControllerInterface
	zitadelService *services.ZitadelService
}

func NewAuthHandler(app app.App, router fiber.Router) *AuthHandler {
	log := logger.New("handlers").File("auth_handler")
	return &AuthHandler{
		authController: app.Controllers.Auth,
		zitadelService: app.Services.Zitadel,
		Handler: Handler{
			log:        log,
			router:     router,
			middleware: app.Middleware,
		},
	}
}

func (h *AuthHandler) Register() {
	auth := h.router.Group("/auth")

	authRateLimit := limiter.New(limiter.Config{
		Max:        10,              // 10 requests
		Expiration: 1 * time.Minute, // per minute
		KeyGenerator: func(c *fiber.Ctx) string {
			// Rate limit by IP address
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			h.log.Warn("Auth rate limit exceeded", "ip", c.IP(), "path", c.Path())
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":       "Rate limit exceeded. Too many authentication requests.",
				"retry_after": "60 seconds",
			})
		},
		SkipFailedRequests:     false, // Count failed requests
		SkipSuccessfulRequests: false, // Count all requests
	})

	auth.Get("/config", h.getAuthConfig)
	auth.Post("/callback", authRateLimit, h.oidcCallback)

	protected := auth.Group("/", h.middleware.RequireAuth(h.zitadelService))
	protected.Post("/logout", h.logout)
}

func (h *AuthHandler) getAuthConfig(c *fiber.Ctx) error {
	config, err := h.authController.GetAuthConfig()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(config)
}

func (h *AuthHandler) logout(c *fiber.Ctx) error {
	log := h.log.Function("logout")
	user := middleware.GetUser(c)

	var reqBody authController.LogoutRequest
	if err := c.BodyParser(&reqBody); err != nil {
		// Continue with logout even if body parsing fails
		log.Er(
			"Failed to parse logout request body",
			err,
			"contentType",
			c.Get("Content-Type"),
			"body",
			string(c.Body()),
		)
	}

	// Extract access token from Authorization header
	authHeader := c.Get("Authorization")
	if authHeader != "" {
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) == 2 && strings.ToLower(tokenParts[0]) == "bearer" {
			reqBody.AccessToken = tokenParts[1]
		}
	}

	response, err := h.authController.LogoutUser(c.Context(), reqBody, user)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(response)
}

func (h *AuthHandler) oidcCallback(c *fiber.Ctx) error {
	var req authController.OIDCCallbackRequest
	if err := c.BodyParser(&req); err != nil {
		h.log.Info(
			"Failed to parse callback request body",
			"error",
			err.Error(),
			"contentType",
			c.Get("Content-Type"),
			"body",
			string(c.Body()),
		)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
	}

	h.log.Info(
		"Received OIDC callback request",
		"hasIDToken",
		req.IDToken != "",
		"hasAccessToken",
		req.AccessToken != "",
		"state",
		req.State,
	)

	response, err := h.authController.HandleOIDCCallback(c.Context(), req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Authentication failed",
		})
	}

	return c.JSON(response)
}
