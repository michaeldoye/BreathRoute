package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/breatheroute/breatheroute/internal/api/middleware"
	"github.com/breatheroute/breatheroute/internal/auth"
)

func TestAuth_MissingAuthorizationHeader(t *testing.T) {
	authService := createTestAuthService(t)
	authMiddleware := middleware.Auth(authService)

	handler := authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "missing authorization header")
}

func TestAuth_InvalidAuthorizationFormat(t *testing.T) {
	authService := createTestAuthService(t)
	authMiddleware := middleware.Auth(authService)

	handler := authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name   string
		header string
	}{
		{"no bearer prefix", "token123"},
		{"basic auth", "Basic dXNlcjpwYXNz"},
		{"bearer lowercase no space", "bearer token123"},
		{"empty bearer", "Bearer "},
		{"just bearer", "Bearer"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
			req.Header.Set("Authorization", tt.header)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusUnauthorized, rec.Code)
		})
	}
}

func TestAuth_InvalidToken(t *testing.T) {
	authService := createTestAuthService(t)
	authMiddleware := middleware.Auth(authService)

	handler := authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.Header.Set("Authorization", "Bearer invalid.jwt.token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	// Invalid tokens are detected and reported as such
	assert.Contains(t, rec.Body.String(), "invalid access token")
}

func TestAuth_ValidToken(t *testing.T) {
	authService := createTestAuthService(t)
	authMiddleware := middleware.Auth(authService)

	// Generate a valid token
	user := &auth.User{
		ID:        "usr_testuser123",
		AppleSub:  "apple.123",
		Locale:    "nl-NL",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	jwtService := auth.NewJWTService(auth.JWTConfig{
		SigningKey: "test-secret-key-for-testing-only",
		Issuer:     "https://api.breatheroute.nl",
		Audience:   "breatheroute-api",
	})

	token, _, err := jwtService.GenerateAccessToken(user)
	require.NoError(t, err)

	var capturedUserID string
	handler := authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUserID = middleware.GetUserID(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, user.ID, capturedUserID)
}

func TestAuth_CaseInsensitiveBearer(t *testing.T) {
	authService := createTestAuthService(t)
	authMiddleware := middleware.Auth(authService)

	// Generate a valid token
	user := &auth.User{
		ID:        "usr_testuser123",
		AppleSub:  "apple.123",
		Locale:    "nl-NL",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	jwtService := auth.NewJWTService(auth.JWTConfig{
		SigningKey: "test-secret-key-for-testing-only",
		Issuer:     "https://api.breatheroute.nl",
		Audience:   "breatheroute-api",
	})

	token, _, err := jwtService.GenerateAccessToken(user)
	require.NoError(t, err)

	handler := authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Test with different case variations
	cases := []string{"Bearer ", "bearer ", "BEARER "}
	for _, prefix := range cases {
		t.Run(prefix, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
			req.Header.Set("Authorization", prefix+token)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
		})
	}
}

func TestGetUserID_NoAuth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	userID := middleware.GetUserID(req.Context())
	assert.Empty(t, userID)
}

// createTestAuthService creates an auth service for testing.
func createTestAuthService(t *testing.T) *auth.Service {
	t.Helper()

	siwaVerifier := auth.NewSIWAVerifier(auth.SIWAConfig{
		BundleID: "nl.breatheroute.app",
	})

	jwtService := auth.NewJWTService(auth.JWTConfig{
		SigningKey: "test-secret-key-for-testing-only",
		Issuer:     "https://api.breatheroute.nl",
		Audience:   "breatheroute-api",
	})

	userRepo := auth.NewInMemoryUserRepository()
	refreshRepo := auth.NewInMemoryRefreshTokenRepository()

	return auth.NewService(auth.ServiceConfig{
		SIWAVerifier:  siwaVerifier,
		JWTService:    jwtService,
		UserRepo:      userRepo,
		RefreshRepo:   refreshRepo,
		DefaultLocale: "nl-NL",
	})
}
