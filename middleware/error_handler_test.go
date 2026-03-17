package middleware

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/hra42/tenantai/models"
	"github.com/hra42/tenantai/service"
)

func testErrorHandler(t *testing.T, returnErr error, wantStatus int, wantCode string) {
	t.Helper()

	app := fiber.New(fiber.Config{ErrorHandler: ErrorHandler})
	app.Get("/test", func(c fiber.Ctx) error {
		return returnErr
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != wantStatus {
		t.Errorf("status = %d, want %d", resp.StatusCode, wantStatus)
	}

	body, _ := io.ReadAll(resp.Body)
	var apiErr models.APIError
	if err := json.Unmarshal(body, &apiErr); err != nil {
		t.Fatalf("unmarshal error response: %v (body: %s)", err, body)
	}
	if apiErr.Error.Code != wantCode {
		t.Errorf("code = %q, want %q", apiErr.Error.Code, wantCode)
	}
	if apiErr.Error.Status != wantStatus {
		t.Errorf("error.status = %d, want %d", apiErr.Error.Status, wantStatus)
	}
}

func TestErrorHandler_AppError(t *testing.T) {
	testErrorHandler(t,
		&AppError{Status: 422, Code: CodeInvalidRequest, Message: "bad field"},
		422, CodeInvalidRequest,
	)
}

func TestErrorHandler_AppError_InternalError(t *testing.T) {
	testErrorHandler(t,
		NewInternalError("something broke"),
		http.StatusInternalServerError, CodeInternalError,
	)
}

func TestErrorHandler_AppError_MissingHeader(t *testing.T) {
	testErrorHandler(t,
		NewMissingHeaderError("X-Service-ID"),
		http.StatusBadRequest, CodeMissingHeader,
	)
}

func TestErrorHandler_FiberError_NotFound(t *testing.T) {
	testErrorHandler(t,
		fiber.NewError(fiber.StatusNotFound, "not found"),
		http.StatusNotFound, CodeServiceNotFound,
	)
}

func TestErrorHandler_FiberError_BadRequest(t *testing.T) {
	testErrorHandler(t,
		fiber.NewError(fiber.StatusBadRequest, "bad request"),
		http.StatusBadRequest, CodeInvalidRequest,
	)
}

func TestErrorHandler_FiberError_TooManyRequests(t *testing.T) {
	testErrorHandler(t,
		fiber.NewError(fiber.StatusTooManyRequests, "slow down"),
		http.StatusTooManyRequests, CodeRateLimited,
	)
}

func TestErrorHandler_FiberError_ServerError(t *testing.T) {
	testErrorHandler(t,
		fiber.NewError(fiber.StatusBadGateway, "upstream down"),
		http.StatusBadGateway, CodeInternalError,
	)
}

func TestErrorHandler_ErrServiceNotFound(t *testing.T) {
	testErrorHandler(t,
		service.ErrServiceNotFound,
		http.StatusNotFound, CodeServiceNotFound,
	)
}

func TestErrorHandler_ErrServiceExists(t *testing.T) {
	testErrorHandler(t,
		service.ErrServiceExists,
		http.StatusConflict, CodeInvalidRequest,
	)
}

func TestErrorHandler_GenericError(t *testing.T) {
	testErrorHandler(t,
		errors.New("unexpected failure"),
		http.StatusInternalServerError, CodeInternalError,
	)
}
