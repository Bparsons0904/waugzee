package middleware

import (
	"context"
	"slices"
	"strings"
	"waugzee/internal/models"
	"waugzee/internal/services"

	"github.com/gofiber/fiber/v2"
)

// AuthContextKey is used to store auth info in context
type AuthContextKey string

const (
	AuthInfoKey  AuthContextKey = "auth_info"
	UserIDKey    AuthContextKey = "user_id"
	UserKey      AuthContextKey = "user"
	UserKeyFiber string         = "User" // Fiber context key (string)
)

// AuthInfo contains user authentication information
type AuthInfo struct {
	UserID    string
	Email     string
	Name      string
	Roles     []string
	ProjectID string
}

// RequireAuth middleware validates OIDC tokens and requires authentication
func (m *Middleware) RequireAuth(zitadelService *services.ZitadelService) fiber.Handler {
	log := m.log.Function("RequireAuth")

	return func(c *fiber.Ctx) error {
		// Skip auth if Zitadel is not configured
		if !zitadelService.IsConfigured() {
			log.Warn("Zitadel not configured, skipping authentication")
			return c.Next()
		}

		// Extract token from Authorization header
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			log.Info("missing authorization header")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Authorization header required",
			})
		}

		// Check for Bearer token format
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || strings.ToLower(tokenParts[0]) != "bearer" {
			log.Info("invalid authorization header format")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid authorization header format",
			})
		}

		token := tokenParts[1]
		if token == "" {
			log.Info("empty token")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Token required",
			})
		}

		// Validate token with Zitadel - try JWT first, fallback to introspection
		var tokenInfo *services.TokenInfo
		var err error
		var validationMethod string

		if isJWTToken(token) {
			// Try JWT validation first (local, fast)
			tokenInfo, err = zitadelService.ValidateIDToken(c.Context(), token)
			validationMethod = "JWT"
			
			// If JWT validation fails due to token format/type, fallback to introspection
			if err != nil {
				log.Debug("JWT validation failed, falling back to introspection", "error", err.Error())
				tokenInfo, err = zitadelService.ValidateToken(c.Context(), token)
				validationMethod = "introspection_fallback"
			}
		} else {
			// Not a JWT token, use introspection directly
			tokenInfo, err = zitadelService.ValidateToken(c.Context(), token)
			validationMethod = "introspection"
		}

		if err != nil {
			log.Info("token validation failed", "method", validationMethod, "error", err.Error())
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid token",
			})
		}

		if !tokenInfo.Valid {
			log.Info("token is not active", "method", validationMethod)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Token is not active",
			})
		}

		// Store auth info in context
		authInfo := &AuthInfo{
			UserID:    tokenInfo.UserID,
			Email:     tokenInfo.Email,
			Name:      tokenInfo.Name,
			Roles:     tokenInfo.Roles,
			ProjectID: tokenInfo.ProjectID,
		}

		// Add to Fiber context
		c.Locals(string(AuthInfoKey), authInfo)
		c.Locals(string(UserIDKey), tokenInfo.UserID)

		// Fetch user from database using OIDC User ID
		user, err := m.userRepo.GetByOIDCUserID(c.Context(), tokenInfo.UserID)
		if err != nil {
			log.Info(
				"user not found in database",
				"oidcUserID",
				tokenInfo.UserID,
				"error",
				err.Error(),
			)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "User not found",
			})
		}


		// Store user in Fiber context
		c.Locals(UserKeyFiber, user)

		// Add to Go context for services
		ctx := context.WithValue(c.Context(), AuthInfoKey, authInfo)
		ctx = context.WithValue(ctx, UserIDKey, tokenInfo.UserID)
		ctx = context.WithValue(ctx, UserKey, user)
		c.SetUserContext(ctx)

		log.Info(
			"user authenticated",
			"method",
			validationMethod,
			"userID",
			tokenInfo.UserID,
			"email",
			tokenInfo.Email,
			"dbUserID",
			user.ID,
		)
		return c.Next()
	}
}

// OptionalAuth middleware validates OIDC tokens but doesn't require authentication
func (m *Middleware) OptionalAuth(zitadelService *services.ZitadelService) fiber.Handler {
	log := m.log.Function("OptionalAuth")

	return func(c *fiber.Ctx) error {
		// Skip auth if Zitadel is not configured
		if !zitadelService.IsConfigured() {
			return c.Next()
		}

		// Extract token from Authorization header
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Next() // No token, continue without auth
		}

		// Check for Bearer token format
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || strings.ToLower(tokenParts[0]) != "bearer" {
			return c.Next() // Invalid format, continue without auth
		}

		token := tokenParts[1]
		if token == "" {
			return c.Next() // No token, continue without auth
		}

		// Validate token with Zitadel - try JWT first, fallback to introspection
		var tokenInfo *services.TokenInfo
		var err error
		var validationMethod string

		if isJWTToken(token) {
			// Try JWT validation first (local, fast)
			tokenInfo, err = zitadelService.ValidateIDToken(c.Context(), token)
			validationMethod = "JWT"
			
			// If JWT validation fails due to token format/type, fallback to introspection
			if err != nil {
				log.Debug("JWT validation failed, falling back to introspection", "error", err.Error())
				tokenInfo, err = zitadelService.ValidateToken(c.Context(), token)
				validationMethod = "introspection_fallback"
			}
		} else {
			// Not a JWT token, use introspection directly
			tokenInfo, err = zitadelService.ValidateToken(c.Context(), token)
			validationMethod = "introspection"
		}

		if err != nil || !tokenInfo.Valid {
			log.Debug("optional auth token validation failed", "method", validationMethod, "error", err)
			return c.Next() // Invalid token, continue without auth
		}

		// Store auth info in context
		authInfo := &AuthInfo{
			UserID:    tokenInfo.UserID,
			Email:     tokenInfo.Email,
			Name:      tokenInfo.Name,
			Roles:     tokenInfo.Roles,
			ProjectID: tokenInfo.ProjectID,
		}

		// Add to Fiber context
		c.Locals(string(AuthInfoKey), authInfo)
		c.Locals(string(UserIDKey), tokenInfo.UserID)

		// Try to fetch user from database using OIDC User ID
		user, err := m.userRepo.GetByOIDCUserID(c.Context(), tokenInfo.UserID)
		if err != nil {
			log.Debug(
				"user not found in database for optional auth",
				"oidcUserID",
				tokenInfo.UserID,
				"error",
				err.Error(),
			)
			// Continue without user for optional auth
		} else {
			// Store user in Fiber context if found
			c.Locals(UserKeyFiber, user)
		}

		// Add to Go context for services
		ctx := context.WithValue(c.Context(), AuthInfoKey, authInfo)
		ctx = context.WithValue(ctx, UserIDKey, tokenInfo.UserID)
		if user != nil {
			ctx = context.WithValue(ctx, UserKey, user)
		}
		c.SetUserContext(ctx)

		log.Info("optional auth successful", "method", validationMethod, "userID", tokenInfo.UserID, "email", tokenInfo.Email)
		return c.Next()
	}
}

// RequireRole middleware checks if the authenticated user has a specific role
func (m *Middleware) RequireRole(role string) fiber.Handler {
	log := m.log.Function("RequireRole")

	return func(c *fiber.Ctx) error {
		authInfo := GetAuthInfo(c)
		if authInfo == nil {
			log.Info("no auth info found for role check", "requiredRole", role)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Authentication required",
			})
		}

		// Check if user has the required role
		if slices.Contains(authInfo.Roles, role) {
			log.Info("role check passed", "userID", authInfo.UserID, "role", role)
			return c.Next()
		}

		log.Info(
			"insufficient permissions",
			"userID",
			authInfo.UserID,
			"requiredRole",
			role,
			"userRoles",
			authInfo.Roles,
		)
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Insufficient permissions",
		})
	}
}

// GetAuthInfo extracts auth info from Fiber context
func GetAuthInfo(c *fiber.Ctx) *AuthInfo {
	authInfo, ok := c.Locals(string(AuthInfoKey)).(*AuthInfo)
	if !ok {
		return nil
	}
	return authInfo
}

// GetUserID extracts user ID from Fiber context
func GetUserID(c *fiber.Ctx) string {
	userID, ok := c.Locals(string(UserIDKey)).(string)
	if !ok {
		return ""
	}
	return userID
}

// GetAuthInfoFromContext extracts auth info from Go context
func GetAuthInfoFromContext(ctx context.Context) *AuthInfo {
	authInfo, ok := ctx.Value(AuthInfoKey).(*AuthInfo)
	if !ok {
		return nil
	}
	return authInfo
}

// GetUserIDFromContext extracts user ID from Go context
func GetUserIDFromContext(ctx context.Context) string {
	userID, ok := ctx.Value(UserIDKey).(string)
	if !ok {
		return ""
	}
	return userID
}

// GetUser extracts user from Fiber context
func GetUser(c *fiber.Ctx) *models.User {
	user, ok := c.Locals(UserKeyFiber).(*models.User)
	if !ok {
		return nil
	}
	return user
}

// GetUserFromContext extracts user from Go context
func GetUserFromContext(ctx context.Context) *models.User {
	user, ok := ctx.Value(UserKey).(*models.User)
	if !ok {
		return nil
	}
	return user
}

// isJWTToken checks if a token has JWT structure (3 base64 segments separated by dots)
func isJWTToken(token string) bool {
	parts := strings.Split(token, ".")
	return len(parts) == 3 &&
		len(parts[0]) > 0 &&
		len(parts[1]) > 0 &&
		len(parts[2]) > 0
}

