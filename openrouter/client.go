package openrouter

import (
	"context"
	"fmt"
	"log"

	or "github.com/hra42/openrouter-go"

	"github.com/hra42/tenantai/middleware"
	"github.com/hra42/tenantai/models"
)

// ChatCompleter abstracts chat completion operations for testability.
type ChatCompleter interface {
	ChatComplete(ctx context.Context, req *models.ChatCompletionRequest) (*models.ChatCompletionResponse, error)
	ChatCompleteStream(ctx context.Context, req *models.ChatCompletionRequest) (*or.ChatStream, error)
}

// Client wraps the OpenRouter SDK client.
type Client struct {
	client *or.Client
	debug  bool
}

// NewClient creates a new OpenRouter client with optional debug logging.
func NewClient(apiKey string, debug bool) *Client {
	return &Client{
		client: or.NewClient(or.WithAPIKey(apiKey)),
		debug:  debug,
	}
}

// ChatComplete sends a chat completion request to OpenRouter.
func (c *Client) ChatComplete(ctx context.Context, req *models.ChatCompletionRequest) (*models.ChatCompletionResponse, error) {
	messages := make([]or.Message, len(req.Messages))
	for i, msg := range req.Messages {
		switch msg.Role {
		case "system":
			messages[i] = or.CreateSystemMessage(msg.Content)
		case "assistant":
			messages[i] = or.CreateAssistantMessage(msg.Content)
		default:
			messages[i] = or.CreateUserMessage(msg.Content)
		}
	}

	opts := []or.ChatCompletionOption{
		or.WithModel(req.Model),
	}
	if req.Temperature != nil {
		opts = append(opts, or.WithTemperature(*req.Temperature))
	}
	if req.MaxTokens != nil {
		opts = append(opts, or.WithMaxTokens(*req.MaxTokens))
	}
	if req.TopP != nil {
		opts = append(opts, or.WithTopP(*req.TopP))
	}
	if len(req.Stop) > 0 {
		opts = append(opts, or.WithStop(req.Stop...))
	}

	if c.debug {
		log.Printf("DEBUG openrouter: ChatComplete model=%s messages=%d", req.Model, len(req.Messages))
	}

	resp, err := c.client.ChatComplete(ctx, messages, opts...)
	if err != nil {
		return nil, mapError(err)
	}

	if c.debug {
		log.Printf("DEBUG openrouter: ChatComplete response id=%s model=%s choices=%d", resp.ID, resp.Model, len(resp.Choices))
	}

	choices := make([]models.Choice, len(resp.Choices))
	for i, ch := range resp.Choices {
		content := ""
		if s, ok := ch.Message.Content.(string); ok {
			content = s
		}
		choices[i] = models.Choice{
			Index: ch.Index,
			Message: models.ChatMessage{
				Role:    ch.Message.Role,
				Content: content,
			},
			FinishReason: ch.FinishReason,
		}
	}

	result := &models.ChatCompletionResponse{
		ID:      resp.ID,
		Object:  resp.Object,
		Created: resp.Created,
		Model:   resp.Model,
		Choices: choices,
	}
	if resp.Usage.TotalTokens > 0 {
		result.Usage = &models.Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		}
	}

	return result, nil
}

// ChatCompleteStream sends a streaming chat completion request.
func (c *Client) ChatCompleteStream(ctx context.Context, req *models.ChatCompletionRequest) (*or.ChatStream, error) {
	messages := make([]or.Message, len(req.Messages))
	for i, msg := range req.Messages {
		switch msg.Role {
		case "system":
			messages[i] = or.CreateSystemMessage(msg.Content)
		case "assistant":
			messages[i] = or.CreateAssistantMessage(msg.Content)
		default:
			messages[i] = or.CreateUserMessage(msg.Content)
		}
	}

	opts := []or.ChatCompletionOption{
		or.WithModel(req.Model),
	}
	if req.Temperature != nil {
		opts = append(opts, or.WithTemperature(*req.Temperature))
	}
	if req.MaxTokens != nil {
		opts = append(opts, or.WithMaxTokens(*req.MaxTokens))
	}
	if req.TopP != nil {
		opts = append(opts, or.WithTopP(*req.TopP))
	}
	if len(req.Stop) > 0 {
		opts = append(opts, or.WithStop(req.Stop...))
	}

	if c.debug {
		log.Printf("DEBUG openrouter: ChatCompleteStream model=%s messages=%d", req.Model, len(req.Messages))
	}

	stream, err := c.client.ChatCompleteStream(ctx, messages, opts...)
	if err != nil {
		return nil, mapError(err)
	}
	return stream, nil
}

// mapError translates SDK errors to AppError.
func mapError(err error) error {
	if reqErr, ok := or.IsRequestError(err); ok {
		if reqErr.IsRateLimitError() {
			return &middleware.AppError{
				Status:  429,
				Code:    middleware.CodeRateLimited,
				Message: fmt.Sprintf("rate limited: %s", reqErr.Message),
			}
		}
		if reqErr.IsAuthenticationError() {
			return &middleware.AppError{
				Status:  401,
				Code:    middleware.CodeOpenRouterError,
				Message: "invalid API key",
			}
		}
		return middleware.NewOpenRouterError(reqErr.Message, reqErr.StatusCode)
	}
	if valErr, ok := or.IsValidationError(err); ok {
		return middleware.NewInvalidRequestError(fmt.Sprintf("validation error: %s - %s", valErr.Field, valErr.Message))
	}
	return middleware.NewInternalError(fmt.Sprintf("openrouter error: %v", err))
}
