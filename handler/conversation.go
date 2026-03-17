package handler

import (
	"database/sql"
	"encoding/json"
	"strconv"

	"github.com/gofiber/fiber/v3"

	"github.com/hra42/tenantai/middleware"
	"github.com/hra42/tenantai/models"
	"github.com/hra42/tenantai/service"
)

// ConversationHandler handles conversation history queries.
type ConversationHandler struct {
	mgr service.ServiceManager
}

// NewConversationHandler creates a new ConversationHandler.
func NewConversationHandler(mgr service.ServiceManager) *ConversationHandler {
	return &ConversationHandler{mgr: mgr}
}

// HandleList returns paginated conversation history for a service.
func (h *ConversationHandler) HandleList(c fiber.Ctx) error {
	id := c.Params("id")

	db, err := h.mgr.GetDBConnection(c.Context(), id)
	if err != nil {
		return err
	}

	// Parse pagination params
	limit := 100
	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}
	if limit > 1000 {
		limit = 1000
	}

	offset := 0
	if o := c.Query("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}

	sessionID := c.Query("session_id")

	// Validate sort param
	sort := "created_at"
	if s := c.Query("sort"); s == "updated_at" {
		sort = "updated_at"
	}

	// Build query
	baseWhere := ""
	args := []interface{}{}
	if sessionID != "" {
		baseWhere = " WHERE session_id = ?"
		args = append(args, sessionID)
	}

	// Count total
	var total int
	countQuery := "SELECT COUNT(*) FROM conversations" + baseWhere
	if err := db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return middleware.NewInternalError("failed to count conversations")
	}

	// Fetch rows — CAST needed for DuckDB TIMESTAMP/JSON → Go string scanning
	query := "SELECT id, CAST(created_at AS VARCHAR), session_id, model, CAST(messages AS VARCHAR), finish_reason FROM conversations" +
		baseWhere + " ORDER BY " + sort + " DESC LIMIT ? OFFSET ?"
	queryArgs := append(args, limit, offset)

	rows, err := db.Query(query, queryArgs...)
	if err != nil {
		return middleware.NewInternalError("failed to query conversations")
	}
	defer func() { _ = rows.Close() }()

	conversations := []models.ConversationResponse{}
	for rows.Next() {
		var conv models.ConversationResponse
		var createdAt string
		var sessionIDVal sql.NullString
		var finishReason sql.NullString
		var messagesStr string

		if err := rows.Scan(&conv.ID, &createdAt, &sessionIDVal, &conv.Model, &messagesStr, &finishReason); err != nil {
			return middleware.NewInternalError("failed to scan conversation row")
		}

		conv.CreatedAt = createdAt
		if sessionIDVal.Valid {
			conv.SessionID = &sessionIDVal.String
		}
		if finishReason.Valid {
			conv.FinishReason = &finishReason.String
		}
		conv.Messages = json.RawMessage(messagesStr)

		conversations = append(conversations, conv)
	}

	return c.JSON(models.ConversationListResponse{
		Data:   conversations,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}
