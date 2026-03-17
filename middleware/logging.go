package middleware

import (
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v3"
)

// RequestLogger returns middleware that logs every request with structured fields.
func RequestLogger() fiber.Handler {
	return func(c fiber.Ctx) error {
		start := time.Now()

		err := c.Next()

		duration := time.Since(start)
		status := c.Response().StatusCode()

		attrs := []slog.Attr{
			slog.String("method", c.Method()),
			slog.String("path", c.Path()),
			slog.Int("status", status),
			slog.Duration("duration", duration),
			slog.String("ip", c.IP()),
		}

		if serviceID := c.Get("X-Service-ID"); serviceID != "" {
			attrs = append(attrs, slog.String("service_id", serviceID))
		}

		msg := "request"
		switch {
		case status >= 500:
			slog.LogAttrs(c.Context(), slog.LevelError, msg, attrs...)
		case status >= 400:
			slog.LogAttrs(c.Context(), slog.LevelWarn, msg, attrs...)
		default:
			slog.LogAttrs(c.Context(), slog.LevelInfo, msg, attrs...)
		}

		return err
	}
}
