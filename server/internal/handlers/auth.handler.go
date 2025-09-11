package handlers

import (
	"context"
	"fmt"
	"strings"
	"time"
	"waugzee/internal/app"
	"waugzee/internal/database"
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
	db             database.DB
}

func NewAuthHandler(app app.App, router fiber.Router) *AuthHandler {
	log := logger.New("handlers").File("auth_handler")
	return &AuthHandler{
		zitadelService: app.ZitadelService,
		userRepo:       app.UserRepo,
		db:             app.Database,
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
	admin := auth.Group(
		"/admin",
		h.middleware.RequireAuth(h.zitadelService),
		h.middleware.RequireRole("admin"),
	)
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
		"configured":  true,
		"domain":      "configured", // Domain info available via service
		"instanceUrl": "configured", // Instance URL available via service
		"clientId":    "configured", // Client ID available via service
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
	codeChallenge := c.Query("code_challenge") // PKCE code challenge
	nonce := c.Query("nonce")                 // Nonce for replay attack protection

	if redirectURI == "" {
		log.Info("missing redirect_uri parameter")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "redirect_uri parameter is required",
		})
	}

	// Store nonce in cache if provided (for later validation during token exchange)
	if nonce != "" {
		err := h.storeNonce(c.Context(), nonce)
		if err != nil {
			log.Info("failed to store nonce", "error", err.Error())
			// Continue without failing - nonce validation will be optional
		}
	}

	authURL := h.zitadelService.GetAuthorizationURL(state, redirectURI, codeChallenge, nonce)
	if authURL == "" {
		log.Info("failed to generate authorization URL")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate authorization URL",
		})
	}

	log.Info("generated authorization URL", "state", state, "redirectURI", redirectURI, "hasPKCE", codeChallenge != "")
	return c.JSON(fiber.Map{
		"authorizationUrl": authURL,
		"state":            state,
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
		log.Info(
			"failed to fetch user from database",
			"error",
			err.Error(),
			"oidcUserID",
			authInfo.UserID,
		)
		// Fallback to basic info from token if database fetch fails
		return c.JSON(fiber.Map{
			"user": fiber.Map{
				"id":        authInfo.UserID,
				"email":     authInfo.Email,
				"name":      authInfo.Name,
				"roles":     authInfo.Roles,
				"projectId": authInfo.ProjectID,
			},
		})
	}

	// Return user profile from our database (more complete and faster)
	userProfile := user.ToProfile()
	userProfile.ID = authInfo.UserID // Use OIDC ID for consistency with frontend

	log.Info(
		"user profile retrieved from database",
		"userID",
		user.ID,
		"oidcUserID",
		authInfo.UserID,
	)
	return c.JSON(fiber.Map{
		"user": userProfile,
	})
}

// LogoutRequest represents the logout request body
type LogoutRequest struct {
	RefreshToken          string `json:"refresh_token,omitempty"`
	IDToken               string `json:"id_token,omitempty"`
	PostLogoutRedirectURI string `json:"post_logout_redirect_uri,omitempty"`
	State                 string `json:"state,omitempty"`
}

// logout handles user logout with proper OIDC token revocation and cache cleanup
func (h *AuthHandler) logout(c *fiber.Ctx) error {
	log := h.log.Function("logout")

	authInfo := middleware.GetAuthInfo(c)
	var userID string
	if authInfo != nil {
		userID = authInfo.UserID
		log.Info("processing logout request", "userID", userID)
	}

	// Parse request body for logout parameters
	var req LogoutRequest
	if err := c.BodyParser(&req); err != nil {
		log.Info("failed to parse logout request body", "error", err.Error())
		// Continue with logout even if body parsing fails
	}

	var revokedTokens []string
	var accessToken string

	// Extract access token from Authorization header
	authHeader := c.Get("Authorization")
	if authHeader != "" {
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) == 2 && strings.ToLower(tokenParts[0]) == "bearer" {
			accessToken = tokenParts[1]
		}
	}

	// Revoke access token if present
	if accessToken != "" && h.zitadelService.IsConfigured() {
		if err := h.zitadelService.RevokeToken(c.Context(), accessToken, "access_token"); err != nil {
			log.Warn("failed to revoke access token", "error", err.Error())
			// Don't block logout on revocation failure
		} else {
			revokedTokens = append(revokedTokens, "access_token")
			log.Info("access token revoked successfully")
		}
	}

	// Revoke refresh token if provided
	if req.RefreshToken != "" && h.zitadelService.IsConfigured() {
		if err := h.zitadelService.RevokeToken(c.Context(), req.RefreshToken, "refresh_token"); err != nil {
			log.Warn("failed to revoke refresh token", "error", err.Error())
			// Don't block logout on revocation failure
		} else {
			revokedTokens = append(revokedTokens, "refresh_token")
			log.Info("refresh token revoked successfully")
		}
	}

	// Clear user cache data if we have auth info
	if authInfo != nil {
		// Get user from database to find UUID for cache cleanup
		user, err := h.userRepo.GetByOIDCUserID(c.Context(), authInfo.UserID)
		if err != nil {
			log.Warn(
				"failed to get user for cache cleanup",
				"error",
				err.Error(),
				"oidcUserID",
				authInfo.UserID,
			)
		} else {
			// Use the repository's cache clearing logic from Delete method
			// Clear user cache by UUID
			if err := h.clearUserCache(user.ID.String(), authInfo.UserID); err != nil {
				log.Warn("failed to clear user cache", "error", err.Error(), "userID", user.ID)
			} else {
				log.Info("user cache cleared successfully", "userID", user.ID)
			}
		}
	}

	// Generate logout URL using Zitadel service
	var logoutURL string
	if h.zitadelService.IsConfigured() {
		// Use ID token if provided in request body, otherwise use empty string
		// (don't use access token as ID token hint - that causes Zitadel to return invalid_request)
		idTokenHint := req.IDToken

		url, err := h.zitadelService.GetLogoutURL(
			c.Context(),
			idTokenHint,
			req.PostLogoutRedirectURI,
			req.State,
		)
		if err != nil {
			log.Warn("failed to generate logout URL", "error", err.Error())
			// Continue with basic logout response
		} else {
			logoutURL = url
			log.Info("logout URL generated successfully")
		}
	}

	if userID != "" {
		log.Info("user logout completed", "userID", userID, "revokedTokens", len(revokedTokens))
	}

	response := fiber.Map{
		"message": "Logout successful",
	}

	if logoutURL != "" {
		response["logout_url"] = logoutURL
	}

	if len(revokedTokens) > 0 {
		response["revoked_tokens"] = revokedTokens
	}

	return c.JSON(response)
}

// clearUserCache clears both user cache and OIDC mapping cache
func (h *AuthHandler) clearUserCache(userUUID, oidcUserID string) error {
	log := h.log.Function("clearUserCache")

	// Clear user cache by UUID (mimics userRepository.Delete cache clearing logic)
	if err := database.NewCacheBuilder(h.db.Cache.User, userUUID).Delete(); err != nil {
		log.Warn("failed to remove user from cache", "userID", userUUID, "error", err)
		return err
	}

	// Clear OIDC mapping cache
	if oidcUserID != "" {
		oidcCacheKey := "oidc:" + oidcUserID
		if err := database.NewCacheBuilder(h.db.Cache.User, oidcCacheKey).Delete(); err != nil {
			log.Warn(
				"failed to remove OIDC mapping from cache",
				"oidcUserID",
				oidcUserID,
				"error",
				err,
			)
			return err
		}
	}

	return nil
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
		log.Info(
			"missing required fields",
			"code",
			req.Code != "",
			"redirectURI",
			req.RedirectURI != "",
			"hasCodeVerifier",
			req.CodeVerifier != "",
		)
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

	// Validate nonce if present in ID token (anti-replay protection)
	if tokenInfo.Nonce != "" {
		if err := h.validateAndCleanupNonce(c.Context(), tokenInfo.Nonce); err != nil {
			log.Info("nonce validation failed", "error", err.Error(), "nonce", tokenInfo.Nonce)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Nonce validation failed",
			})
		}
		log.Info("nonce validation successful", "nonce", tokenInfo.Nonce)
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
		log.Info(
			"failed to find or create OIDC user",
			"error",
			err.Error(),
			"oidcUserID",
			tokenInfo.UserID,
		)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create user session",
		})
	}

	log.Info("token exchange successful", "userID", user.ID, "email", user.Email)

	// Build response including ID token for logout flow
	response := fiber.Map{
		"access_token": tokenResp.AccessToken,
		"token_type":   tokenResp.TokenType,
		"expires_in":   tokenResp.ExpiresIn,
		"user":         user.ToProfile(),
	}

	// Include ID token if available (needed for proper OIDC logout)
	if tokenResp.IDToken != "" {
		response["id_token"] = tokenResp.IDToken
	}

	return c.JSON(response)
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

	codeVerifier := c.Query("code_verifier")
	if codeVerifier == "" {
		codeVerifier = c.FormValue("code_verifier")
	}

	if code == "" || redirectURI == "" {
		log.Info(
			"missing required parameters",
			"code",
			code != "",
			"redirectURI",
			redirectURI != "",
		)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Code and redirect_uri are required",
		})
	}

	// Exchange code for token
	tokenReq := services.TokenExchangeRequest{
		Code:         code,
		RedirectURI:  redirectURI,
		State:        state,
		CodeVerifier: codeVerifier, // Include PKCE code verifier
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
		log.Info(
			"OIDC callback token validation failed",
			"error",
			err,
			"hasIDToken",
			tokenResp.IDToken != "",
		)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication failed",
		})
	}

	// Validate nonce if present in ID token (anti-replay protection)
	if tokenInfo.Nonce != "" {
		if err := h.validateAndCleanupNonce(c.Context(), tokenInfo.Nonce); err != nil {
			log.Info("OIDC callback nonce validation failed", "error", err.Error(), "nonce", tokenInfo.Nonce)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Authentication failed",
			})
		}
		log.Info("OIDC callback nonce validation successful", "nonce", tokenInfo.Nonce)
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
		log.Info(
			"OIDC callback failed to create user",
			"error",
			err.Error(),
			"oidcUserID",
			tokenInfo.UserID,
		)
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

// storeNonce stores a nonce value in the cache with TTL for later validation
func (h *AuthHandler) storeNonce(ctx context.Context, nonce string) error {
	log := h.log.Function("storeNonce")

	if nonce == "" {
		return fmt.Errorf("nonce is required")
	}

	// Store nonce with 10 minute TTL (same as OIDC state parameter lifetime)
	err := database.NewCacheBuilder(h.db.Cache.Session, nonce).
		WithHashPattern("nonce:%s").
		WithValue("1"). // Simple marker value
		WithTTL(10 * time.Minute).
		WithContext(ctx).
		Set()

	if err != nil {
		return log.Err("failed to store nonce in cache", err, "nonce", nonce)
	}

	log.Info("nonce stored successfully", "nonce", nonce)
	return nil
}

// validateAndCleanupNonce validates a nonce from ID token and removes it from cache
func (h *AuthHandler) validateAndCleanupNonce(ctx context.Context, nonce string) error {
	log := h.log.Function("validateAndCleanupNonce")

	if nonce == "" {
		return fmt.Errorf("nonce is required")
	}

	// Check if nonce exists in cache
	var result string
	found, err := database.NewCacheBuilder(h.db.Cache.Session, nonce).
		WithHashPattern("nonce:%s").
		WithContext(ctx).
		Get(&result)

	if err != nil {
		return log.Err("failed to validate nonce", err, "nonce", nonce)
	}

	if !found {
		return log.Err("nonce not found or expired", fmt.Errorf("nonce validation failed"), "nonce", nonce)
	}

	// Remove nonce from cache (one-time use)
	err = database.NewCacheBuilder(h.db.Cache.Session, nonce).
		WithHashPattern("nonce:%s").
		WithContext(ctx).
		Delete()

	if err != nil {
		log.Warn("failed to cleanup nonce from cache", "error", err.Error(), "nonce", nonce)
		// Don't fail validation if cleanup fails
	}

	log.Info("nonce validated and cleaned up successfully", "nonce", nonce)
	return nil
}

