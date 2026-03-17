package handler

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/hra42/tenantai/middleware"
	"github.com/hra42/tenantai/models"
	"github.com/hra42/tenantai/openrouter"
)

// ConversationLog holds data for async conversation logging.
type ConversationLog struct {
	ID           string
	SessionID    string
	Model        string
	Messages     json.RawMessage
	FinishReason string
	DB           *sql.DB
	// TODO: Add PromptVersionID string field for prompt A/B testing (see docs/EXTENDING.md)
	// TODO: Add UserID string field for per-user tracking when JWT auth is enabled
}

// ConversationLogger writes conversation logs asynchronously.
type ConversationLogger struct {
	ch   chan ConversationLog
	done chan struct{}
	wg   sync.WaitGroup
}

// NewConversationLogger creates a logger with a buffered channel.
func NewConversationLogger(bufferSize int) *ConversationLogger {
	cl := &ConversationLogger{
		ch:   make(chan ConversationLog, bufferSize),
		done: make(chan struct{}),
	}
	cl.wg.Add(1)
	go cl.worker()
	return cl
}

// Log sends a conversation entry for async writing. Drops if buffer is full.
func (cl *ConversationLogger) Log(entry ConversationLog) {
	select {
	case cl.ch <- entry:
	default:
		slog.Warn("conversation log buffer full, dropping entry", "id", entry.ID)
	}
}

// Close stops the logger and waits for pending writes to drain.
func (cl *ConversationLogger) Close() {
	close(cl.ch)
	cl.wg.Wait()
}

func (cl *ConversationLogger) worker() {
	defer cl.wg.Done()
	for entry := range cl.ch {
		_, err := entry.DB.Exec(
			"INSERT INTO conversations (id, session_id, model, messages, finish_reason) VALUES (?, ?, ?, ?, ?)",
			entry.ID, entry.SessionID, entry.Model, string(entry.Messages), entry.FinishReason,
		)
		if err != nil {
			slog.Error("failed to log conversation", "id", entry.ID, "error", err)
		}
	}
}

// ChatHandler handles chat completion requests.
type ChatHandler struct {
	orClient openrouter.ChatCompleter
	logger   *ConversationLogger
}

// NewChatHandler creates a new ChatHandler.
func NewChatHandler(orClient openrouter.ChatCompleter, logger *ConversationLogger) *ChatHandler {
	return &ChatHandler{
		orClient: orClient,
		logger:   logger,
	}
}

// HandleChatCompletion processes a chat completion request.
func (h *ChatHandler) HandleChatCompletion(c fiber.Ctx) error {
	svc, err := middleware.GetServiceFromContext(c)
	if err != nil {
		return middleware.NewInternalError(err.Error())
	}

	db, err := middleware.GetDBFromContext(c)
	if err != nil {
		return middleware.NewInternalError(err.Error())
	}

	var req models.ChatCompletionRequest
	if err := c.Bind().JSON(&req); err != nil {
		return middleware.NewInvalidRequestError("invalid request body")
	}

	if req.Model == "" {
		return middleware.NewInvalidRequestError("model is required")
	}
	if len(req.Messages) == 0 {
		return middleware.NewInvalidRequestError("messages must not be empty")
	}

	if req.Stream {
		return h.handleStream(c, db, svc.ID, &req)
	}

	resp, err := h.orClient.ChatComplete(c.Context(), &req)
	if err != nil {
		return err
	}

	// Async conversation logging
	convID := uuid.New().String()
	sessionID := c.Get("X-Session-ID")

	finishReason := ""
	if len(resp.Choices) > 0 {
		finishReason = resp.Choices[0].FinishReason
	}

	messagesJSON, err := json.Marshal(map[string]interface{}{
		"request":  req.Messages,
		"response": resp.Choices,
	})
	if err != nil {
		slog.Error("failed to marshal conversation messages", "error", err)
	} else {
		h.logger.Log(ConversationLog{
			ID:           convID,
			SessionID:    sessionID,
			Model:        req.Model,
			Messages:     messagesJSON,
			FinishReason: finishReason,
			DB:           db,
		})
	}

	// TODO: Send webhook notification for new conversation (see docs/EXTENDING.md)
	// TODO: Inject prompt version from PromptSelector before calling OpenRouter
	// TODO: Extract user context from JWT claims when auth middleware is enabled

	return c.JSON(resp)
}

func (h *ChatHandler) handleStream(c fiber.Ctx, db *sql.DB, serviceID string, req *models.ChatCompletionRequest) error {
	stream, err := h.orClient.ChatCompleteStream(c.Context(), req)
	if err != nil {
		return err
	}

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")

	convID := uuid.New().String()
	sessionID := c.Get("X-Session-ID")

	return c.SendStreamWriter(func(w *bufio.Writer) {
		var fullContent string
		var finishReason string

		for resp := range stream.Events() {
			for _, choice := range resp.Choices {
				if choice.Delta != nil {
					if content, ok := choice.Delta.Content.(string); ok {
						fullContent += content
					}
				}
				if choice.FinishReason != "" {
					finishReason = choice.FinishReason
				}
			}

			data, err := json.Marshal(resp)
			if err != nil {
				slog.Error("failed to marshal stream event", "error", err)
				continue
			}
			_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
			_ = w.Flush()
		}

		if streamErr := stream.Err(); streamErr != nil {
			slog.Error("stream error", "service_id", serviceID, "error", streamErr)
		}

		_, _ = fmt.Fprintf(w, "data: [DONE]\n\n")
		_ = w.Flush()

		// Log conversation
		messagesJSON, err := json.Marshal(map[string]interface{}{
			"request":  req.Messages,
			"response": fullContent,
		})
		if err == nil {
			h.logger.Log(ConversationLog{
				ID:           convID,
				SessionID:    sessionID,
				Model:        req.Model,
				Messages:     messagesJSON,
				FinishReason: finishReason,
				DB:           db,
			})
		}
	})
}
