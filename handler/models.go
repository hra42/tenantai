package handler

import (
	"github.com/gofiber/fiber/v3"

	"github.com/hra42/tenantai/openrouter"
)

// ModelsHandler handles model listing requests.
type ModelsHandler struct {
	lister openrouter.ModelLister
}

// NewModelsHandler creates a new ModelsHandler.
func NewModelsHandler(lister openrouter.ModelLister) *ModelsHandler {
	return &ModelsHandler{lister: lister}
}

// HandleList returns available models from OpenRouter.
func (h *ModelsHandler) HandleList(c fiber.Ctx) error {
	resp, err := h.lister.ListModels(c.Context())
	if err != nil {
		return err
	}
	return c.JSON(resp)
}
