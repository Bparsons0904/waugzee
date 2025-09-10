package handlers

import (
	"waugzee/internal/app"
	"waugzee/internal/handlers/middleware"
	"waugzee/internal/logger"
	"waugzee/internal/services"

	"github.com/gofiber/fiber/v2"
)

type AuthHandler struct {
	Handler
	zitadelService *services.ZitadelService
}

func NewAuthHandler(app app.App, router fiber.Router) *AuthHandler {
	log := logger.New("handlers").File("auth_handler")
	return &AuthHandler{
		zitadelService: app.ZitadelService,
		Handler: Handler{
			log:        log,
			router:     router,
			middleware: app.Middleware,
		},
	}
}

func (h *AuthHandler) Register() {
	auth := h.router.Group("/auth")
	
	// Public endpoints
	auth.Get("/config", h.getAuthConfig)
	auth.Get("/login-url", h.getLoginURL)
	
	// Protected endpoints - require valid OIDC token
	protected := auth.Group("/", h.middleware.RequireAuth(h.zitadelService))
	protected.Get("/me", h.getCurrentUser)
	protected.Post("/logout", h.logout)
	
	// Admin endpoints - require admin role
	admin := auth.Group("/admin", h.middleware.RequireAuth(h.zitadelService), h.middleware.RequireRole("admin"))
	admin.Get("/users", h.getAllUsers)
}

// getAuthConfig returns authentication configuration for the client
func (h *AuthHandler) getAuthConfig(c *fiber.Ctx) error {
	log := h.log.Function("getAuthConfig")

	if !h.zitadelService.IsConfigured() {
		log.Info("Zitadel not configured")
		return c.JSON(fiber.Map{
			"configured": false,
			"message":    "Authentication not configured",
		})
	}

	return c.JSON(fiber.Map{
		"configured": true,
		"domain":     "configured", // Domain info available via service
		"instanceUrl": "configured", // Instance URL available via service  
		"clientId":   "configured", // Client ID available via service
	})
}

// getLoginURL generates an authorization URL for OIDC login flow
func (h *AuthHandler) getLoginURL(c *fiber.Ctx) error {
	log := h.log.Function("getLoginURL")

	if !h.zitadelService.IsConfigured() {
		log.Info("Zitadel not configured")
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Authentication not configured",
		})
	}

	// Get parameters from query string
	state := c.Query("state", "default-state")
	redirectURI := c.Query("redirect_uri")
	
	if redirectURI == "" {
		log.Info("missing redirect_uri parameter")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "redirect_uri parameter is required",
		})
	}

	authURL := h.zitadelService.GetAuthorizationURL(state, redirectURI)
	if authURL == "" {
		log.Info("failed to generate authorization URL")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate authorization URL",
		})
	}

	log.Info("generated authorization URL", "state", state, "redirectURI", redirectURI)
	return c.JSON(fiber.Map{
		"authorizationUrl": authURL,
		"state":           state,
	})
}

// getCurrentUser returns information about the currently authenticated user
func (h *AuthHandler) getCurrentUser(c *fiber.Ctx) error {
	log := h.log.Function("getCurrentUser")

	authInfo := middleware.GetAuthInfo(c)
	if authInfo == nil {
		log.Info("no auth info found")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	// Optionally fetch detailed user info from Zitadel
	userInfo, err := h.zitadelService.GetUserInfo(c.Context(), authInfo.UserID)
	if err != nil {
		log.Info("failed to fetch detailed user info", "error", err.Error(), "userID", authInfo.UserID)
		// Return basic info from token if detailed fetch fails
		return c.JSON(fiber.Map{
			"user": fiber.Map{
				"id":       authInfo.UserID,
				"email":    authInfo.Email,
				"name":     authInfo.Name,
				"roles":    authInfo.Roles,
				"projectId": authInfo.ProjectID,
			},
		})
	}

	// Return detailed user info
	return c.JSON(fiber.Map{
		"user": fiber.Map{
			"id":       authInfo.UserID,
			"email":    authInfo.Email,
			"name":     authInfo.Name,
			"roles":    authInfo.Roles,
			"projectId": authInfo.ProjectID,
			"detailed": userInfo,
		},
	})
}

// logout handles user logout (client-side operation for OIDC)
func (h *AuthHandler) logout(c *fiber.Ctx) error {
	log := h.log.Function("logout")

	authInfo := middleware.GetAuthInfo(c)
	if authInfo != nil {
		log.Info("user logged out", "userID", authInfo.UserID)
	}

	// For OIDC, logout is primarily handled client-side by clearing tokens
	// Server-side logout would involve token revocation with Zitadel
	return c.JSON(fiber.Map{
		"message": "Logout successful",
		"logoutUrl": "/logout", // Logout URL would be configured via Zitadel service
	})
}

// getAllUsers returns all users (admin only)
func (h *AuthHandler) getAllUsers(c *fiber.Ctx) error {
	log := h.log.Function("getAllUsers")

	authInfo := middleware.GetAuthInfo(c)
	log.Info("admin requesting all users", "adminID", authInfo.UserID)

	// This is a placeholder - in a real implementation, you'd fetch users from Zitadel
	return c.JSON(fiber.Map{
		"message": "Admin endpoint - get all users",
		"users":   []interface{}{},
	})
}