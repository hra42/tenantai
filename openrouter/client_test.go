package openrouter

import (
	"errors"
	"testing"

	or "github.com/hra42/openrouter-go"

	"github.com/hra42/tenantai/middleware"
)

func TestMapError_RateLimit(t *testing.T) {
	err := mapError(&or.RequestError{
		StatusCode: 429,
		Message:    "rate limit exceeded",
	})

	var appErr *middleware.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T: %v", err, err)
	}
	if appErr.Status != 429 {
		t.Errorf("status = %d, want 429", appErr.Status)
	}
	if appErr.Code != middleware.CodeRateLimited {
		t.Errorf("code = %q, want %q", appErr.Code, middleware.CodeRateLimited)
	}
}

func TestMapError_AuthError(t *testing.T) {
	err := mapError(&or.RequestError{
		StatusCode: 401,
		Message:    "invalid key",
	})

	var appErr *middleware.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T: %v", err, err)
	}
	if appErr.Status != 401 {
		t.Errorf("status = %d, want 401", appErr.Status)
	}
	if appErr.Code != middleware.CodeOpenRouterError {
		t.Errorf("code = %q, want %q", appErr.Code, middleware.CodeOpenRouterError)
	}
}

func TestMapError_OtherRequestError(t *testing.T) {
	err := mapError(&or.RequestError{
		StatusCode: 500,
		Message:    "server error",
	})

	var appErr *middleware.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T: %v", err, err)
	}
	if appErr.Status != 500 {
		t.Errorf("status = %d, want 500", appErr.Status)
	}
	if appErr.Code != middleware.CodeOpenRouterError {
		t.Errorf("code = %q, want %q", appErr.Code, middleware.CodeOpenRouterError)
	}
}

func TestMapError_ValidationError(t *testing.T) {
	err := mapError(&or.ValidationError{
		Field:   "model",
		Message: "model is required",
	})

	var appErr *middleware.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T: %v", err, err)
	}
	if appErr.Status != 400 {
		t.Errorf("status = %d, want 400", appErr.Status)
	}
	if appErr.Code != middleware.CodeInvalidRequest {
		t.Errorf("code = %q, want %q", appErr.Code, middleware.CodeInvalidRequest)
	}
}

func TestMapError_GenericError(t *testing.T) {
	err := mapError(errors.New("something unexpected"))

	var appErr *middleware.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T: %v", err, err)
	}
	if appErr.Status != 500 {
		t.Errorf("status = %d, want 500", appErr.Status)
	}
	if appErr.Code != middleware.CodeInternalError {
		t.Errorf("code = %q, want %q", appErr.Code, middleware.CodeInternalError)
	}
}
