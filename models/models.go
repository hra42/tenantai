package models

import "encoding/json"

// Chat completion request/response types

type ChatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Temperature *float64      `json:"temperature,omitempty"`
	MaxTokens   *int          `json:"max_tokens,omitempty"`
	Stream      bool          `json:"stream,omitempty"`
	Stop        []string      `json:"stop,omitempty"`
	TopP        *float64      `json:"top_p,omitempty"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   *Usage   `json:"usage,omitempty"`
}

type Choice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Service management types

type CreateServiceRequest struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ServiceResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

type ServiceDetailResponse struct {
	ServiceResponse
	DBFileSizeBytes   int64 `json:"db_file_size_bytes"`
	ConversationCount int   `json:"conversation_count"`
}

// Conversation types

type ConversationResponse struct {
	ID           string          `json:"id"`
	CreatedAt    string          `json:"created_at"`
	SessionID    *string         `json:"session_id,omitempty"`
	Model        string          `json:"model"`
	Messages     json.RawMessage `json:"messages"`
	FinishReason *string         `json:"finish_reason,omitempty"`
}

type ConversationListResponse struct {
	Data   []ConversationResponse `json:"data"`
	Total  int                    `json:"total"`
	Limit  int                    `json:"limit"`
	Offset int                    `json:"offset"`
}

// Error types

type APIError struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Message string `json:"message"`
	Code    string `json:"code"`
	Status  int    `json:"status"`
}
