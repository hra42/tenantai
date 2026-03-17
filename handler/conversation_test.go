package handler

import (
	"context"
	"encoding/json"
	"fmt"
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

// newTestServiceManager creates a real DefaultServiceManager backed by a temp dir.
func newTestServiceManager(t *testing.T) *service.DefaultServiceManager {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "services")
	mgr, err := service.NewServiceManager(dir, 1)
	if err != nil {
		t.Fatalf("NewServiceManager: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })
	return mgr
}

// insertConversations inserts n conversation rows into the given service's DB.
func insertConversations(t *testing.T, mgr *service.DefaultServiceManager, serviceID string, n int, sessionID string) {
	t.Helper()
	db, err := mgr.GetDBConnection(context.Background(), serviceID)
	if err != nil {
		t.Fatalf("GetDBConnection: %v", err)
	}
	for i := 0; i < n; i++ {
		sid := sessionID
		_, err := db.Exec(
			"INSERT INTO conversations (id, session_id, model, messages, finish_reason) VALUES (?, ?, ?, ?, ?)",
			fmt.Sprintf("conv-%d", i), sid, "gpt-4", `{"request":[],"response":[]}`, "stop",
		)
		if err != nil {
			t.Fatalf("insert conversation %d: %v", i, err)
		}
	}
}

func setupConversationApp(t *testing.T, mgr *service.DefaultServiceManager) *fiber.App {
	t.Helper()
	handler := NewConversationHandler(mgr)
	app := fiber.New(fiber.Config{ErrorHandler: middleware.ErrorHandler})
	app.Get("/services/:id/conversations", handler.HandleList)
	return app
}

func TestConversationHandler_DefaultPagination(t *testing.T) {
	mgr := newTestServiceManager(t)
	ctx := context.Background()
	_, _ = mgr.Create(ctx, "conv-test", "Conv Test")
	insertConversations(t, mgr, "conv-test", 5, "sess-1")

	app := setupConversationApp(t, mgr)

	req := httptest.NewRequest(http.MethodGet, "/services/conv-test/conversations", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want 200, body: %s", resp.StatusCode, body)
	}

	body, _ := io.ReadAll(resp.Body)
	var result models.ConversationListResponse
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result.Total != 5 {
		t.Errorf("total = %d, want 5", result.Total)
	}
	if result.Limit != 100 {
		t.Errorf("limit = %d, want 100 (default)", result.Limit)
	}
	if result.Offset != 0 {
		t.Errorf("offset = %d, want 0 (default)", result.Offset)
	}
	if len(result.Data) != 5 {
		t.Errorf("data len = %d, want 5", len(result.Data))
	}
}

func TestConversationHandler_CustomLimitOffset(t *testing.T) {
	mgr := newTestServiceManager(t)
	ctx := context.Background()
	_, _ = mgr.Create(ctx, "pag-test", "Pagination Test")
	insertConversations(t, mgr, "pag-test", 10, "sess-1")

	app := setupConversationApp(t, mgr)

	req := httptest.NewRequest(http.MethodGet, "/services/pag-test/conversations?limit=3&offset=2", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}

	body, _ := io.ReadAll(resp.Body)
	var result models.ConversationListResponse
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result.Total != 10 {
		t.Errorf("total = %d, want 10", result.Total)
	}
	if result.Limit != 3 {
		t.Errorf("limit = %d, want 3", result.Limit)
	}
	if result.Offset != 2 {
		t.Errorf("offset = %d, want 2", result.Offset)
	}
	if len(result.Data) != 3 {
		t.Errorf("data len = %d, want 3", len(result.Data))
	}
}

func TestConversationHandler_SessionIDFilter(t *testing.T) {
	mgr := newTestServiceManager(t)
	ctx := context.Background()
	_, _ = mgr.Create(ctx, "sess-test", "Session Test")
	insertConversations(t, mgr, "sess-test", 3, "session-a")

	// Insert some with a different session
	db, _ := mgr.GetDBConnection(ctx, "sess-test")
	for i := 0; i < 2; i++ {
		_, _ = db.Exec(
			"INSERT INTO conversations (id, session_id, model, messages, finish_reason) VALUES (?, ?, ?, ?, ?)",
			fmt.Sprintf("other-%d", i), "session-b", "gpt-4", `{}`, "stop",
		)
	}

	app := setupConversationApp(t, mgr)

	req := httptest.NewRequest(http.MethodGet, "/services/sess-test/conversations?session_id=session-a", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}

	body, _ := io.ReadAll(resp.Body)
	var result models.ConversationListResponse
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result.Total != 3 {
		t.Errorf("total = %d, want 3 (session-a only)", result.Total)
	}
}

func TestConversationHandler_SortParameter(t *testing.T) {
	mgr := newTestServiceManager(t)
	ctx := context.Background()
	_, _ = mgr.Create(ctx, "sort-test", "Sort Test")
	insertConversations(t, mgr, "sort-test", 3, "sess-1")

	app := setupConversationApp(t, mgr)

	// sort=updated_at should not error
	req := httptest.NewRequest(http.MethodGet, "/services/sort-test/conversations?sort=updated_at", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want 200, body: %s", resp.StatusCode, body)
	}

	// sort=invalid should fall back to created_at (no error)
	req = httptest.NewRequest(http.MethodGet, "/services/sort-test/conversations?sort=invalid", nil)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200 for invalid sort (falls back to created_at)", resp.StatusCode)
	}
}

func TestConversationHandler_LimitCappedAt1000(t *testing.T) {
	mgr := newTestServiceManager(t)
	ctx := context.Background()
	_, _ = mgr.Create(ctx, "cap-test", "Cap Test")
	insertConversations(t, mgr, "cap-test", 2, "sess-1")

	app := setupConversationApp(t, mgr)

	req := httptest.NewRequest(http.MethodGet, "/services/cap-test/conversations?limit=5000", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}

	body, _ := io.ReadAll(resp.Body)
	var result models.ConversationListResponse
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result.Limit != 1000 {
		t.Errorf("limit = %d, want 1000 (capped)", result.Limit)
	}
}

func TestConversationHandler_NonexistentService(t *testing.T) {
	mgr := newTestServiceManager(t)
	app := setupConversationApp(t, mgr)

	req := httptest.NewRequest(http.MethodGet, "/services/nonexistent/conversations", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}
