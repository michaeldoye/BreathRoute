package middleware_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/breatheroute/breatheroute/internal/api/middleware"
)

func TestLogger_LogsRequest(t *testing.T) {
	var buf bytes.Buffer
	log := zerolog.New(&buf)

	handler := middleware.Logger(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("response body"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/test/path", http.NoBody)
	req.Header.Set("User-Agent", "test-agent")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Parse log output
	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "request completed", logEntry["message"])
	assert.Equal(t, "GET", logEntry["method"])
	assert.Equal(t, "/test/path", logEntry["path"])
	assert.Equal(t, float64(200), logEntry["status"])
	assert.Equal(t, float64(13), logEntry["bytes"]) // len("response body")
	assert.Equal(t, "test-agent", logEntry["user_agent"])
	assert.NotEmpty(t, logEntry["duration"])
}

func TestLogger_LogsErrorStatus(t *testing.T) {
	var buf bytes.Buffer
	log := zerolog.New(&buf)

	handler := middleware.Logger(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/resource", http.NoBody)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "POST", logEntry["method"])
	assert.Equal(t, float64(500), logEntry["status"])
}

func TestLogger_IncludesRequestID(t *testing.T) {
	var buf bytes.Buffer
	log := zerolog.New(&buf)

	// Chain RequestID middleware before Logger
	handler := middleware.RequestID(
		middleware.Logger(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})),
	)

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	requestID, ok := logEntry["request_id"].(string)
	assert.True(t, ok)
	assert.Contains(t, requestID, "req_")
}

func TestLogger_IncludesTraceID(t *testing.T) {
	// Setup test tracer
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	defer func() { _ = tp.Shutdown(context.Background()) }()

	var buf bytes.Buffer
	log := zerolog.New(&buf)

	// Chain Tracing middleware before Logger
	handler := middleware.Tracing("test-service")(
		middleware.Logger(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})),
	)

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	// Should have trace_id and span_id
	traceID, ok := logEntry["trace_id"].(string)
	assert.True(t, ok)
	assert.NotEmpty(t, traceID)
	assert.Len(t, traceID, 32) // trace ID is 32 hex chars

	spanID, ok := logEntry["span_id"].(string)
	assert.True(t, ok)
	assert.NotEmpty(t, spanID)
	assert.Len(t, spanID, 16) // span ID is 16 hex chars
}

func TestLogger_DefaultStatusCode(t *testing.T) {
	var buf bytes.Buffer
	log := zerolog.New(&buf)

	handler := middleware.Logger(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Don't call WriteHeader - should default to 200
		_, _ = w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, float64(200), logEntry["status"])
}
