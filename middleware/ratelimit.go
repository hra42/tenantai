package middleware

import (
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
)

type tokenBucket struct {
	tokens     float64
	maxTokens  float64
	refillRate float64 // tokens per second
	lastRefill time.Time
}

// RateLimiter implements an in-memory per-IP token bucket rate limiter.
type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*tokenBucket
	rps     int
	stop    chan struct{}
}

// NewRateLimiter creates a rate limiter allowing requestsPerSecond per IP.
func NewRateLimiter(requestsPerSecond int) *RateLimiter {
	rl := &RateLimiter{
		buckets: make(map[string]*tokenBucket),
		rps:     requestsPerSecond,
		stop:    make(chan struct{}),
	}
	go rl.cleanup()
	return rl
}

// Allow checks whether the given IP is allowed to make a request.
func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	b, ok := rl.buckets[ip]
	if !ok {
		b = &tokenBucket{
			tokens:     float64(rl.rps),
			maxTokens:  float64(rl.rps),
			refillRate: float64(rl.rps),
			lastRefill: time.Now(),
		}
		rl.buckets[ip] = b
	}

	now := time.Now()
	elapsed := now.Sub(b.lastRefill).Seconds()
	b.tokens += elapsed * b.refillRate
	if b.tokens > b.maxTokens {
		b.tokens = b.maxTokens
	}
	b.lastRefill = now

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// Close stops the background cleanup goroutine.
func (rl *RateLimiter) Close() {
	close(rl.stop)
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			rl.mu.Lock()
			now := time.Now()
			for ip, b := range rl.buckets {
				if now.Sub(b.lastRefill) > 10*time.Minute {
					delete(rl.buckets, ip)
				}
			}
			rl.mu.Unlock()
		case <-rl.stop:
			return
		}
	}
}

// RateLimit returns middleware that applies the given RateLimiter.
func RateLimit(limiter *RateLimiter) fiber.Handler {
	return func(c fiber.Ctx) error {
		if !limiter.Allow(c.IP()) {
			return &AppError{
				Status:  fiber.StatusTooManyRequests,
				Code:    CodeRateLimited,
				Message: "rate limit exceeded",
			}
		}
		return c.Next()
	}
}
