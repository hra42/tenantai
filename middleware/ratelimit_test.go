package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestRateLimiter_AllowsBurst(t *testing.T) {
	rl := NewRateLimiter(5)
	defer rl.Close()

	for i := 0; i < 5; i++ {
		if !rl.Allow("1.2.3.4") {
			t.Errorf("request %d should be allowed", i+1)
		}
	}
}

func TestRateLimiter_RejectsAfterBurst(t *testing.T) {
	rl := NewRateLimiter(3)
	defer rl.Close()

	for i := 0; i < 3; i++ {
		rl.Allow("1.2.3.4")
	}

	if rl.Allow("1.2.3.4") {
		t.Error("request after burst should be rejected")
	}
}

func TestRateLimiter_IndependentPerIP(t *testing.T) {
	rl := NewRateLimiter(2)
	defer rl.Close()

	// Exhaust IP A
	rl.Allow("10.0.0.1")
	rl.Allow("10.0.0.1")
	if rl.Allow("10.0.0.1") {
		t.Error("IP A should be rate limited")
	}

	// IP B should still be allowed
	if !rl.Allow("10.0.0.2") {
		t.Error("IP B should not be rate limited")
	}
}

func TestRateLimit_Middleware_Returns429(t *testing.T) {
	rl := NewRateLimiter(1)
	defer rl.Close()

	app := fiber.New(fiber.Config{ErrorHandler: ErrorHandler})
	app.Use(RateLimit(rl))
	app.Get("/test", func(c fiber.Ctx) error {
		return c.SendString("ok")
	})

	// First request should succeed
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("first request: status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	// Second request should be rate limited
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("second request: status = %d, want %d", resp.StatusCode, http.StatusTooManyRequests)
	}
}
