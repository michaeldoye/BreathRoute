package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/httprate"

	"github.com/breatheroute/breatheroute/internal/api/models"
)

// RateLimitConfig holds configuration for rate limiting.
type RateLimitConfig struct {
	// Requests per window
	RequestLimit int
	// Window duration
	WindowLength time.Duration
}

// Default rate limit configurations.
var (
	// AuthRateLimit applies to authentication endpoints (10 req/min).
	AuthRateLimit = RateLimitConfig{
		RequestLimit: 10,
		WindowLength: time.Minute,
	}

	// ExpensiveRateLimit applies to computationally expensive endpoints (30 req/min).
	ExpensiveRateLimit = RateLimitConfig{
		RequestLimit: 30,
		WindowLength: time.Minute,
	}

	// StandardRateLimit applies to standard endpoints (100 req/min).
	StandardRateLimit = RateLimitConfig{
		RequestLimit: 100,
		WindowLength: time.Minute,
	}
)

// RateLimitByIP creates a rate limiter middleware using client IP address.
// Uses X-Forwarded-For header if present (extracted by chi's RealIP middleware).
func RateLimitByIP(cfg RateLimitConfig) func(http.Handler) http.Handler {
	return httprate.Limit(
		cfg.RequestLimit,
		cfg.WindowLength,
		httprate.WithKeyFuncs(httprate.KeyByRealIP),
		httprate.WithLimitHandler(rateLimitExceededHandler),
	)
}

// RateLimitByUser creates a rate limiter middleware using authenticated user ID.
// Falls back to IP-based rate limiting for unauthenticated requests.
func RateLimitByUser(cfg RateLimitConfig) func(http.Handler) http.Handler {
	return httprate.Limit(
		cfg.RequestLimit,
		cfg.WindowLength,
		httprate.WithKeyFuncs(keyByUserOrIP),
		httprate.WithLimitHandler(rateLimitExceededHandler),
	)
}

// keyByUserOrIP returns the user ID if authenticated, otherwise the client IP.
func keyByUserOrIP(r *http.Request) (string, error) {
	// Try to get user ID from context (set by auth middleware)
	if userID := GetUserID(r.Context()); userID != "" {
		return "user:" + userID, nil
	}

	// Fall back to IP-based rate limiting
	return httprate.KeyByRealIP(r)
}

// rateLimitExceededHandler writes an RFC7807 Problem response when rate limit is exceeded.
func rateLimitExceededHandler(w http.ResponseWriter, r *http.Request) {
	traceID := GetRequestID(r.Context())

	problem := models.NewTooManyRequests(traceID, "Rate limit exceeded. Please try again later.")
	problem.Instance = r.URL.Path

	// Add Retry-After header (estimate based on window)
	// httprate doesn't expose exact reset time, so we use a conservative estimate
	w.Header().Set("Retry-After", strconv.Itoa(60)) // 60 seconds

	problem.Write(w)
}
