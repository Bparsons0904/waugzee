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
		authController: app.AuthController,
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

	// Rate limiting for sensitive auth endpoints
	authRateLimit := limiter.New(limiter.Config{
		Max:        10,                   // 10 requests
		Expiration: 1 * time.Minute,     // per minute
		KeyGenerator: func(c *fiber.Ctx) string {
			// Rate limit by IP address
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			h.log.Warn("Auth rate limit exceeded", "ip", c.IP(), "path", c.Path())
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Rate limit exceeded. Too many authentication requests.",
				"retry_after": "60 seconds",
			})
		},
		SkipFailedRequests:     false, // Count failed requests
		SkipSuccessfulRequests: false, // Count all requests
	})

	// Stricter rate limiting for token exchange (most sensitive)
	tokenRateLimit := limiter.New(limiter.Config{
		Max:        5,                   // 5 requests  
		Expiration: 1 * time.Minute,     // per minute
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			h.log.Warn("Token exchange rate limit exceeded", "ip", c.IP())
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Rate limit exceeded. Too many token exchange attempts.",
				"retry_after": "60 seconds",
			})
		},
	})

	// Public endpoints with rate limiting
	auth.Get("/config", h.getAuthConfig)
	auth.Get("/login-url", authRateLimit, h.getLoginURL)
	auth.Post("/token-exchange", tokenRateLimit, h.exchangeToken)
	auth.Post("/callback", authRateLimit, h.oidcCallback)

	// Protected endpoints - require valid OIDC token
	protected := auth.Group("/", h.middleware.RequireAuth(h.zitadelService))
	protected.Post("/logout", h.logout)

	// Admin endpoints - require admin role
	admin := auth.Group(
		"/admin",
		h.middleware.RequireAuth(h.zitadelService),
		h.middleware.RequireRole("admin"),
	)
	admin.Get("/users", h.getAllUsers)
}

// getAuthConfig returns authentication configuration for the client
func (h *AuthHandler) getAuthConfig(c *fiber.Ctx) error {
	config, err := h.authController.GetAuthConfig()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(config)
}

// getLoginURL generates an authorization URL for OIDC login flow
func (h *AuthHandler) getLoginURL(c *fiber.Ctx) error {
	// Get parameters from query string
	state := c.Query("state", "default-state")
	redirectURI := c.Query("redirect_uri")
	codeChallenge := c.Query("code_challenge") // PKCE code challenge
	nonce := c.Query("nonce")                 // Nonce for replay attack protection

	if redirectURI == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "redirect_uri parameter is required",
		})
	}

	response, err := h.authController.GenerateAuthURL(state, redirectURI, codeChallenge, nonce)
	if err != nil {
		if err.Error() == "authentication not configured" {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(response)
}



// logout handles user logout with proper OIDC token revocation and cache cleanup
func (h *AuthHandler) logout(c *fiber.Ctx) error {
	authInfo := middleware.GetAuthInfo(c)

	// Parse request body for logout parameters
	var reqBody authController.LogoutRequest
	if err := c.BodyParser(&reqBody); err != nil {
		// Continue with logout even if body parsing fails
	}

	// Extract access token from Authorization header
	authHeader := c.Get("Authorization")
	if authHeader != "" {
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) == 2 && strings.ToLower(tokenParts[0]) == "bearer" {
			reqBody.AccessToken = tokenParts[1]
		}
	}

	// Convert middleware AuthInfo to controller AuthInfo
	var controllerAuthInfo *authController.AuthInfo
	if authInfo != nil {
		controllerAuthInfo = &authController.AuthInfo{
			UserID:    authInfo.UserID,
			Email:     authInfo.Email,
			Name:      authInfo.Name,
			Roles:     authInfo.Roles,
			ProjectID: authInfo.ProjectID,
		}
	}

	response, err := h.authController.LogoutUser(c.Context(), reqBody, controllerAuthInfo)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(response)
}


// exchangeToken handles the OIDC authorization code exchange for access token
func (h *AuthHandler) exchangeToken(c *fiber.Ctx) error {
	var req services.TokenExchangeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	response, err := h.authController.ValidateAndExchangeToken(c.Context(), req)
	if err != nil {
		switch err.Error() {
		case "authentication not configured":
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error": err.Error(),
			})
		case "code and redirect_uri are required":
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		case "token exchange failed", "invalid token received", "nonce validation failed":
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": err.Error(),
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
	}

	return c.JSON(response)
}

// oidcCallback handles the OIDC callback from Zitadel (alternative to token exchange)
func (h *AuthHandler) oidcCallback(c *fiber.Ctx) error {
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

	codeVerifier := c.Query("code_verifier")
	if codeVerifier == "" {
		codeVerifier = c.FormValue("code_verifier")
	}

	req := authController.OIDCCallbackRequest{
		Code:         code,
		RedirectURI:  redirectURI,
		State:        state,
		CodeVerifier: codeVerifier,
	}

	response, err := h.authController.HandleOIDCCallback(c.Context(), req)
	if err != nil {
		switch err.Error() {
		case "authentication not configured":
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error": err.Error(),
			})
		case "code and redirect_uri are required":
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		case "authentication failed":
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": err.Error(),
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
	}

	return c.JSON(response)
}

// getAllUsers returns all users (admin only)
func (h *AuthHandler) getAllUsers(c *fiber.Ctx) error {
	authInfo := middleware.GetAuthInfo(c)
	
	// Convert middleware AuthInfo to controller AuthInfo
	var controllerAuthInfo *authController.AuthInfo
	if authInfo != nil {
		controllerAuthInfo = &authController.AuthInfo{
			UserID:    authInfo.UserID,
			Email:     authInfo.Email,
			Name:      authInfo.Name,
			Roles:     authInfo.Roles,
			ProjectID: authInfo.ProjectID,
		}
	}

	response, err := h.authController.GetAllUsers(c.Context(), controllerAuthInfo)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(response)
}


