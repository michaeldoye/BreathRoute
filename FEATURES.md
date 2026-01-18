# BreatheRoute Feature Documentation

This document tracks all implemented features, their intended purpose, and how they work. It serves as a living reference for the development team.

---

## Table of Contents

- [API Service](#api-service-ticket-2010)
- [Observability](#observability-ticket-2007)
- [Authentication](#authentication-ticket-2008)
- [API Security Controls](#api-security-controls-ticket-2013)
- [Air Quality Provider](#air-quality-provider-ticket-2021)
- [Weather Provider](#weather-provider-ticket-2022)
- [Pollen Provider](#pollen-provider-ticket-2023)
- [Transit Provider](#transit-provider-ticket-2024)
- [Provider Resilience](#provider-resilience-ticket-2025)
- [Background Refresh Job](#background-refresh-job-ticket-2026)
- [Planned Features](#planned-features)

---

## API Service (Ticket 2010)

**Status**: ✅ Complete
**Purpose**: Provide a production-ready HTTP API that implements the BreatheRoute OpenAPI specification, enabling mobile clients to interact with air quality routing services.

### Overview

The API service is built with Go 1.24 and the Chi router, following RESTful conventions. It serves as the primary interface for the iOS app to manage user profiles, commutes, alerts, and route computations.

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

#### Graceful Shutdown

| Aspect | Details |
|--------|---------|
| **Purpose** | Allow in-flight requests to complete before the server stops, preventing data loss |
| **How it works** | Listens for SIGINT/SIGTERM signals. When received, stops accepting new connections, waits up to 30 seconds for existing requests to finish, then exits cleanly. |
| **Location** | `cmd/api/main.go` |

### API Endpoints

| Category | Endpoints | Purpose |
|----------|-----------|---------|
| **Ops** | `/v1/ops/health`, `/ready`, `/status` | Health monitoring and Kubernetes probes |
| **Auth** | `/v1/auth/siwa`, `/refresh`, `/logout`, `/logout-all` | Authentication with Sign in with Apple |
| **User** | `/v1/me`, `/v1/me/consents`, `/v1/me/profile` | User info and preferences |
| **Commutes** | `/v1/me/commutes/*` | CRUD for saved commutes |
| **Alerts** | `/v1/me/alerts/subscriptions/*` | Push notification subscriptions |
| **Devices** | `/v1/me/devices/*` | Device registration for push |
| **Routes** | `/v1/routes:compute` | Route calculation with air quality |
| **Alerts Preview** | `/v1/alerts/preview` | Departure time recommendations |
| **GDPR** | `/v1/gdpr/export-requests/*`, `/v1/gdpr/deletion-requests/*` | Data portability and deletion |
| **Metadata** | `/v1/metadata/enums`, `/v1/metadata/air-quality/stations` | Reference data |
| **Admin** | `/v1/admin/feature-flags/*` | Feature flag management |

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

#### Distributed Tracing

| Aspect | Details |
|--------|---------|
| **Purpose** | Track requests across multiple services and identify latency bottlenecks |
| **How it works** | Creates a span for each HTTP request with W3C Trace Context propagation. Spans include HTTP semantic conventions and custom attributes. Child spans can be created for database calls, external API requests, etc. |
| **Location** | `internal/api/middleware/tracing.go`, `internal/telemetry/telemetry.go` |

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

---

## Authentication (Ticket 2008)

**Status**: ✅ Complete
**Purpose**: Secure API access using Sign in with Apple as the primary authentication method, with JWT access tokens and refresh token rotation.

### Overview

Authentication uses Apple's identity tokens verified server-side. Upon successful verification, the API issues short-lived JWT access tokens (1 hour) and long-lived refresh tokens (30 days).

### Features

#### Sign in with Apple (SIWA)

| Aspect | Details |
|--------|---------|
| **Purpose** | Privacy-focused authentication without email/password |
| **How it works** | iOS app obtains identity token from Apple. Backend verifies signature using Apple's public keys, validates claims, and creates/retrieves user. |
| **Location** | `internal/auth/siwa.go` |

**Flow**:
```
┌──────────────┐     Identity Token      ┌──────────────┐
│   iOS App    │ ──────────────────────► │  API Server  │
└──────────────┘                         └──────┬───────┘
                                                │
                                    ┌───────────▼───────────┐
                                    │ 1. Fetch Apple JWKS   │
                                    │ 2. Verify signature   │
                                    │ 3. Validate claims    │
                                    │ 4. Find/create user   │
                                    │ 5. Issue tokens       │
                                    └───────────────────────┘
```

#### JWT Access Tokens

| Aspect | Details |
|--------|---------|
| **Purpose** | Stateless authentication for API requests |
| **How it works** | HS256-signed tokens with 1-hour TTL. Contains user ID and issued-at time. Validated on every authenticated request. |
| **Location** | `internal/auth/jwt.go` |

#### Refresh Token Rotation

| Aspect | Details |
|--------|---------|
| **Purpose** | Long-lived sessions with revocation capability |
| **How it works** | Opaque tokens stored in database. When refreshed, old token is revoked and new token issued. Supports logout-all for security. |
| **Location** | `internal/auth/service.go` |

---

## API Security Controls (Ticket 2013)

**Status**: ✅ Complete
**Purpose**: Protect public endpoints against abuse and accidental overload with rate limiting, security headers, and TLS enforcement.

### Features

#### Security Headers Middleware

| Aspect | Details |
|--------|---------|
| **Purpose** | Protect against common web vulnerabilities |
| **How it works** | Sets standard security headers on all responses |
| **Location** | `internal/api/middleware/security.go` |

**Headers Set**:
| Header | Value | Purpose |
|--------|-------|---------|
| `X-Content-Type-Options` | `nosniff` | Prevent MIME sniffing |
| `X-Frame-Options` | `DENY` | Prevent clickjacking |
| `Strict-Transport-Security` | `max-age=31536000; includeSubDomains` | Force HTTPS |
| `Content-Security-Policy` | `default-src 'none'` | Restrict resources |
| `Referrer-Policy` | `strict-origin-when-cross-origin` | Control referrer |
| `Permissions-Policy` | `geolocation=(), camera=(), microphone=()` | Disable features |

#### Rate Limiting

| Aspect | Details |
|--------|---------|
| **Purpose** | Prevent abuse and ensure fair usage |
| **How it works** | In-memory rate limiting using go-chi/httprate. Per-IP for public endpoints, per-user for authenticated endpoints. |
| **Location** | `internal/api/middleware/ratelimit.go` |

**Rate Limit Tiers**:
| Endpoint Category | Per-IP Limit | Per-User Limit |
|-------------------|--------------|----------------|
| Auth endpoints (`/auth/*`) | 10/min | - |
| Expensive compute (`/routes:compute`) | 30/min | - |
| Standard endpoints | 100/min | 100/min |

**Response on Rate Limit**:
```json
{
  "type": "https://api.breatheroute.com/problems/too-many-requests",
  "title": "Too many requests",
  "status": 429,
  "detail": "Rate limit exceeded. Please retry after 60 seconds.",
  "retryAfter": 60
}
```

#### TLS Enforcement

| Aspect | Details |
|--------|---------|
| **Purpose** | Ensure all traffic is encrypted |
| **How it works** | Middleware checks `X-Forwarded-Proto` header (set by Cloud Run). Returns 421 if not HTTPS. Disabled for local development. |
| **Location** | `internal/api/middleware/security.go` |

---

## Air Quality Provider (Ticket 2021)

**Status**: ✅ Complete
**Purpose**: Integrate with Luchtmeetnet API to fetch real-time air quality measurements from Dutch monitoring stations.

### Overview

The air quality service fetches station data and measurements from Luchtmeetnet (RIVM's air quality monitoring network). Data is cached with TTL and supports stale-if-error for resilience.

### Features

#### Luchtmeetnet Client

| Aspect | Details |
|--------|---------|
| **Purpose** | Fetch air quality data from official Dutch monitoring network |
| **How it works** | HTTP client with circuit breaker protection. Fetches station list and current measurements. Maps API response to domain models. |
| **Location** | `internal/airquality/luchtmeetnet/client.go` |

**Data Fetched**:
- Station metadata (ID, name, location, pollutants measured)
- Current measurements (NO₂, PM2.5, PM10, O₃)
- Measurement timestamps

#### Air Quality Service

| Aspect | Details |
|--------|---------|
| **Purpose** | Provide cached, resilient access to air quality data |
| **How it works** | Wraps Luchtmeetnet client with TTL caching (5 min), stale-if-error (30 min), and snapshot generation for routing. |
| **Location** | `internal/airquality/service.go` |

**Snapshot Structure**:
```go
type Snapshot struct {
    Stations    []Station    // All stations with latest readings
    GeneratedAt time.Time    // When snapshot was created
    TTL         time.Duration // Cache validity
}
```

---

## Weather Provider (Ticket 2022)

**Status**: ✅ Complete
**Purpose**: Integrate with OpenWeatherMap to provide current weather conditions that affect commute comfort and exposure.

### Features

#### OpenWeatherMap Client

| Aspect | Details |
|--------|---------|
| **Purpose** | Fetch current weather for route endpoints |
| **How it works** | Uses resilient HTTP client with circuit breaker. Fetches temperature, humidity, wind, and weather conditions. |
| **Location** | `internal/weather/openweathermap/client.go` |

**Data Fetched**:
- Temperature (°C)
- Humidity (%)
- Wind speed and direction
- Weather conditions (rain, clear, etc.)

#### Weather Service

| Aspect | Details |
|--------|---------|
| **Purpose** | Provide cached weather data for routes |
| **How it works** | 5-minute TTL caching with stale-if-error. Weather affects exposure scoring (rain reduces PM dispersion, etc.). |
| **Location** | `internal/weather/service.go` |

---

## Pollen Provider (Ticket 2023)

**Status**: ✅ Complete
**Purpose**: Integrate with Ambee API to provide pollen forecasts for users with allergies.

### Features

#### Ambee Pollen Client

| Aspect | Details |
|--------|---------|
| **Purpose** | Fetch pollen levels and forecasts |
| **How it works** | Resilient HTTP client fetching regional pollen data. Maps Ambee risk levels to domain risk categories. |
| **Location** | `internal/pollen/ambee/client.go` |

**Data Fetched**:
- Tree pollen levels
- Grass pollen levels
- Weed pollen levels
- Overall risk level (LOW, MEDIUM, HIGH, VERY_HIGH)

#### Feature Flag Control

| Aspect | Details |
|--------|---------|
| **Purpose** | Disable pollen feature without deployment |
| **How it works** | `disable_pollen` feature flag controls whether pollen data is fetched. When disabled, pollen weight is redistributed to other factors. |
| **Location** | `internal/pollen/service.go` |

---

## Transit Provider (Ticket 2024)

**Status**: ✅ Complete
**Purpose**: Integrate with NS API to provide train disruption information affecting commutes.

### Features

#### NS API Client

| Aspect | Details |
|--------|---------|
| **Purpose** | Fetch current train disruptions and station info |
| **How it works** | Resilient HTTP client calling NS disruption and station endpoints. Maps Dutch/English disruption types to domain models. Generates advisory messages based on severity. |
| **Location** | `internal/transit/ns/client.go` |

**Disruption Types**:
| Type | Impact | Description |
|------|--------|-------------|
| `MAINTENANCE` | MINOR | Planned maintenance work |
| `DISRUPTION` | MAJOR | Unplanned service disruption |
| `CALAMITY` | SEVERE | Major incident |

#### Transit Service

| Aspect | Details |
|--------|---------|
| **Purpose** | Provide cached disruption data |
| **How it works** | 5-minute TTL caching. Supports route-specific disruption queries. Feature flag `disable_transit` for emergency disable. |
| **Location** | `internal/transit/service.go` |

**Advisory Messages**:
```
MINOR:  "Minor delays possible. Allow extra time."
MAJOR:  "Consider alternative transport options."
SEVERE: "Service suspended. Use alternative transport."
```

---

## Provider Resilience (Ticket 2025)

**Status**: ✅ Complete
**Purpose**: Protect against external API failures with circuit breakers, retry logic, and health monitoring.

### Features

#### Circuit Breaker

| Aspect | Details |
|--------|---------|
| **Purpose** | Fail fast when external services are down |
| **How it works** | Opens after 50% failure rate (minimum 5 requests). Half-open state allows probe requests. Automatically closes after success. |
| **Location** | `internal/provider/resilience/circuit_breaker.go` |

**States**:
```
                  success
    ┌──────────────────────────────┐
    │                              │
    ▼                              │
┌────────┐   failure   ┌────────┐  │   ┌────────────┐
│ CLOSED │────────────►│  OPEN  │──┴──►│ HALF-OPEN  │
└────────┘  threshold  └────────┘      └────────────┘
    ▲                      │ timeout       │
    │                      ▼               │
    │                  probe request       │
    │                      │               │
    └──────────────────────┴───────────────┘
                 success
```

#### Exponential Backoff Retry

| Aspect | Details |
|--------|---------|
| **Purpose** | Retry transient failures with increasing delays |
| **How it works** | Retries 5xx and network errors. Initial delay 100ms, max 5s. Maximum 3 attempts. Uses cenkalti/backoff. |
| **Location** | `internal/provider/resilience/client.go` |

#### Provider Health Registry

| Aspect | Details |
|--------|---------|
| **Purpose** | Track and expose provider health status |
| **How it works** | Central registry tracks circuit breaker state, last success/failure times, and error messages. Exposed via `/v1/ops/status` endpoint. |
| **Location** | `internal/provider/resilience/registry.go` |

**System Status Response**:
```json
{
  "status": "ok",
  "providers": [
    {
      "provider": "luchtmeetnet",
      "status": "ok",
      "lastSuccessAt": "2024-01-15T12:00:00Z"
    },
    {
      "provider": "ns-api",
      "status": "degraded",
      "lastFailureAt": "2024-01-15T11:55:00Z",
      "message": "connection timeout"
    }
  ]
}
```

---

## Background Refresh Job (Ticket 2026)

**Status**: ✅ Complete
**Purpose**: Keep provider caches warm by proactively refreshing data for major Dutch cities.

### Overview

A background worker process refreshes air quality, weather, pollen, and transit data on a schedule. This ensures low-latency responses for users in the Randstad metropolitan area.

### Features

#### Refresh Configuration

| Aspect | Details |
|--------|---------|
| **Purpose** | Define which locations to pre-cache |
| **How it works** | Configurable list of cities with priority levels. Each city has multiple points (e.g., city center, train station). |
| **Location** | `internal/worker/config.go` |

**Default Targets**:
| City | Priority | Points |
|------|----------|--------|
| Amsterdam | 1 | Centraal, Zuid, Zuidoost, Noord |
| Rotterdam | 1 | Centraal, Zuid, West |
| Den Haag | 1 | Centraal, HS, Scheveningen |
| Utrecht | 1 | Centraal, Science Park |
| Eindhoven | 2 | Centraal, High Tech Campus |
| Schiphol | 2 | Airport |
| Leiden, Haarlem, Delft, Amersfoort | 3 | Centraal |

#### Concurrent Processing

| Aspect | Details |
|--------|---------|
| **Purpose** | Refresh all points efficiently |
| **How it works** | Worker pool with configurable concurrency (default: 3). Points processed in priority order. Per-point timeout prevents blocking. |
| **Location** | `internal/worker/refresh.go` |

#### Pub/Sub Integration

| Aspect | Details |
|--------|---------|
| **Purpose** | Trigger refresh jobs via Cloud Scheduler |
| **How it works** | Worker subscribes to Pub/Sub topic. Scheduler publishes refresh messages on schedule. Supports health check messages. |
| **Location** | `internal/worker/pubsub.go` |

**Message Types**:
| Job Type | Purpose |
|----------|---------|
| `provider_refresh` | Full refresh of all configured points |
| `health_check` | Single-point refresh to verify connectivity |

#### Refresh Metrics

| Aspect | Details |
|--------|---------|
| **Purpose** | Monitor refresh job performance |
| **How it works** | Tracks total refreshes, success/failure counts, duration, cache hits/misses, and per-provider statistics. |
| **Location** | `internal/worker/refresh.go` |

**Metrics Available**:
```json
{
  "total_refreshes": 100,
  "successful_refreshes": 95,
  "failed_refreshes": 5,
  "airquality_refreshes": 100,
  "weather_refreshes": 100,
  "pollen_refreshes": 98,
  "transit_refreshes": 50,
  "last_refresh_at": "2024-01-15T12:00:00Z",
  "last_refresh_duration": "5.2s",
  "cache_hits": 50,
  "cache_misses": 150
}
```

---

## Planned Features

Features in the backlog that are not yet implemented:

| Ticket | Feature | Description |
|--------|---------|-------------|
| 2011 | Route Engine | Multi-modal route calculation with exposure scoring |
| 2012 | Push Notifications | APNs integration for departure alerts |
| 2014 | Database Layer | PostgreSQL with PostGIS for spatial queries |

---

## Test Coverage

All features include comprehensive test coverage:

| Package | Tests | Coverage |
|---------|-------|----------|
| `internal/api` | 22 | Router integration tests |
| `internal/api/middleware` | 35+ | Middleware unit tests |
| `internal/api/models` | 12 | Model and Problem tests |
| `internal/auth` | 20+ | Auth service and SIWA tests |
| `internal/airquality` | 15+ | Air quality service tests |
| `internal/weather` | 10+ | Weather service tests |
| `internal/pollen` | 10+ | Pollen service tests |
| `internal/transit` | 12+ | Transit service tests |
| `internal/provider/resilience` | 15+ | Circuit breaker and client tests |
| `internal/worker` | 18 | Refresh job tests |
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
| `cmd/worker/main.go` | Worker service entrypoint |
| `internal/api/router.go` | Chi router configuration |
| `internal/api/middleware/*.go` | Request processing middleware |
| `internal/api/handler/*.go` | HTTP handlers |
| `internal/api/models/*.go` | Request/response models |
| `internal/auth/*.go` | Authentication services |
| `internal/airquality/*.go` | Air quality service |
| `internal/weather/*.go` | Weather service |
| `internal/pollen/*.go` | Pollen service |
| `internal/transit/*.go` | Transit service |
| `internal/provider/resilience/*.go` | Resilient HTTP client |
| `internal/worker/*.go` | Background job processing |
| `internal/featureflags/*.go` | Feature flag management |
| `internal/telemetry/*.go` | OpenTelemetry initialization |

---

*Last updated: January 2026*
