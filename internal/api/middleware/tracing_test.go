package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	"github.com/breatheroute/breatheroute/internal/api/middleware"
)

func setupTestTracer() (*tracetest.SpanRecorder, func()) {
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))

	// Set as global provider
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return sr, func() {
		_ = tp.Shutdown(context.Background())
	}
}

func TestTracing_CreatesSpan(t *testing.T) {
	sr, cleanup := setupTestTracer()
	defer cleanup()

	handler := middleware.Tracing("test-service")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify span is in context
		span := trace.SpanFromContext(r.Context())
		assert.True(t, span.SpanContext().IsValid())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test/path", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	spans := sr.Ended()
	assert.Len(t, spans, 1)
	assert.Equal(t, "GET /test/path", spans[0].Name())
}

func TestTracing_PropagatesContext(t *testing.T) {
	sr, cleanup := setupTestTracer()
	defer cleanup()

	handler := middleware.Tracing("test-service")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	// Set trace context header (W3C Trace Context format)
	req.Header.Set("traceparent", "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	spans := sr.Ended()
	assert.Len(t, spans, 1)
	// Span should be a child of the propagated context
	assert.Equal(t, "0af7651916cd43dd8448eb211c80319c", spans[0].SpanContext().TraceID().String())
}

func TestTracing_RecordsStatusCode(t *testing.T) {
	sr, cleanup := setupTestTracer()
	defer cleanup()

	handler := middleware.Tracing("test-service")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	spans := sr.Ended()
	assert.Len(t, spans, 1)

	// Check that status code attribute is set
	attrs := spans[0].Attributes()
	found := false
	for _, attr := range attrs {
		if attr.Key == "http.status_code" {
			found = true
			assert.Equal(t, int64(404), attr.Value.AsInt64())
			break
		}
	}
	assert.True(t, found, "http.status_code attribute should be set")
}

func TestTracing_MarksErrorOnServerError(t *testing.T) {
	sr, cleanup := setupTestTracer()
	defer cleanup()

	handler := middleware.Tracing("test-service")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	spans := sr.Ended()
	assert.Len(t, spans, 1)

	// Span should be marked as error
	status := spans[0].Status()
	assert.Equal(t, codes.Error, status.Code)
	assert.Equal(t, "Internal Server Error", status.Description)
}

func TestTracing_IncludesRequestID(t *testing.T) {
	sr, cleanup := setupTestTracer()
	defer cleanup()

	// Chain RequestID middleware with Tracing middleware
	handler := middleware.RequestID(
		middleware.Tracing("test-service")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})),
	)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	spans := sr.Ended()
	assert.Len(t, spans, 1)

	// Check that request.id attribute is set
	attrs := spans[0].Attributes()
	found := false
	for _, attr := range attrs {
		if attr.Key == "request.id" {
			found = true
			assert.Contains(t, attr.Value.AsString(), "req_")
			break
		}
	}
	assert.True(t, found, "request.id attribute should be set")
}
