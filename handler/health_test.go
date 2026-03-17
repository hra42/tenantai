package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/hra42/tenantai/service"
)

type mockServiceManager struct {
	services []*service.Service
	dbs      map[string]*sql.DB
	listErr  error
	dbErr    error
}

func (m *mockServiceManager) Create(_ context.Context, id, name string) (*service.Service, error) {
	return nil, nil
}

func (m *mockServiceManager) Get(_ context.Context, id string) (*service.Service, error) {
	return nil, nil
}

func (m *mockServiceManager) List(_ context.Context) ([]*service.Service, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.services, nil
}

func (m *mockServiceManager) Delete(_ context.Context, id string) error {
	return nil
}

func (m *mockServiceManager) GetDBConnection(_ context.Context, id string) (*sql.DB, error) {
	if m.dbErr != nil {
		return nil, m.dbErr
	}
	if db, ok := m.dbs[id]; ok {
		return db, nil
	}
	return nil, errors.New("db not found")
}

func (m *mockServiceManager) Close() error {
	return nil
}

func TestHandleHealth(t *testing.T) {
	h := NewHealthHandler(&mockServiceManager{})
	app := fiber.New()
	app.Get("/health", h.HandleHealth)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result["status"] != "ok" {
		t.Errorf("status = %q, want ok", result["status"])
	}
}

func TestHandleReady_NoServices(t *testing.T) {
	h := NewHealthHandler(&mockServiceManager{})
	app := fiber.New()
	app.Get("/ready", h.HandleReady)

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result["status"] != "ready" {
		t.Errorf("status = %q, want ready", result["status"])
	}
}

func TestHandleReady_ListError(t *testing.T) {
	h := NewHealthHandler(&mockServiceManager{
		listErr: errors.New("db broken"),
	})
	app := fiber.New()
	app.Get("/ready", h.HandleReady)

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusServiceUnavailable)
	}
}

func TestHandleReady_DBConnectionError(t *testing.T) {
	h := NewHealthHandler(&mockServiceManager{
		services: []*service.Service{{ID: "svc1", Name: "Test"}},
		dbErr:    errors.New("connection refused"),
	})
	app := fiber.New()
	app.Get("/ready", h.HandleReady)

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusServiceUnavailable)
	}
}
