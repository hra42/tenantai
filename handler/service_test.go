package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/hra42/tenantai/middleware"
	"github.com/hra42/tenantai/models"
	"github.com/hra42/tenantai/service"
)

func setupServiceApp(t *testing.T) (*fiber.App, *service.DefaultServiceManager) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "services")
	mgr, err := service.NewServiceManager(dir, 1)
	if err != nil {
		t.Fatalf("NewServiceManager: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	handler := NewServiceHandler(mgr, dir)
	app := fiber.New(fiber.Config{ErrorHandler: middleware.ErrorHandler})
	app.Post("/services", handler.HandleCreate)
	app.Get("/services", handler.HandleList)
	app.Get("/services/:id", handler.HandleGet)
	app.Delete("/services/:id", handler.HandleDelete)

	return app, mgr
}

func TestServiceHandler_CreateValid(t *testing.T) {
	app, _ := setupServiceApp(t)

	reqBody := models.CreateServiceRequest{ID: "my-svc", Name: "My Service"}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/services", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want 201, body: %s", resp.StatusCode, respBody)
	}

	respBody, _ := io.ReadAll(resp.Body)
	var svcResp models.ServiceResponse
	if err := json.Unmarshal(respBody, &svcResp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if svcResp.ID != "my-svc" {
		t.Errorf("ID = %q, want my-svc", svcResp.ID)
	}
	if svcResp.Name != "My Service" {
		t.Errorf("Name = %q, want My Service", svcResp.Name)
	}
	if svcResp.CreatedAt == "" {
		t.Error("CreatedAt should not be empty")
	}
}

func TestServiceHandler_CreateMissingName(t *testing.T) {
	app, _ := setupServiceApp(t)

	reqBody := models.CreateServiceRequest{ID: "no-name"}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/services", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}

	respBody, _ := io.ReadAll(resp.Body)
	var apiErr models.APIError
	if err := json.Unmarshal(respBody, &apiErr); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if apiErr.Error.Code != middleware.CodeInvalidRequest {
		t.Errorf("code = %q, want %q", apiErr.Error.Code, middleware.CodeInvalidRequest)
	}
}

func TestServiceHandler_CreateMissingID(t *testing.T) {
	app, _ := setupServiceApp(t)

	reqBody := models.CreateServiceRequest{Name: "No ID"}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/services", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

func TestServiceHandler_CreateInvalidID(t *testing.T) {
	app, _ := setupServiceApp(t)

	reqBody := models.CreateServiceRequest{ID: "INVALID", Name: "Bad ID"}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/services", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

func TestServiceHandler_CreateDuplicate(t *testing.T) {
	app, _ := setupServiceApp(t)

	reqBody := models.CreateServiceRequest{ID: "dup-svc", Name: "Dup"}
	body, _ := json.Marshal(reqBody)

	// First create
	req := httptest.NewRequest(http.MethodPost, "/services", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("first create status = %d, want 201", resp.StatusCode)
	}

	// Second create (duplicate)
	body, _ = json.Marshal(reqBody)
	req = httptest.NewRequest(http.MethodPost, "/services", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("duplicate create status = %d, want 409", resp.StatusCode)
	}
}

func TestServiceHandler_List(t *testing.T) {
	app, mgr := setupServiceApp(t)
	ctx := context.Background()
	_, _ = mgr.Create(ctx, "svc-a", "Service A")
	_, _ = mgr.Create(ctx, "svc-b", "Service B")

	req := httptest.NewRequest(http.MethodGet, "/services", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var services []models.ServiceResponse
	if err := json.Unmarshal(body, &services); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(services) != 2 {
		t.Errorf("len = %d, want 2", len(services))
	}
}

func TestServiceHandler_GetWithDetails(t *testing.T) {
	app, mgr := setupServiceApp(t)
	ctx := context.Background()
	_, _ = mgr.Create(ctx, "detail-svc", "Detail Service")

	req := httptest.NewRequest(http.MethodGet, "/services/detail-svc", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want 200, body: %s", resp.StatusCode, body)
	}

	body, _ := io.ReadAll(resp.Body)
	var detail models.ServiceDetailResponse
	if err := json.Unmarshal(body, &detail); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if detail.ID != "detail-svc" {
		t.Errorf("ID = %q, want detail-svc", detail.ID)
	}
	if detail.Name != "Detail Service" {
		t.Errorf("Name = %q, want Detail Service", detail.Name)
	}
	if detail.ConversationCount != 0 {
		t.Errorf("ConversationCount = %d, want 0", detail.ConversationCount)
	}
}

func TestServiceHandler_Delete(t *testing.T) {
	app, mgr := setupServiceApp(t)
	ctx := context.Background()
	_, _ = mgr.Create(ctx, "del-svc", "Delete Me")

	req := httptest.NewRequest(http.MethodDelete, "/services/del-svc", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("status = %d, want 204", resp.StatusCode)
	}

	// Verify it's gone
	_, err = mgr.Get(ctx, "del-svc")
	if err != service.ErrServiceNotFound {
		t.Errorf("expected ErrServiceNotFound after delete, got %v", err)
	}
}

func TestServiceHandler_GetNonexistent(t *testing.T) {
	app, _ := setupServiceApp(t)

	req := httptest.NewRequest(http.MethodGet, "/services/nonexistent", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}

func TestServiceHandler_DeleteNonexistent(t *testing.T) {
	app, _ := setupServiceApp(t)

	req := httptest.NewRequest(http.MethodDelete, "/services/nonexistent", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}
