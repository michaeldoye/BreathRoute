package middleware

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const meterName = "github.com/breatheroute/breatheroute/internal/api/middleware"

// Metrics holds the OpenTelemetry metrics instruments.
type Metrics struct {
	requestDuration  metric.Float64Histogram
	requestTotal     metric.Int64Counter
	requestsInFlight metric.Int64UpDownCounter
	responseSize     metric.Int64Histogram
}

// NewMetrics creates a new Metrics instance with initialized instruments.
func NewMetrics() (*Metrics, error) {
	meter := otel.Meter(meterName)

	requestDuration, err := meter.Float64Histogram(
		"http.server.request.duration",
		metric.WithDescription("Duration of HTTP server requests in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	requestTotal, err := meter.Int64Counter(
		"http.server.request.total",
		metric.WithDescription("Total number of HTTP server requests"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, err
	}

	requestsInFlight, err := meter.Int64UpDownCounter(
		"http.server.requests_in_flight",
		metric.WithDescription("Number of HTTP requests currently being processed"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, err
	}

	responseSize, err := meter.Int64Histogram(
		"http.server.response.size",
		metric.WithDescription("Size of HTTP server responses in bytes"),
		metric.WithUnit("By"),
	)
	if err != nil {
		return nil, err
	}

	return &Metrics{
		requestDuration:  requestDuration,
		requestTotal:     requestTotal,
		requestsInFlight: requestsInFlight,
		responseSize:     responseSize,
	}, nil
}

// Middleware returns an HTTP middleware that records metrics for each request.
func (m *Metrics) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Track request in flight
			attrs := []attribute.KeyValue{
				attribute.String("http.method", r.Method),
				attribute.String("http.route", r.URL.Path),
			}
			m.requestsInFlight.Add(r.Context(), 1, metric.WithAttributes(attrs...))
			defer m.requestsInFlight.Add(r.Context(), -1, metric.WithAttributes(attrs...))

			// Wrap response writer
			wrapped := newMetricsResponseWriter(w)

			// Process request
			next.ServeHTTP(wrapped, r)

			// Calculate duration
			duration := time.Since(start).Seconds()

			// Build attributes with status code
			attrs = append(attrs, attribute.String("http.status_code", strconv.Itoa(wrapped.statusCode)))

			// Add error attribute for 4xx/5xx responses
			if wrapped.statusCode >= 400 {
				attrs = append(attrs, attribute.Bool("error", true))
			}

			// Record metrics
			m.requestDuration.Record(r.Context(), duration, metric.WithAttributes(attrs...))
			m.requestTotal.Add(r.Context(), 1, metric.WithAttributes(attrs...))
			m.responseSize.Record(r.Context(), wrapped.written, metric.WithAttributes(attrs...))
		})
	}
}

// metricsResponseWriter wraps http.ResponseWriter to capture response metadata.
type metricsResponseWriter struct {
	http.ResponseWriter
	statusCode int
	written    int64
}

func newMetricsResponseWriter(w http.ResponseWriter) *metricsResponseWriter {
	return &metricsResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
}

func (rw *metricsResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *metricsResponseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.written += int64(n)
	return n, err
}

// ProviderMetrics holds metrics for external provider calls.
type ProviderMetrics struct {
	requestDuration metric.Float64Histogram
	requestTotal    metric.Int64Counter
	cacheHitRate    metric.Float64Counter
	cacheMissRate   metric.Float64Counter
}

// NewProviderMetrics creates metrics for monitoring external provider calls.
func NewProviderMetrics(providerName string) (*ProviderMetrics, error) {
	meter := otel.Meter(meterName)

	requestDuration, err := meter.Float64Histogram(
		"provider.request.duration",
		metric.WithDescription("Duration of provider requests in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	requestTotal, err := meter.Int64Counter(
		"provider.request.total",
		metric.WithDescription("Total number of provider requests"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, err
	}

	cacheHitRate, err := meter.Float64Counter(
		"provider.cache.hit",
		metric.WithDescription("Number of cache hits"),
		metric.WithUnit("{hit}"),
	)
	if err != nil {
		return nil, err
	}

	cacheMissRate, err := meter.Float64Counter(
		"provider.cache.miss",
		metric.WithDescription("Number of cache misses"),
		metric.WithUnit("{miss}"),
	)
	if err != nil {
		return nil, err
	}

	return &ProviderMetrics{
		requestDuration: requestDuration,
		requestTotal:    requestTotal,
		cacheHitRate:    cacheHitRate,
		cacheMissRate:   cacheMissRate,
	}, nil
}

// RecordRequest records metrics for a provider request.
func (m *ProviderMetrics) RecordRequest(provider, operation string, duration time.Duration, err error) {
	attrs := []attribute.KeyValue{
		attribute.String("provider.name", provider),
		attribute.String("provider.operation", operation),
	}

	if err != nil {
		attrs = append(attrs, attribute.Bool("error", true))
	}

	// Use background context for metrics to avoid context cancellation issues
	ctx := context.TODO()
	m.requestDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
	m.requestTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordCacheHit records a cache hit for a provider.
func (m *ProviderMetrics) RecordCacheHit(provider, operation string) {
	attrs := []attribute.KeyValue{
		attribute.String("provider.name", provider),
		attribute.String("provider.operation", operation),
	}
	m.cacheHitRate.Add(context.TODO(), 1, metric.WithAttributes(attrs...))
}

// RecordCacheMiss records a cache miss for a provider.
func (m *ProviderMetrics) RecordCacheMiss(provider, operation string) {
	attrs := []attribute.KeyValue{
		attribute.String("provider.name", provider),
		attribute.String("provider.operation", operation),
	}
	m.cacheMissRate.Add(context.TODO(), 1, metric.WithAttributes(attrs...))
}
