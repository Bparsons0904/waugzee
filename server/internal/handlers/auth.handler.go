package handlers

import (
	"strings"
	"waugzee/internal/app"
	"waugzee/internal/handlers/middleware"
	"waugzee/internal/logger"
	"waugzee/internal/models"
	"waugzee/internal/repositories"
	"waugzee/internal/services"

	"github.com/gofiber/fiber/v2"
)

type AuthHandler struct {
	Handler
	zitadelService *services.ZitadelService
	userRepo       repositories.UserRepository
}

func NewAuthHandler(app app.App, router fiber.Router) *AuthHandler {
	log := logger.New("handlers").File("auth_handler")
	return &AuthHandler{
		zitadelService: app.ZitadelService,
		userRepo:       app.UserRepo,
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
	auth.Post("/token-exchange", h.exchangeToken)
	auth.Post("/callback", h.oidcCallback)
	
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

	// Fetch user from our local database using OIDC user ID
	user, err := h.userRepo.GetByOIDCUserID(c.Context(), authInfo.UserID)
	if err != nil {
		log.Info("failed to fetch user from database", "error", err.Error(), "oidcUserID", authInfo.UserID)
		// Fallback to basic info from token if database fetch fails
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

	// Return user profile from our database (more complete and faster)
	userProfile := user.ToProfile()
	userProfile.ID = authInfo.UserID // Use OIDC ID for consistency with frontend

	log.Info("user profile retrieved from database", "userID", user.ID, "oidcUserID", authInfo.UserID)
	return c.JSON(fiber.Map{
		"user": userProfile,
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

// exchangeToken handles the OIDC authorization code exchange for access token
func (h *AuthHandler) exchangeToken(c *fiber.Ctx) error {
	log := h.log.Function("exchangeToken")

	if !h.zitadelService.IsConfigured() {
		log.Info("Zitadel not configured")
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Authentication not configured",
		})
	}

	var req services.TokenExchangeRequest
	if err := c.BodyParser(&req); err != nil {
		log.Info("invalid request body", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Code == "" || req.RedirectURI == "" {
		log.Info("missing required fields", "code", req.Code != "", "redirectURI", req.RedirectURI != "")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Code and redirect_uri are required",
		})
	}

	// Exchange code for token
	tokenResp, err := h.zitadelService.ExchangeCodeForToken(c.Context(), req)
	if err != nil {
		log.Info("token exchange failed", "error", err.Error())
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Token exchange failed",
		})
	}

	// Validate the ID token and get user info (use ID token for public OIDC clients)
	var tokenInfo *services.TokenInfo
	if tokenResp.IDToken != "" {
		tokenInfo, err = h.zitadelService.ValidateIDToken(c.Context(), tokenResp.IDToken)
	} else {
		// Fallback to access token validation for confidential clients
		tokenInfo, err = h.zitadelService.ValidateToken(c.Context(), tokenResp.AccessToken)
	}
	
	if err != nil || !tokenInfo.Valid {
		log.Info("token validation failed", "error", err, "hasIDToken", tokenResp.IDToken != "")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid token received",
		})
	}

	// Find or create user from OIDC claims
	oidcReq := models.OIDCUserCreateRequest{
		OIDCUserID:      tokenInfo.UserID,
		Email:           &tokenInfo.Email,
		Name:            &tokenInfo.Name,
		FirstName:       tokenInfo.Name, // Will be split if needed
		LastName:        "",
		OIDCProvider:    "zitadel",
		OIDCProjectID:   &tokenInfo.ProjectID,
		ProfileVerified: true,
	}

	// Split name into first/last if possible
	if tokenInfo.Name != "" {
		names := strings.Fields(tokenInfo.Name)
		if len(names) > 0 {
			oidcReq.FirstName = names[0]
		}
		if len(names) > 1 {
			oidcReq.LastName = strings.Join(names[1:], " ")
		}
	}

	user, err := h.userRepo.FindOrCreateOIDCUser(c.Context(), oidcReq)
	if err != nil {
		log.Info("failed to find or create OIDC user", "error", err.Error(), "oidcUserID", tokenInfo.UserID)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create user session",
		})
	}

	log.Info("token exchange successful", "userID", user.ID, "email", user.Email)
	return c.JSON(fiber.Map{
		"access_token": tokenResp.AccessToken,
		"token_type":   tokenResp.TokenType,
		"expires_in":   tokenResp.ExpiresIn,
		"user":         user.ToProfile(),
	})
}

// oidcCallback handles the OIDC callback from Zitadel (alternative to token exchange)
func (h *AuthHandler) oidcCallback(c *fiber.Ctx) error {
	log := h.log.Function("oidcCallback")

	if !h.zitadelService.IsConfigured() {
		log.Info("Zitadel not configured")
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Authentication not configured",
		})
	}

	// Get parameters from query or body
	code := c.Query("code")
	if code == "" {
		code = c.FormValue("code")
	}

	redirectURI := c.Query("redirect_uri")
	if redirectURI == "" {
		redirectURI = c.FormValue("redirect_uri")
	}

	state := c.Query("state")
	if state == "" {
		state = c.FormValue("state")
	}

	if code == "" || redirectURI == "" {
		log.Info("missing required parameters", "code", code != "", "redirectURI", redirectURI != "")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Code and redirect_uri are required",
		})
	}

	// Exchange code for token
	tokenReq := services.TokenExchangeRequest{
		Code:        code,
		RedirectURI: redirectURI,
		State:       state,
	}

	tokenResp, err := h.zitadelService.ExchangeCodeForToken(c.Context(), tokenReq)
	if err != nil {
		log.Info("OIDC callback token exchange failed", "error", err.Error())
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication failed",
		})
	}

	// Validate the ID token and get user info (use ID token for public OIDC clients)
	var tokenInfo *services.TokenInfo
	if tokenResp.IDToken != "" {
		tokenInfo, err = h.zitadelService.ValidateIDToken(c.Context(), tokenResp.IDToken)
	} else {
		// Fallback to access token validation for confidential clients
		tokenInfo, err = h.zitadelService.ValidateToken(c.Context(), tokenResp.AccessToken)
	}
	
	if err != nil || !tokenInfo.Valid {
		log.Info("OIDC callback token validation failed", "error", err, "hasIDToken", tokenResp.IDToken != "")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication failed",
		})
	}

	// Find or create user from OIDC claims
	oidcReq := models.OIDCUserCreateRequest{
		OIDCUserID:      tokenInfo.UserID,
		Email:           &tokenInfo.Email,
		Name:            &tokenInfo.Name,
		FirstName:       tokenInfo.Name,
		LastName:        "",
		OIDCProvider:    "zitadel",
		OIDCProjectID:   &tokenInfo.ProjectID,
		ProfileVerified: true,
	}

	// Split name into first/last if possible
	if tokenInfo.Name != "" {
		names := strings.Fields(tokenInfo.Name)
		if len(names) > 0 {
			oidcReq.FirstName = names[0]
		}
		if len(names) > 1 {
			oidcReq.LastName = strings.Join(names[1:], " ")
		}
	}

	user, err := h.userRepo.FindOrCreateOIDCUser(c.Context(), oidcReq)
	if err != nil {
		log.Info("OIDC callback failed to create user", "error", err.Error(), "oidcUserID", tokenInfo.UserID)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Authentication failed",
		})
	}

	log.Info("OIDC callback successful", "userID", user.ID, "email", user.Email)
	return c.JSON(fiber.Map{
		"access_token": tokenResp.AccessToken,
		"token_type":   tokenResp.TokenType,
		"expires_in":   tokenResp.ExpiresIn,
		"state":        state,
		"user":         user.ToProfile(),
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