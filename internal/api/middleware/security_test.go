package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/breatheroute/breatheroute/internal/api/middleware"
)

func TestSecurityHeaders(t *testing.T) {
	handler := middleware.SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify all security headers are set
	assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", rec.Header().Get("X-Frame-Options"))
	assert.Equal(t, "max-age=31536000; includeSubDomains", rec.Header().Get("Strict-Transport-Security"))
	assert.Equal(t, "default-src 'none'; frame-ancestors 'none'", rec.Header().Get("Content-Security-Policy"))
	assert.Equal(t, "strict-origin-when-cross-origin", rec.Header().Get("Referrer-Policy"))
	assert.Equal(t, "geolocation=(), camera=(), microphone=()", rec.Header().Get("Permissions-Policy"))
}

func TestSecurityHeaders_PreservesExistingHeaders(t *testing.T) {
	handler := middleware.SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Custom-Header", "custom-value")
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "custom-value", rec.Header().Get("X-Custom-Header"))
	// Security headers should still be present
	assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
}

func TestRequireTLS_DisabledByDefault(t *testing.T) {
	// Clear the env var to ensure it's disabled
	t.Setenv("REQUIRE_TLS", "")

	handler := middleware.RequireTLS(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.Header.Set("X-Forwarded-Proto", "http") // Not HTTPS
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Should allow request when REQUIRE_TLS is not set
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRequireTLS_Enabled_RejectsHTTP(t *testing.T) {
	t.Setenv("REQUIRE_TLS", "true")

	handler := middleware.RequireTLS(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.Header.Set("X-Forwarded-Proto", "http")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Contains(t, rec.Body.String(), "TLS required")
	assert.Contains(t, rec.Body.String(), "This endpoint requires HTTPS")
}

func TestRequireTLS_Enabled_AllowsHTTPS(t *testing.T) {
	t.Setenv("REQUIRE_TLS", "true")

	handler := middleware.RequireTLS(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.Header.Set("X-Forwarded-Proto", "https")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRequireTLS_Enabled_NoHeaderAllowed(t *testing.T) {
	t.Setenv("REQUIRE_TLS", "true")

	handler := middleware.RequireTLS(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	// No X-Forwarded-Proto header (e.g., direct connection)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Allow requests without the header (direct connections, local dev)
	assert.Equal(t, http.StatusOK, rec.Code)
}
