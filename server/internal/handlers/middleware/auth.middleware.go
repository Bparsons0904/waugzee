package middleware

import (
	"context"
	"strings"
	"waugzee/internal/models"
	"waugzee/internal/services"

	"github.com/gofiber/fiber/v2"
)

// AuthContextKey is used to store auth info in context
type AuthContextKey string

const (
	UserKey      AuthContextKey = "user"
	UserKeyFiber string         = "User" // Fiber context key (string)
)


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

		// Validate token with Zitadel using hybrid approach
		tokenInfo, validationMethod, err := zitadelService.ValidateTokenWithFallback(c.Context(), token)

		if err != nil {
			log.Info("token validation failed", "method", validationMethod, "error", err.Error())
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid token",
			})
		}


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
		ctx := context.WithValue(c.Context(), UserKey, user)
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

// GetUser extracts user from Fiber context
func GetUser(c *fiber.Ctx) *models.User {
	user, ok := c.Locals(UserKeyFiber).(*models.User)
	if !ok {
		return nil
	}
	return user
}


