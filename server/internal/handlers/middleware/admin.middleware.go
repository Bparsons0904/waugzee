package middleware

import (
	"github.com/gofiber/fiber/v2"
)

func (m *Middleware) RequireAdmin() fiber.Handler {
	log := m.log.Function("RequireAdmin")

	return func(c *fiber.Ctx) error {
		user := GetUser(c)
		if user == nil {
			log.Info("user not found in context")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Authentication required",
			})
		}

		if !user.IsAdmin {
			log.Info("user is not admin", "userID", user.ID, "email", user.Email)
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Admin access required",
			})
		}

		return c.Next()
	}
}
