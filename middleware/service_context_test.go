package middleware

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/hra42/tenantai/models"
	"github.com/hra42/tenantai/service"
)

// mockServiceManager implements service.ServiceManager for testing.
type mockServiceManager struct {
	services map[string]*service.Service
	dbs      map[string]*sql.DB
}

func newMockServiceManager() *mockServiceManager {
	return &mockServiceManager{
		services: make(map[string]*service.Service),
		dbs:      make(map[string]*sql.DB),
	}
}

func (m *mockServiceManager) Create(_ context.Context, id, name string) (*service.Service, error) {
	return nil, nil
}

func (m *mockServiceManager) Get(_ context.Context, id string) (*service.Service, error) {
	svc, ok := m.services[id]
	if !ok {
		return nil, service.ErrServiceNotFound
	}
	return svc, nil
}

func (m *mockServiceManager) List(_ context.Context) ([]*service.Service, error) {
	return nil, nil
}

func (m *mockServiceManager) Delete(_ context.Context, id string) error {
	return nil
}

func (m *mockServiceManager) GetDBConnection(_ context.Context, id string) (*sql.DB, error) {
	db, ok := m.dbs[id]
	if !ok {
		return nil, service.ErrServiceNotFound
	}
	return db, nil
}

func (m *mockServiceManager) Close() error {
	return nil
}

func (m *mockServiceManager) addService(svc *service.Service, db *sql.DB) {
	m.services[svc.ID] = svc
	m.dbs[svc.ID] = db
}

func TestServiceContext_ValidServiceID(t *testing.T) {
	mgr := newMockServiceManager()

	// Use a real in-memory DuckDB for the DB value
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("open in-memory duckdb: %v", err)
	}
	defer func() { _ = db.Close() }()

	svc := &service.Service{ID: "test-svc", Name: "Test"}
	mgr.addService(svc, db)

	app := fiber.New(fiber.Config{ErrorHandler: ErrorHandler})
	app.Use(ServiceContext(mgr))
	app.Get("/test", func(c fiber.Ctx) error {
		gotSvc, err := GetServiceFromContext(c)
		if err != nil {
			return NewInternalError(err.Error())
		}
		gotDB, err := GetDBFromContext(c)
		if err != nil {
			return NewInternalError(err.Error())
		}
		if gotSvc.ID != "test-svc" {
			t.Errorf("service ID = %q, want test-svc", gotSvc.ID)
		}
		if gotDB == nil {
			t.Error("expected non-nil DB from context")
		}
		return c.SendString("ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Service-ID", "test-svc")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestServiceContext_MissingHeader(t *testing.T) {
	mgr := newMockServiceManager()

	app := fiber.New(fiber.Config{ErrorHandler: ErrorHandler})
	app.Use(ServiceContext(mgr))
	app.Get("/test", func(c fiber.Ctx) error {
		return c.SendString("should not reach")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	// No X-Service-ID header
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}

	body, _ := io.ReadAll(resp.Body)
	var apiErr models.APIError
	if err := json.Unmarshal(body, &apiErr); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if apiErr.Error.Code != CodeMissingHeader {
		t.Errorf("code = %q, want %q", apiErr.Error.Code, CodeMissingHeader)
	}
}

func TestServiceContext_UnknownService(t *testing.T) {
	mgr := newMockServiceManager()

	app := fiber.New(fiber.Config{ErrorHandler: ErrorHandler})
	app.Use(ServiceContext(mgr))
	app.Get("/test", func(c fiber.Ctx) error {
		return c.SendString("should not reach")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Service-ID", "nonexistent")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}

	body, _ := io.ReadAll(resp.Body)
	var apiErr models.APIError
	if err := json.Unmarshal(body, &apiErr); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if apiErr.Error.Code != CodeServiceNotFound {
		t.Errorf("code = %q, want %q", apiErr.Error.Code, CodeServiceNotFound)
	}
}
