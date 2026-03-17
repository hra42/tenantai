package handler

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	or "github.com/hra42/openrouter-go"
)

type mockModelLister struct {
	resp *or.ModelsResponse
	err  error
}

func (m *mockModelLister) ListModels(_ context.Context) (*or.ModelsResponse, error) {
	return m.resp, m.err
}

func TestModelsHandler_HandleList(t *testing.T) {
	mock := &mockModelLister{
		resp: &or.ModelsResponse{
			Data: []or.Model{
				{ID: "openai/gpt-4o", Name: "GPT-4o"},
				{ID: "anthropic/claude-3.5-sonnet", Name: "Claude 3.5 Sonnet"},
			},
		},
	}

	app := fiber.New()
	h := NewModelsHandler(mock)
	app.Get("/v1/models", h.HandleList)

	req := httptest.NewRequest("GET", "/v1/models", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestModelsHandler_HandleList_Error(t *testing.T) {
	mock := &mockModelLister{
		err: fiber.NewError(502, "upstream error"),
	}

	app := fiber.New(fiber.Config{
		ErrorHandler: func(c fiber.Ctx, err error) error {
			if e, ok := err.(*fiber.Error); ok {
				return c.Status(e.Code).JSON(fiber.Map{"error": e.Message})
			}
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		},
	})
	h := NewModelsHandler(mock)
	app.Get("/v1/models", h.HandleList)

	req := httptest.NewRequest("GET", "/v1/models", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 502 {
		t.Errorf("expected status 502, got %d", resp.StatusCode)
	}
}
