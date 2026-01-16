# BreatheRoute Feature Documentation

This document tracks all implemented features, their intended purpose, and how they work. It serves as a living reference for the development team.

---

## Table of Contents

- [API Service](#api-service-ticket-2010)
- [Observability](#observability-ticket-2007)
- [Planned Features](#planned-features)

---

## API Service (Ticket 2010)

**Status**: ✅ Complete
**Purpose**: Provide a production-ready HTTP API that implements the BreatheRoute OpenAPI specification, enabling mobile clients to interact with air quality routing services.

### Overview

The API service is built with Go 1.22 and the Chi router, following RESTful conventions. It serves as the primary interface for the iOS app to manage user profiles, commutes, alerts, and route computations.

### Features

#### Chi Router with Versioned Endpoints

| Aspect | Details |
|--------|---------|
| **Purpose** | Provide a stable, versioned API that allows backward-compatible evolution |
| **How it works** | All endpoints are prefixed with `/v1/`. When breaking changes are needed, a `/v2/` version can be added while maintaining the old version. Chi router provides fast HTTP routing with middleware support. |
| **Location** | `internal/api/router.go` |

```go
// Example: Adding routes with Chi
r.Route("/v1", func(r chi.Router) {
    r.Get("/ops/health", opsHandler.HealthCheck)
    r.Route("/me", func(r chi.Router) {
        r.Get("/", meHandler.GetMe)
        r.Get("/profile", profileHandler.GetProfile)
    })
})
```

#### Request ID Middleware

| Aspect | Details |
|--------|---------|
| **Purpose** | Enable request tracing across distributed systems and correlate logs/errors to specific requests |
| **How it works** | Generates a unique `req_<uuid>` identifier for each request. If the client sends `X-Request-Id` header, that value is preserved. The ID is stored in the request context and returned in the response header. |
| **Location** | `internal/api/middleware/request_id.go` |

```go
// Extracting request ID in handlers
requestID := middleware.GetRequestID(r.Context())
log.Info().Str("request_id", requestID).Msg("processing request")
```

**Request Flow**:
```
Client Request
     │
     ▼
┌────────────────────────┐
│ Check X-Request-Id     │
│ header exists?         │
└────────────┬───────────┘
             │
     ┌───────┴───────┐
     │ Yes           │ No
     ▼               ▼
┌──────────┐  ┌──────────────┐
│ Use      │  │ Generate     │
│ existing │  │ req_<uuid>   │
└────┬─────┘  └──────┬───────┘
     │               │
     └───────┬───────┘
             ▼
┌────────────────────────┐
│ Store in context       │
│ Set response header    │
└────────────────────────┘
```

#### RFC 7807 Problem+JSON Error Responses

| Aspect | Details |
|--------|---------|
| **Purpose** | Provide consistent, machine-readable error responses that include debugging information |
| **How it works** | All errors return `Content-Type: application/problem+json` with standardized fields: `type`, `title`, `status`, `detail`, `traceId`, and optionally `errors` for validation failures. |
| **Location** | `internal/api/models/problem.go` |

**Example Response**:
```json
{
  "type": "https://api.breatheroute.com/problems/validation",
  "title": "Validation error",
  "status": 400,
  "detail": "Request body contains invalid fields",
  "traceId": "req_a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "instance": "/v1/routes:compute",
  "errors": [
    {
      "field": "origin.lat",
      "message": "must be between -90 and 90",
      "code": "OUT_OF_RANGE"
    }
  ]
}
```

**Problem Types**:
| Type | HTTP Status | Use Case |
|------|-------------|----------|
| `validation` | 400 | Invalid request data |
| `unauthorized` | 401 | Missing/invalid auth |
| `not_found` | 404 | Resource doesn't exist |
| `conflict` | 409 | Duplicate resource |
| `too_many_requests` | 429 | Rate limit exceeded |
| `internal` | 500 | Server error |
| `unavailable` | 503 | Service temporarily down |

#### Panic Recovery Middleware

| Aspect | Details |
|--------|---------|
| **Purpose** | Prevent crashes from panics in handlers and return graceful error responses |
| **How it works** | Wraps all handlers in a recover block. If a panic occurs, it logs the stack trace, returns a 500 Problem+JSON response, and keeps the server running. |
| **Location** | `internal/api/middleware/recovery.go` |

```go
// Panic in handler
func (h *Handler) DoSomething(w http.ResponseWriter, r *http.Request) {
    panic("unexpected error") // Would crash server without recovery
}

// With recovery middleware, client receives:
// HTTP 500
// {"type":"...internal","title":"Internal server error","traceId":"req_..."}
```

#### Graceful Shutdown

| Aspect | Details |
|--------|---------|
| **Purpose** | Allow in-flight requests to complete before the server stops, preventing data loss |
| **How it works** | Listens for SIGINT/SIGTERM signals. When received, stops accepting new connections, waits up to 30 seconds for existing requests to finish, then exits cleanly. |
| **Location** | `cmd/api/main.go` |

```go
// Shutdown sequence
quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
<-quit  // Block until signal received

ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

server.Shutdown(ctx)  // Wait for in-flight requests
```

### API Endpoints

| Category | Endpoints | Purpose |
|----------|-----------|---------|
| **Ops** | `/v1/ops/health`, `/ready`, `/status` | Health monitoring and Kubernetes probes |
| **User** | `/v1/me`, `/v1/me/consents`, `/v1/me/profile` | User info and preferences |
| **Commutes** | `/v1/me/commutes/*` | CRUD for saved commutes |
| **Alerts** | `/v1/me/alerts/subscriptions/*` | Push notification subscriptions |
| **Devices** | `/v1/me/devices/*` | Device registration for push |
| **Routes** | `/v1/routes:compute` | Route calculation with air quality |
| **Alerts Preview** | `/v1/alerts/preview` | Departure time recommendations |
| **GDPR** | `/v1/gdpr/export-requests/*`, `/v1/gdpr/deletion-requests/*` | Data portability and deletion |
| **Metadata** | `/v1/metadata/enums`, `/v1/metadata/air-quality/stations` | Reference data |

---

## Observability (Ticket 2007)

**Status**: ✅ Complete
**Purpose**: Enable production debugging from day one with distributed tracing, structured logging, and metrics collection via OpenTelemetry.

### Overview

The observability stack uses OpenTelemetry for vendor-agnostic instrumentation. Traces and metrics can be exported to any OTLP-compatible backend (Jaeger, Grafana Tempo, Datadog, etc.).

### Features

#### Structured JSON Logging

| Aspect | Details |
|--------|---------|
| **Purpose** | Enable log aggregation, searching, and correlation in production |
| **How it works** | Every log entry is JSON with consistent fields: `request_id`, `trace_id`, `span_id`, `method`, `path`, `status`, `duration`. Zerolog provides zero-allocation logging for high performance. |
| **Location** | `internal/api/middleware/logging.go` |

**Example Log Entry**:
```json
{
  "level": "info",
  "time": "2024-01-15T08:30:00Z",
  "service": "breatheroute-api",
  "version": "1.0.0",
  "request_id": "req_a1b2c3d4",
  "trace_id": "0af7651916cd43dd8448eb211c80319c",
  "span_id": "b7ad6b7169203331",
  "method": "POST",
  "path": "/v1/routes:compute",
  "status": 200,
  "bytes": 1234,
  "duration": 45.2,
  "remote_addr": "10.0.0.1",
  "user_agent": "BreatheRoute/1.0 iOS/17.0",
  "message": "request completed"
}
```

**Correlation**:
- `request_id` - Links logs to a specific API request
- `trace_id` - Links to distributed trace (same across services)
- `span_id` - Links to specific span in trace

#### Distributed Tracing

| Aspect | Details |
|--------|---------|
| **Purpose** | Track requests across multiple services and identify latency bottlenecks |
| **How it works** | Creates a span for each HTTP request with W3C Trace Context propagation. Spans include HTTP semantic conventions and custom attributes. Child spans can be created for database calls, external API requests, etc. |
| **Location** | `internal/api/middleware/tracing.go`, `internal/telemetry/telemetry.go` |

**Trace Propagation**:
```
┌──────────────────────────────────────────────────────────────────┐
│                        Trace: 0af765...                          │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │ Span: API Request (45ms)                                    │ │
│  │ POST /v1/routes:compute                                     │ │
│  │                                                             │ │
│  │  ┌────────────────────────────┐  ┌────────────────────────┐ │ │
│  │  │ Span: DB Query (5ms)       │  │ Span: Air Quality (30ms)│ │
│  │  │ SELECT FROM routes         │  │ GET luchtmeetnet.nl    │ │
│  │  └────────────────────────────┘  └────────────────────────┘ │ │
│  └─────────────────────────────────────────────────────────────┘ │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

**Span Attributes**:
| Attribute | Description |
|-----------|-------------|
| `http.method` | GET, POST, etc. |
| `http.url` | Full request URL |
| `http.route` | URL path pattern |
| `http.status_code` | Response status |
| `http.response_size` | Response body size |
| `request.id` | BreatheRoute request ID |
| `error` | Set to true for 5xx responses |

**Context Propagation**:
```
Incoming Request Headers:
  traceparent: 00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01
                  └─────────────────┬────────────────────┘
                              trace-id

Outgoing Request Headers (to external services):
  traceparent: 00-0af7651916cd43dd8448eb211c80319c-NEW_SPAN_ID-01
```

#### HTTP Metrics

| Aspect | Details |
|--------|---------|
| **Purpose** | Monitor API performance, error rates, and capacity |
| **How it works** | Records metrics for every request using OpenTelemetry semantic conventions. Metrics are aggregated and exported to OTLP every 15 seconds. |
| **Location** | `internal/api/middleware/metrics.go` |

**Metrics Collected**:

| Metric | Type | Labels | Purpose |
|--------|------|--------|---------|
| `http.server.request.duration` | Histogram | method, route, status_code | Latency percentiles (p50, p95, p99) |
| `http.server.request.total` | Counter | method, route, status_code | Request rate, error rate |
| `http.server.requests_in_flight` | UpDownCounter | method, route | Current load, capacity planning |
| `http.server.response.size` | Histogram | method, route | Response size distribution |

**Example Prometheus Queries**:
```promql
# Request rate by endpoint
rate(http_server_request_total[5m])

# P99 latency
histogram_quantile(0.99, rate(http_server_request_duration_bucket[5m]))

# Error rate
sum(rate(http_server_request_total{status_code=~"5.."}[5m]))
  / sum(rate(http_server_request_total[5m]))

# Current load
sum(http_server_requests_in_flight)
```

#### Provider Metrics

| Aspect | Details |
|--------|---------|
| **Purpose** | Monitor external API dependencies (Luchtmeetnet, NS API, etc.) |
| **How it works** | Records latency and success/failure for each external call. Tracks cache hit/miss rates for data that's cached. |
| **Location** | `internal/api/middleware/metrics.go` |

**Metrics Collected**:

| Metric | Type | Labels | Purpose |
|--------|------|--------|---------|
| `provider.request.duration` | Histogram | provider.name, provider.operation | External API latency |
| `provider.request.total` | Counter | provider.name, provider.operation, error | External API reliability |
| `provider.cache.hit` | Counter | provider.name, provider.operation | Cache effectiveness |
| `provider.cache.miss` | Counter | provider.name, provider.operation | Cache misses |

**Usage Example**:
```go
pm, _ := middleware.NewProviderMetrics("luchtmeetnet")

// When making an external call
start := time.Now()
data, err := client.GetStations(ctx)
pm.RecordRequest(ctx, "luchtmeetnet", "get-stations", time.Since(start), err)

// When using cache
if cached {
    pm.RecordCacheHit("luchtmeetnet", "get-stations")
} else {
    pm.RecordCacheMiss("luchtmeetnet", "get-stations")
}
```

#### OTLP Export

| Aspect | Details |
|--------|---------|
| **Purpose** | Send telemetry data to observability backends |
| **How it works** | Exports traces and metrics via gRPC to any OTLP-compatible collector. Batches data for efficiency. Configurable via environment variables. |
| **Location** | `internal/telemetry/telemetry.go` |

**Configuration**:
| Variable | Default | Description |
|----------|---------|-------------|
| `OTEL_ENABLED` | `false` | Enable OTLP export |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `localhost:4317` | Collector address |
| `APP_ENV` | `development` | Environment tag |

**Architecture**:
```
┌────────────────────┐     gRPC     ┌─────────────────────┐
│   BreatheRoute     │─────────────►│   OTLP Collector    │
│   API Service      │              │  (e.g., OTel Agent) │
└────────────────────┘              └──────────┬──────────┘
                                               │
                    ┌──────────────────────────┼──────────────────────────┐
                    │                          │                          │
                    ▼                          ▼                          ▼
           ┌─────────────────┐       ┌─────────────────┐       ┌─────────────────┐
           │  Jaeger/Tempo   │       │   Prometheus    │       │    Grafana      │
           │   (Tracing)     │       │   (Metrics)     │       │  (Dashboards)   │
           └─────────────────┘       └─────────────────┘       └─────────────────┘
```

---

## Planned Features

Features in the backlog that are not yet implemented:

| Ticket | Feature | Description |
|--------|---------|-------------|
| 2008 | Authentication | JWT-based auth with Sign in with Apple |
| 2009 | Air Quality Provider | Integration with Luchtmeetnet API |
| 2011 | Route Engine | Multi-modal route calculation with exposure scoring |
| 2012 | Push Notifications | APNs integration for departure alerts |
| 2013 | Caching Layer | Redis caching for air quality and route data |
| 2014 | Database Layer | PostgreSQL with PostGIS for spatial queries |

---

## Test Coverage

All features include comprehensive test coverage:

| Package | Tests | Coverage |
|---------|-------|----------|
| `internal/api` | 22 | Router integration tests |
| `internal/api/middleware` | 23 | Middleware unit tests |
| `internal/api/models` | 12 | Model and Problem tests |
| `internal/telemetry` | 4 | Telemetry initialization tests |

Run tests with:
```bash
go test ./... -v
go test ./... -cover
```

---

## File Reference

| File | Purpose |
|------|---------|
| `cmd/api/main.go` | API server entrypoint with telemetry init |
| `internal/api/router.go` | Chi router configuration |
| `internal/api/middleware/request_id.go` | Request ID generation |
| `internal/api/middleware/logging.go` | Structured logging |
| `internal/api/middleware/tracing.go` | Distributed tracing spans |
| `internal/api/middleware/metrics.go` | HTTP and provider metrics |
| `internal/api/middleware/recovery.go` | Panic recovery |
| `internal/api/middleware/content_type.go` | JSON content type |
| `internal/api/models/problem.go` | RFC 7807 Problem responses |
| `internal/api/models/*.go` | Request/response models |
| `internal/api/handler/*.go` | HTTP handlers |
| `internal/api/response/response.go` | Response helpers |
| `internal/telemetry/telemetry.go` | OpenTelemetry initialization |

---

*Last updated: January 2024*
