package handler

import (
	"os"
	"path/filepath"

	"github.com/gofiber/fiber/v3"

	"github.com/hra42/tenantai/middleware"
	"github.com/hra42/tenantai/models"
	"github.com/hra42/tenantai/service"
)

// ServiceHandler handles service CRUD operations.
type ServiceHandler struct {
	mgr         service.ServiceManager
	servicesDir string
}

// NewServiceHandler creates a new ServiceHandler.
func NewServiceHandler(mgr service.ServiceManager, servicesDir string) *ServiceHandler {
	return &ServiceHandler{mgr: mgr, servicesDir: servicesDir}
}

// HandleCreate creates a new service.
func (h *ServiceHandler) HandleCreate(c fiber.Ctx) error {
	var req models.CreateServiceRequest
	if err := c.Bind().JSON(&req); err != nil {
		return middleware.NewInvalidRequestError("invalid request body")
	}

	if req.Name == "" {
		return middleware.NewInvalidRequestError("name is required")
	}
	if req.ID == "" {
		return middleware.NewInvalidRequestError("id is required")
	}
	if err := service.ValidateServiceID(req.ID); err != nil {
		return middleware.NewInvalidRequestError(err.Error())
	}

	svc, err := h.mgr.Create(c.Context(), req.ID, req.Name)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(models.ServiceResponse{
		ID:        svc.ID,
		Name:      svc.Name,
		CreatedAt: svc.CreatedAt.Format("2006-01-02T15:04:05Z"),
	})
}

// HandleList returns all services.
func (h *ServiceHandler) HandleList(c fiber.Ctx) error {
	services, err := h.mgr.List(c.Context())
	if err != nil {
		return err
	}

	resp := make([]models.ServiceResponse, len(services))
	for i, svc := range services {
		resp[i] = models.ServiceResponse{
			ID:        svc.ID,
			Name:      svc.Name,
			CreatedAt: svc.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	return c.JSON(resp)
}

// HandleGet returns a service with details.
func (h *ServiceHandler) HandleGet(c fiber.Ctx) error {
	id := c.Params("id")

	svc, err := h.mgr.Get(c.Context(), id)
	if err != nil {
		return err
	}

	// Get DB file size
	var fileSize int64
	dbPath := filepath.Join(h.servicesDir, id+".db")
	if info, err := os.Stat(dbPath); err == nil {
		fileSize = info.Size()
	}

	// Get conversation count
	var convCount int
	db, err := h.mgr.GetDBConnection(c.Context(), id)
	if err == nil {
		_ = db.QueryRow("SELECT COUNT(*) FROM conversations").Scan(&convCount)
	}

	return c.JSON(models.ServiceDetailResponse{
		ServiceResponse: models.ServiceResponse{
			ID:        svc.ID,
			Name:      svc.Name,
			CreatedAt: svc.CreatedAt.Format("2006-01-02T15:04:05Z"),
		},
		DBFileSizeBytes:   fileSize,
		ConversationCount: convCount,
	})
}

// HandleDelete deletes a service.
func (h *ServiceHandler) HandleDelete(c fiber.Ctx) error {
	id := c.Params("id")

	if err := h.mgr.Delete(c.Context(), id); err != nil {
		return err
	}

	return c.SendStatus(fiber.StatusNoContent)
}
