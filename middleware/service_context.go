package middleware

import (
	"database/sql"
	"fmt"

	"github.com/gofiber/fiber/v3"
	"github.com/hra42/tenantai/service"
)

const (
	localsKeyService = "service"
	localsKeyDB      = "serviceDB"
)

// dbRef wraps *sql.DB without implementing io.Closer, preventing
// Fiber v3 from closing the connection when request locals are cleaned up.
type dbRef struct {
	DB *sql.DB
}

// ServiceContext returns middleware that extracts X-Service-ID and loads
// the service and its DB connection into the request context.
func ServiceContext(mgr service.ServiceManager) fiber.Handler {
	return func(c fiber.Ctx) error {
		serviceID := c.Get("X-Service-ID")
		if serviceID == "" {
			return NewMissingHeaderError("X-Service-ID")
		}

		svc, err := mgr.Get(c.Context(), serviceID)
		if err != nil {
			return err
		}

		db, err := mgr.GetDBConnection(c.Context(), serviceID)
		if err != nil {
			return err
		}

		c.Locals(localsKeyService, svc)
		c.Locals(localsKeyDB, &dbRef{DB: db})

		return c.Next()
	}
}

// GetServiceFromContext retrieves the service stored by ServiceContext middleware.
func GetServiceFromContext(c fiber.Ctx) (*service.Service, error) {
	svc, ok := c.Locals(localsKeyService).(*service.Service)
	if !ok || svc == nil {
		return nil, fmt.Errorf("service not found in context")
	}
	return svc, nil
}

// SetDBInContext stores a DB connection in the request context.
// Use this instead of c.Locals directly to avoid Fiber closing the connection.
func SetDBInContext(c fiber.Ctx, db *sql.DB) {
	c.Locals(localsKeyDB, &dbRef{DB: db})
}

// GetDBFromContext retrieves the DB connection stored by ServiceContext middleware.
func GetDBFromContext(c fiber.Ctx) (*sql.DB, error) {
	ref, ok := c.Locals(localsKeyDB).(*dbRef)
	if !ok || ref == nil {
		return nil, fmt.Errorf("database connection not found in context")
	}
	return ref.DB, nil
}
