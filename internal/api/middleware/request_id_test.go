package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/breatheroute/breatheroute/internal/api/middleware"
)

func TestRequestID_GeneratesNewID(t *testing.T) {
	handler := middleware.RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify ID is available in context
		id := middleware.GetRequestID(r.Context())
		assert.NotEmpty(t, id)
		assert.Contains(t, id, "req_")

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Verify ID is set in response header
	responseID := w.Header().Get("X-Request-Id")
	assert.NotEmpty(t, responseID)
	assert.Contains(t, responseID, "req_")
}

func TestRequestID_PreservesExistingID(t *testing.T) {
	handler := middleware.RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := middleware.GetRequestID(r.Context())
		assert.Equal(t, "existing_request_id", id)

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Request-Id", "existing_request_id")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, "existing_request_id", w.Header().Get("X-Request-Id"))
}

func TestGetRequestID_ReturnsEmptyStringForMissingContext(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	id := middleware.GetRequestID(req.Context())
	assert.Empty(t, id)
}

func TestRequestID_UniqueIDs(t *testing.T) {
	handler := middleware.RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	ids := make(map[string]bool)

	for i := 0; i < 100; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		id := w.Header().Get("X-Request-Id")
		assert.NotEmpty(t, id)

		// Verify uniqueness
		assert.False(t, ids[id], "duplicate request ID generated: %s", id)
		ids[id] = true
	}
}
