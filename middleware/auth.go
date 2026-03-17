package middleware

import (
	"crypto/subtle"
	"strings"

	"github.com/gofiber/fiber/v3"
)

// AdminAuth returns middleware that validates a Bearer token against the configured API key.
// If apiKey is empty, auth is disabled (passthrough).
func AdminAuth(apiKey string) fiber.Handler {
	return func(c fiber.Ctx) error {
		if apiKey == "" {
			return c.Next()
		}

		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return NewUnauthorizedError("missing Authorization header")
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			return NewUnauthorizedError("malformed Authorization header, expected 'Bearer <token>'")
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if subtle.ConstantTimeCompare([]byte(token), []byte(apiKey)) != 1 {
			return NewUnauthorizedError("invalid API key")
		}

		return c.Next()
	}
}
