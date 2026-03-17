package middleware

import "github.com/gofiber/fiber/v3"

// CORS returns middleware that sets permissive CORS headers.
func CORS() fiber.Handler {
	return func(c fiber.Ctx) error {
		c.Set("Access-Control-Allow-Origin", "*")
		c.Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		c.Set("Access-Control-Allow-Headers", "Content-Type, X-Service-ID, X-Session-ID, Authorization")

		if c.Method() == fiber.MethodOptions {
			return c.SendStatus(fiber.StatusNoContent)
		}
		return c.Next()
	}
}
