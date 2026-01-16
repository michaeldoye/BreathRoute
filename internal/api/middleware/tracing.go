package middleware

import (
	"fmt"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

const tracerName = "github.com/breatheroute/breatheroute/internal/api/middleware"

// Tracing returns a middleware that creates spans for HTTP requests.
// It propagates trace context from incoming requests and adds span attributes.
func Tracing(serviceName string) func(http.Handler) http.Handler {
	tracer := otel.Tracer(tracerName)
	propagator := otel.GetTextMapPropagator()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract trace context from incoming request headers
			ctx := propagator.Extract(r.Context(), propagation.HeaderCarrier(r.Header))

			// Create span name from method and route pattern
			spanName := fmt.Sprintf("%s %s", r.Method, r.URL.Path)

			// Start span with semantic conventions
			ctx, span := tracer.Start(ctx, spanName,
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(
					semconv.HTTPMethod(r.Method),
					semconv.HTTPURL(r.URL.String()),
					semconv.HTTPRoute(r.URL.Path),
					semconv.HTTPScheme(scheme(r)),
					semconv.HTTPTarget(r.URL.RequestURI()),
					semconv.NetHostName(r.Host),
					semconv.UserAgentOriginal(r.UserAgent()),
					attribute.String("http.remote_addr", r.RemoteAddr),
				),
			)
			defer span.End()

			// Add request ID to span if available
			if requestID := GetRequestID(ctx); requestID != "" {
				span.SetAttributes(attribute.String("request.id", requestID))
			}

			// Wrap response writer to capture status code
			wrapped := newTracingResponseWriter(w)

			// Process request with trace context
			next.ServeHTTP(wrapped, r.WithContext(ctx))

			// Record response attributes
			span.SetAttributes(
				semconv.HTTPStatusCode(wrapped.statusCode),
				attribute.Int64("http.response_size", wrapped.written),
			)

			// Mark span as error if status >= 500
			if wrapped.statusCode >= 500 {
				span.SetStatus(codes.Error, http.StatusText(wrapped.statusCode))
			}
		})
	}
}

// tracingResponseWriter wraps http.ResponseWriter to capture response metadata.
type tracingResponseWriter struct {
	http.ResponseWriter
	statusCode int
	written    int64
}

func newTracingResponseWriter(w http.ResponseWriter) *tracingResponseWriter {
	return &tracingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
}

func (rw *tracingResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *tracingResponseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.written += int64(n)
	return n, err
}

// scheme returns the request scheme.
func scheme(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	if scheme := r.Header.Get("X-Forwarded-Proto"); scheme != "" {
		return scheme
	}
	return "http"
}

