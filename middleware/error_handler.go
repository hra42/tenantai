package middleware

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/gofiber/fiber/v3"
	"github.com/hra42/tenantai/models"
	"github.com/hra42/tenantai/service"
)

// Error code constants
const (
	CodeServiceNotFound = "SERVICE_NOT_FOUND"
	CodeInvalidRequest  = "INVALID_REQUEST"
	CodeOpenRouterError = "OPENROUTER_ERROR"
	CodeRateLimited     = "RATE_LIMITED"
	CodeInternalError   = "INTERNAL_ERROR"
	CodeMissingHeader   = "MISSING_HEADER"
	CodeUnauthorized    = "UNAUTHORIZED"
)

// AppError is a structured application error.
type AppError struct {
	Status  int
	Code    string
	Message string
}

func (e *AppError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func NewServiceNotFoundError(id string) *AppError {
	return &AppError{
		Status:  fiber.StatusNotFound,
		Code:    CodeServiceNotFound,
		Message: fmt.Sprintf("service %q not found", id),
	}
}

func NewInvalidRequestError(msg string) *AppError {
	return &AppError{
		Status:  fiber.StatusBadRequest,
		Code:    CodeInvalidRequest,
		Message: msg,
	}
}

func NewOpenRouterError(msg string, status int) *AppError {
	return &AppError{
		Status:  status,
		Code:    CodeOpenRouterError,
		Message: msg,
	}
}

func NewMissingHeaderError(header string) *AppError {
	return &AppError{
		Status:  fiber.StatusBadRequest,
		Code:    CodeMissingHeader,
		Message: fmt.Sprintf("missing required header: %s", header),
	}
}

func NewUnauthorizedError(msg string) *AppError {
	return &AppError{
		Status:  fiber.StatusUnauthorized,
		Code:    CodeUnauthorized,
		Message: msg,
	}
}

func NewInternalError(msg string) *AppError {
	return &AppError{
		Status:  fiber.StatusInternalServerError,
		Code:    CodeInternalError,
		Message: msg,
	}
}

// ErrorHandler is the custom Fiber error handler.
func ErrorHandler(c fiber.Ctx, err error) error {
	status := fiber.StatusInternalServerError
	code := CodeInternalError
	message := err.Error()

	var appErr *AppError
	var fiberErr *fiber.Error

	switch {
	case errors.As(err, &appErr):
		status = appErr.Status
		code = appErr.Code
		message = appErr.Message
	case errors.As(err, &fiberErr):
		status = fiberErr.Code
		message = fiberErr.Message
		switch status {
		case fiber.StatusNotFound:
			code = CodeServiceNotFound
		case fiber.StatusBadRequest:
			code = CodeInvalidRequest
		case fiber.StatusTooManyRequests:
			code = CodeRateLimited
		default:
			code = CodeInternalError
		}
	case errors.Is(err, service.ErrServiceNotFound):
		status = fiber.StatusNotFound
		code = CodeServiceNotFound
	case errors.Is(err, service.ErrServiceExists):
		status = fiber.StatusConflict
		code = CodeInvalidRequest
	}

	if status >= 500 {
		slog.Error("server error",
			"status", status,
			"code", code,
			"message", message,
			"method", c.Method(),
			"path", c.Path(),
		)
	}

	return c.Status(status).JSON(models.APIError{
		Error: models.ErrorDetail{
			Message: message,
			Code:    code,
			Status:  status,
		},
	})
}
