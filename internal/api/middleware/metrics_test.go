package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/breatheroute/breatheroute/internal/api/middleware"
)

func TestNewMetrics(t *testing.T) {
	metrics, err := middleware.NewMetrics()
	require.NoError(t, err)
	assert.NotNil(t, metrics)
}

func TestMetrics_Middleware_Success(t *testing.T) {
	metrics, err := middleware.NewMetrics()
	require.NoError(t, err)

	handler := metrics.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/test/path", http.NoBody)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "OK", w.Body.String())
}

func TestMetrics_Middleware_Error(t *testing.T) {
	metrics, err := middleware.NewMetrics()
	require.NoError(t, err)

	handler := metrics.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("error"))
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/resource", http.NoBody)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestMetrics_Middleware_BadRequest(t *testing.T) {
	metrics, err := middleware.NewMetrics()
	require.NoError(t, err)

	handler := metrics.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error": "bad request"}`))
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/resource", http.NoBody)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestMetrics_Middleware_DefaultStatusCode(t *testing.T) {
	metrics, err := middleware.NewMetrics()
	require.NoError(t, err)

	handler := metrics.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Don't explicitly call WriteHeader - should default to 200
		_, _ = w.Write([]byte("response"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestNewProviderMetrics(t *testing.T) {
	pm, err := middleware.NewProviderMetrics("test-provider")
	require.NoError(t, err)
	assert.NotNil(t, pm)
}

func TestProviderMetrics_RecordCacheHit(t *testing.T) {
	pm, err := middleware.NewProviderMetrics("test-provider")
	require.NoError(t, err)

	// Should not panic
	pm.RecordCacheHit("air-quality", "get-stations")
}

func TestProviderMetrics_RecordCacheMiss(t *testing.T) {
	pm, err := middleware.NewProviderMetrics("test-provider")
	require.NoError(t, err)

	// Should not panic
	pm.RecordCacheMiss("air-quality", "get-stations")
}
