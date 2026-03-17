package handler

import (
	"fmt"

	"github.com/gofiber/fiber/v3"
	"github.com/hra42/tenantai/service"
)

// HealthHandler handles health and readiness probes.
type HealthHandler struct {
	mgr service.ServiceManager
}

// NewHealthHandler creates a new HealthHandler.
func NewHealthHandler(mgr service.ServiceManager) *HealthHandler {
	return &HealthHandler{mgr: mgr}
}

// HandleHealth is a liveness probe — always returns 200 if the server is running.
func (h *HealthHandler) HandleHealth(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "ok"})
}

// HandleReady is a readiness probe — checks that all service DBs are reachable.
func (h *HealthHandler) HandleReady(c fiber.Ctx) error {
	services, err := h.mgr.List(c.Context())
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"status": "not ready",
			"error":  fmt.Sprintf("failed to list services: %v", err),
		})
	}

	for _, svc := range services {
		db, err := h.mgr.GetDBConnection(c.Context(), svc.ID)
		if err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"status": "not ready",
				"error":  fmt.Sprintf("service %q DB unavailable: %v", svc.ID, err),
			})
		}
		if err := db.PingContext(c.Context()); err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"status": "not ready",
				"error":  fmt.Sprintf("service %q DB ping failed: %v", svc.ID, err),
			})
		}
	}

	return c.JSON(fiber.Map{
		"status":   "ready",
		"services": len(services),
	})
}
