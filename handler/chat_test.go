package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	or "github.com/hra42/openrouter-go"

	"github.com/gofiber/fiber/v3"
	"github.com/hra42/tenantai/database"
	"github.com/hra42/tenantai/middleware"
	"github.com/hra42/tenantai/models"
	"github.com/hra42/tenantai/service"
)

// chatMockCompleter implements openrouter.ChatCompleter for unit testing.
type chatMockCompleter struct {
	response *models.ChatCompletionResponse
	err      error
}

func (m *chatMockCompleter) ChatComplete(_ context.Context, _ *models.ChatCompletionRequest) (*models.ChatCompletionResponse, error) {
	return m.response, m.err
}

func (m *chatMockCompleter) ChatCompleteStream(_ context.Context, _ *models.ChatCompletionRequest) (*or.ChatStream, error) {
	return nil, m.err
}

// newTestDB opens an in-memory DuckDB and initializes the schema.
func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := database.OpenDB("", 1)
	if err != nil {
		t.Fatalf("open in-memory duckdb: %v", err)
	}
	if err := database.InitializeSchema(db); err != nil {
		t.Fatalf("init schema: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// setupChatApp creates a Fiber app with the chat handler wired up, including
// middleware that injects a service and DB into context.
func setupChatApp(t *testing.T, mock *chatMockCompleter) (*fiber.App, *ConversationLogger) {
	t.Helper()
	db := newTestDB(t)
	logger := NewConversationLogger(64)
	t.Cleanup(func() { logger.Close() })

	handler := NewChatHandler(mock, logger)

	svc := &service.Service{ID: "test-svc", Name: "Test"}

	app := fiber.New(fiber.Config{ErrorHandler: middleware.ErrorHandler})
	// Manually inject service and DB into context (simulating the middleware).
	app.Use(func(c fiber.Ctx) error {
		c.Locals("service", svc)
		middleware.SetDBInContext(c, db)
		return c.Next()
	})
	app.Post("/v1/chat/completions", handler.HandleChatCompletion)

	return app, logger
}

func TestChatHandler_ValidRequest(t *testing.T) {
	mock := &chatMockCompleter{
		response: &models.ChatCompletionResponse{
			ID:    "chatcmpl-123",
			Model: "openai/gpt-4",
			Choices: []models.Choice{
				{Index: 0, Message: models.ChatMessage{Role: "assistant", Content: "Hello!"}, FinishReason: "stop"},
			},
		},
	}

	app, _ := setupChatApp(t, mock)

	reqBody := models.ChatCompletionRequest{
		Model:    "openai/gpt-4",
		Messages: []models.ChatMessage{{Role: "user", Content: "Hi"}},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want %d, body: %s", resp.StatusCode, http.StatusOK, respBody)
	}

	var chatResp models.ChatCompletionResponse
	respBody, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if chatResp.ID != "chatcmpl-123" {
		t.Errorf("response ID = %q, want chatcmpl-123", chatResp.ID)
	}
	if len(chatResp.Choices) != 1 {
		t.Fatalf("choices len = %d, want 1", len(chatResp.Choices))
	}
	if chatResp.Choices[0].Message.Content != "Hello!" {
		t.Errorf("content = %q, want Hello!", chatResp.Choices[0].Message.Content)
	}
}

func TestChatHandler_MissingModel(t *testing.T) {
	mock := &chatMockCompleter{}
	app, _ := setupChatApp(t, mock)

	reqBody := models.ChatCompletionRequest{
		Messages: []models.ChatMessage{{Role: "user", Content: "Hi"}},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
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

func TestChatHandler_EmptyMessages(t *testing.T) {
	mock := &chatMockCompleter{}
	app, _ := setupChatApp(t, mock)

	reqBody := models.ChatCompletionRequest{
		Model:    "openai/gpt-4",
		Messages: []models.ChatMessage{},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestChatHandler_MissingServiceInContext(t *testing.T) {
	mock := &chatMockCompleter{
		response: &models.ChatCompletionResponse{ID: "test"},
	}
	logger := NewConversationLogger(8)
	defer logger.Close()
	handler := NewChatHandler(mock, logger)

	// App WITHOUT middleware injecting service/DB into context
	app := fiber.New(fiber.Config{ErrorHandler: middleware.ErrorHandler})
	app.Post("/v1/chat/completions", handler.HandleChatCompletion)

	reqBody := models.ChatCompletionRequest{
		Model:    "openai/gpt-4",
		Messages: []models.ChatMessage{{Role: "user", Content: "Hi"}},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusInternalServerError)
	}
}

func TestChatHandler_ConversationLogged(t *testing.T) {
	mock := &chatMockCompleter{
		response: &models.ChatCompletionResponse{
			ID:    "chatcmpl-logged",
			Model: "openai/gpt-4",
			Choices: []models.Choice{
				{Index: 0, Message: models.ChatMessage{Role: "assistant", Content: "World"}, FinishReason: "stop"},
			},
		},
	}

	db, err := database.OpenDB("", 1)
	if err != nil {
		t.Fatalf("open in-memory duckdb: %v", err)
	}
	if err := database.InitializeSchema(db); err != nil {
		t.Fatalf("init schema: %v", err)
	}

	logger := NewConversationLogger(64)

	svc := &service.Service{ID: "log-svc", Name: "Log Test"}

	h := NewChatHandler(mock, logger)

	app := fiber.New(fiber.Config{ErrorHandler: middleware.ErrorHandler})
	app.Use(func(c fiber.Ctx) error {
		c.Locals("service", svc)
		middleware.SetDBInContext(c, db)
		return c.Next()
	})
	app.Post("/v1/chat/completions", h.HandleChatCompletion)

	reqBody := models.ChatCompletionRequest{
		Model:    "openai/gpt-4",
		Messages: []models.ChatMessage{{Role: "user", Content: "Hello"}},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	// Close logger to drain the buffer — blocks until worker finishes all writes
	logger.Close()

	// Verify conversation was logged
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM conversations").Scan(&count); err != nil {
		t.Fatalf("count query: %v", err)
	}
	if count != 1 {
		t.Errorf("conversation count = %d, want 1", count)
	}

	_ = db.Close()
}
