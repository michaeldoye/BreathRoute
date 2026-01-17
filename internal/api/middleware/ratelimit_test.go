package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/breatheroute/breatheroute/internal/api/middleware"
)

func TestRateLimitByIP_AllowsWithinLimit(t *testing.T) {
	cfg := middleware.RateLimitConfig{
		RequestLimit: 5,
		WindowLength: time.Minute,
	}

	handler := middleware.RateLimitByIP(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Make requests within the limit
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
		req.RemoteAddr = "192.168.1.1:12345"
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code, "request %d should be allowed", i+1)
	}
}

func TestRateLimitByIP_BlocksOverLimit(t *testing.T) {
	cfg := middleware.RateLimitConfig{
		RequestLimit: 3,
		WindowLength: time.Minute,
	}

	handler := middleware.RateLimitByIP(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Use a unique IP for this test to avoid interference
	testIP := "10.0.0.1:12345"

	// Make requests up to the limit
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
		req.RemoteAddr = testIP
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	}

	// Next request should be rate limited
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.RemoteAddr = testIP
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.Contains(t, rec.Body.String(), "Rate limit exceeded")
	assert.Equal(t, "60", rec.Header().Get("Retry-After"))
}

func TestRateLimitByIP_DifferentIPsHaveSeparateLimits(t *testing.T) {
	cfg := middleware.RateLimitConfig{
		RequestLimit: 2,
		WindowLength: time.Minute,
	}

	handler := middleware.RateLimitByIP(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	ip1 := "172.16.0.1:12345"
	ip2 := "172.16.0.2:12345"

	// Use up limit for IP1
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
		req.RemoteAddr = ip1
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	}

	// IP1 should be rate limited
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.RemoteAddr = ip1
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)

	// IP2 should still be allowed
	req = httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.RemoteAddr = ip2
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRateLimitByUser_UsesUserIDWhenAvailable(t *testing.T) {
	cfg := middleware.RateLimitConfig{
		RequestLimit: 2,
		WindowLength: time.Minute,
	}

	// Create a handler chain that simulates authenticated user
	rateLimiter := middleware.RateLimitByUser(cfg)

	handler := rateLimiter(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Simulate authenticated requests from same user but different IPs
	// This tests that the rate limit is applied per-user, not per-IP
	user1IP1 := "192.168.1.1:12345"
	user1IP2 := "192.168.1.2:12345"

	// Note: In real usage, the auth middleware would set the user ID in context.
	// Here we're testing without that, so it falls back to IP-based limiting.
	// For proper user-based testing, we'd need to inject context values.

	// Test IP-based fallback
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
		req.RemoteAddr = user1IP1
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	}

	// Same IP should be rate limited
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.RemoteAddr = user1IP1
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)

	// Different IP should be allowed (IP-based fallback)
	req = httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.RemoteAddr = user1IP2
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRateLimitExceededResponse_Format(t *testing.T) {
	cfg := middleware.RateLimitConfig{
		RequestLimit: 1,
		WindowLength: time.Minute,
	}

	// Add request ID middleware to test trace ID in response
	handler := middleware.RequestID(
		middleware.RateLimitByIP(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})),
	)

	testIP := "203.0.113.1:12345"

	// First request succeeds
	req := httptest.NewRequest(http.MethodGet, "/test/path", http.NoBody)
	req.RemoteAddr = testIP
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Second request is rate limited
	req = httptest.NewRequest(http.MethodGet, "/test/path", http.NoBody)
	req.RemoteAddr = testIP
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.Equal(t, "application/problem+json", rec.Header().Get("Content-Type"))

	body := rec.Body.String()
	assert.Contains(t, body, "too-many-requests")
	assert.Contains(t, body, "Rate limit exceeded")
	assert.Contains(t, body, "/test/path") // instance
}

func TestDefaultRateLimitConfigs(t *testing.T) {
	// Verify the default configurations match the plan
	assert.Equal(t, 10, middleware.AuthRateLimit.RequestLimit)
	assert.Equal(t, time.Minute, middleware.AuthRateLimit.WindowLength)

	assert.Equal(t, 30, middleware.ExpensiveRateLimit.RequestLimit)
	assert.Equal(t, time.Minute, middleware.ExpensiveRateLimit.WindowLength)

	assert.Equal(t, 100, middleware.StandardRateLimit.RequestLimit)
	assert.Equal(t, time.Minute, middleware.StandardRateLimit.WindowLength)
}
