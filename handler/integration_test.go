package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gofiber/fiber/v3"
	or "github.com/hra42/openrouter-go"

	"github.com/hra42/tenantai/middleware"
	"github.com/hra42/tenantai/models"
	"github.com/hra42/tenantai/service"
)

// mockChatCompleter implements openrouter.ChatCompleter for testing.
type mockChatCompleter struct {
	mu        sync.Mutex
	callCount int
}

func (m *mockChatCompleter) ChatComplete(_ context.Context, req *models.ChatCompletionRequest) (*models.ChatCompletionResponse, error) {
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()

	return &models.ChatCompletionResponse{
		ID:      "chatcmpl-test-123",
		Object:  "chat.completion",
		Created: 1234567890,
		Model:   req.Model,
		Choices: []models.Choice{
			{
				Index: 0,
				Message: models.ChatMessage{
					Role:    "assistant",
					Content: "Hello! This is a mock response.",
				},
				FinishReason: "stop",
			},
		},
		Usage: &models.Usage{
			PromptTokens:     10,
			CompletionTokens: 8,
			TotalTokens:      18,
		},
	}, nil
}

func (m *mockChatCompleter) ChatCompleteStream(_ context.Context, _ *models.ChatCompletionRequest) (*or.ChatStream, error) {
	return nil, fmt.Errorf("streaming not supported in mock")
}

// setupTestApp creates a Fiber app with all routes wired up for integration testing.
func setupTestApp(t *testing.T) (*fiber.App, *service.DefaultServiceManager, *mockChatCompleter, *ConversationLogger) {
	t.Helper()

	servicesDir := t.TempDir() + "/services"
	mgr, err := service.NewServiceManager(servicesDir, 1)
	if err != nil {
		t.Fatalf("failed to create service manager: %v", err)
	}

	mockOR := &mockChatCompleter{}
	logger := NewConversationLogger(256)

	chatHandler := NewChatHandler(mockOR, logger)
	svcHandler := NewServiceHandler(mgr, servicesDir)
	convHandler := NewConversationHandler(mgr)

	app := fiber.New(fiber.Config{
		ErrorHandler: middleware.ErrorHandler,
	})

	// Service management routes
	services := app.Group("/services")
	services.Post("/", svcHandler.HandleCreate)
	services.Get("/", svcHandler.HandleList)
	services.Get("/:id", svcHandler.HandleGet)
	services.Delete("/:id", svcHandler.HandleDelete)
	services.Get("/:id/conversations", convHandler.HandleList)

	// Chat completion routes (require X-Service-ID header)
	v1 := app.Group("/v1", middleware.ServiceContext(mgr))
	v1.Post("/chat/completions", chatHandler.HandleChatCompletion)

	t.Cleanup(func() {
		_ = mgr.Close()
	})

	return app, mgr, mockOR, logger
}

// createService is a helper that creates a service via the API.
func createService(t *testing.T, app *fiber.App, id, name string) models.ServiceResponse {
	t.Helper()

	body, _ := json.Marshal(models.CreateServiceRequest{ID: id, Name: name})
	req := httptest.NewRequest(http.MethodPost, "/services", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to create service %s: %v", id, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status 201 creating service %s, got %d: %s", id, resp.StatusCode, string(respBody))
	}

	var svcResp models.ServiceResponse
	if err := json.NewDecoder(resp.Body).Decode(&svcResp); err != nil {
		t.Fatalf("failed to decode service response: %v", err)
	}

	return svcResp
}

// sendChat is a helper that sends a chat completion request via the API.
func sendChat(t *testing.T, app *fiber.App, serviceID, model, message string) models.ChatCompletionResponse {
	t.Helper()

	body, _ := json.Marshal(models.ChatCompletionRequest{
		Model: model,
		Messages: []models.ChatMessage{
			{Role: "user", Content: message},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Service-ID", serviceID)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to send chat for service %s: %v", serviceID, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status 200 for chat, got %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp models.ChatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		t.Fatalf("failed to decode chat response: %v", err)
	}

	return chatResp
}

// getConversations is a helper that retrieves conversations for a service.
func getConversations(t *testing.T, app *fiber.App, serviceID string) models.ConversationListResponse {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/services/%s/conversations", serviceID), nil)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to get conversations for service %s: %v", serviceID, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status 200 for conversations, got %d: %s", resp.StatusCode, string(respBody))
	}

	var convResp models.ConversationListResponse
	if err := json.NewDecoder(resp.Body).Decode(&convResp); err != nil {
		t.Fatalf("failed to decode conversation response: %v", err)
	}

	return convResp
}

func TestIntegration_FullE2EFlow(t *testing.T) {
	app, _, _, logger := setupTestApp(t)

	// Step 1: Create a service
	svc := createService(t, app, "test-svc", "Test Service")
	if svc.ID != "test-svc" {
		t.Errorf("expected service ID %q, got %q", "test-svc", svc.ID)
	}
	if svc.Name != "Test Service" {
		t.Errorf("expected service name %q, got %q", "Test Service", svc.Name)
	}

	// Step 2: Verify the service appears in the list
	listReq := httptest.NewRequest(http.MethodGet, "/services", nil)
	listResp, err := app.Test(listReq)
	if err != nil {
		t.Fatalf("failed to list services: %v", err)
	}
	defer func() { _ = listResp.Body.Close() }()

	var services []models.ServiceResponse
	listBody, _ := io.ReadAll(listResp.Body)
	if err := json.Unmarshal(listBody, &services); err != nil {
		t.Fatalf("failed to decode services list: %v", err)
	}
	if len(services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(services))
	}

	// Step 3: Send a chat completion request
	chatResp := sendChat(t, app, "test-svc", "gpt-4", "Hello, world!")
	if chatResp.ID != "chatcmpl-test-123" {
		t.Errorf("expected chat response ID %q, got %q", "chatcmpl-test-123", chatResp.ID)
	}
	if len(chatResp.Choices) != 1 {
		t.Fatalf("expected 1 choice, got %d", len(chatResp.Choices))
	}
	if chatResp.Choices[0].Message.Content != "Hello! This is a mock response." {
		t.Errorf("unexpected response content: %q", chatResp.Choices[0].Message.Content)
	}
	if chatResp.Choices[0].FinishReason != "stop" {
		t.Errorf("expected finish_reason %q, got %q", "stop", chatResp.Choices[0].FinishReason)
	}

	// Step 4: Close logger to drain async writes, then verify conversation was logged
	logger.Close()

	convResp := getConversations(t, app, "test-svc")
	if convResp.Total != 1 {
		t.Fatalf("expected 1 conversation, got %d", convResp.Total)
	}
	if len(convResp.Data) != 1 {
		t.Fatalf("expected 1 conversation in data, got %d", len(convResp.Data))
	}

	conv := convResp.Data[0]
	if conv.Model != "gpt-4" {
		t.Errorf("expected conversation model %q, got %q", "gpt-4", conv.Model)
	}
	if conv.FinishReason == nil || *conv.FinishReason != "stop" {
		t.Errorf("expected finish_reason %q, got %v", "stop", conv.FinishReason)
	}

	// Verify messages contain both request and response
	var messages map[string]json.RawMessage
	if err := json.Unmarshal(conv.Messages, &messages); err != nil {
		t.Fatalf("failed to unmarshal conversation messages: %v", err)
	}
	if _, ok := messages["request"]; !ok {
		t.Error("conversation messages missing 'request' key")
	}
	if _, ok := messages["response"]; !ok {
		t.Error("conversation messages missing 'response' key")
	}

	// Step 5: Get service detail and verify conversation count
	detailReq := httptest.NewRequest(http.MethodGet, "/services/test-svc", nil)
	detailResp, err := app.Test(detailReq)
	if err != nil {
		t.Fatalf("failed to get service detail: %v", err)
	}
	defer func() { _ = detailResp.Body.Close() }()

	var detail models.ServiceDetailResponse
	detailBody, _ := io.ReadAll(detailResp.Body)
	if err := json.Unmarshal(detailBody, &detail); err != nil {
		t.Fatalf("failed to decode service detail: %v", err)
	}
	if detail.ConversationCount != 1 {
		t.Errorf("expected conversation count 1, got %d", detail.ConversationCount)
	}
}

func TestIntegration_MultiServiceIsolation(t *testing.T) {
	app, _, _, logger := setupTestApp(t)

	// Create two services
	createService(t, app, "svc-alpha", "Alpha Service")
	createService(t, app, "svc-beta", "Beta Service")

	// Send chat to service alpha
	sendChat(t, app, "svc-alpha", "gpt-4", "Message for alpha")
	sendChat(t, app, "svc-alpha", "gpt-4", "Another message for alpha")

	// Send chat to service beta
	sendChat(t, app, "svc-beta", "claude-3", "Message for beta")

	// Close logger to drain async writes
	logger.Close()

	// Verify alpha has 2 conversations
	alphaConvs := getConversations(t, app, "svc-alpha")
	if alphaConvs.Total != 2 {
		t.Errorf("expected 2 conversations for svc-alpha, got %d", alphaConvs.Total)
	}

	// Verify beta has 1 conversation
	betaConvs := getConversations(t, app, "svc-beta")
	if betaConvs.Total != 1 {
		t.Errorf("expected 1 conversation for svc-beta, got %d", betaConvs.Total)
	}

	// Verify alpha conversations have the correct model
	for _, conv := range alphaConvs.Data {
		if conv.Model != "gpt-4" {
			t.Errorf("expected model %q for alpha conversation, got %q", "gpt-4", conv.Model)
		}
	}

	// Verify beta conversation has the correct model
	if len(betaConvs.Data) > 0 && betaConvs.Data[0].Model != "claude-3" {
		t.Errorf("expected model %q for beta conversation, got %q", "claude-3", betaConvs.Data[0].Model)
	}

	// Delete alpha, verify beta is unaffected
	delReq := httptest.NewRequest(http.MethodDelete, "/services/svc-alpha", nil)
	delResp, err := app.Test(delReq)
	if err != nil {
		t.Fatalf("failed to delete svc-alpha: %v", err)
	}
	defer func() { _ = delResp.Body.Close() }()

	if delResp.StatusCode != http.StatusNoContent {
		t.Errorf("expected status 204 for delete, got %d", delResp.StatusCode)
	}

	// Beta should still have its conversations
	betaConvsAfter := getConversations(t, app, "svc-beta")
	if betaConvsAfter.Total != 1 {
		t.Errorf("expected 1 conversation for svc-beta after deleting svc-alpha, got %d", betaConvsAfter.Total)
	}
}

func TestIntegration_ConcurrentRequests(t *testing.T) {
	app, _, mockOR, logger := setupTestApp(t)

	// Create a single service
	createService(t, app, "concurrent-svc", "Concurrent Test Service")

	// Launch 10 goroutines hitting the same service simultaneously
	const numGoroutines = 10
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()

			body, _ := json.Marshal(models.ChatCompletionRequest{
				Model: "gpt-4",
				Messages: []models.ChatMessage{
					{Role: "user", Content: fmt.Sprintf("Concurrent message %d", idx)},
				},
			})
			req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Service-ID", "concurrent-svc")

			resp, err := app.Test(req)
			if err != nil {
				t.Errorf("goroutine %d: request failed: %v", idx, err)
				return
			}
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode != http.StatusOK {
				respBody, _ := io.ReadAll(resp.Body)
				t.Errorf("goroutine %d: expected status 200, got %d: %s", idx, resp.StatusCode, string(respBody))
				return
			}

			var chatResp models.ChatCompletionResponse
			respBody, _ := io.ReadAll(resp.Body)
			if err := json.Unmarshal(respBody, &chatResp); err != nil {
				t.Errorf("goroutine %d: failed to decode response: %v", idx, err)
				return
			}

			if len(chatResp.Choices) != 1 {
				t.Errorf("goroutine %d: expected 1 choice, got %d", idx, len(chatResp.Choices))
			}
		}(i)
	}

	wg.Wait()

	// Verify the mock was called the expected number of times
	mockOR.mu.Lock()
	callCount := mockOR.callCount
	mockOR.mu.Unlock()

	if callCount != numGoroutines {
		t.Errorf("expected %d calls to ChatComplete, got %d", numGoroutines, callCount)
	}

	// Close logger to drain async writes
	logger.Close()

	// Verify all conversations were logged
	convResp := getConversations(t, app, "concurrent-svc")
	if convResp.Total != numGoroutines {
		t.Errorf("expected %d conversations, got %d", numGoroutines, convResp.Total)
	}
}
