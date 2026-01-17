# Observability Guide

This guide explains how to use the observability stack for BreatheRoute, including logs, traces, and metrics.

## Overview

BreatheRoute uses OpenTelemetry for distributed tracing and metrics collection. The local development stack includes:

| Component | Purpose | Local URL |
|-----------|---------|-----------|
| **Structured Logs** | JSON logs to stdout | Terminal |
| **Jaeger** | Distributed tracing UI | http://localhost:16686 |
| **Prometheus** | Metrics collection & queries | http://localhost:9090 |
| **Grafana** | Dashboards & visualization | http://localhost:3000 |
| **OTel Collector** | Receives telemetry from the app | localhost:4317 (gRPC) |

## Quick Start

### 1. Start the Observability Stack

```bash
# Start Jaeger, Prometheus, Grafana, and OTel Collector
make dev-observability
```

### 2. Start the API with Telemetry Enabled

```bash
# Option A: Use .env file (recommended)
cp .env.example .env  # OTEL_ENABLED=true is already set
make run

# Option B: Inline environment variables
OTEL_ENABLED=true OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317 make run
```

### 3. Generate Some Traffic

```bash
# Health check
curl http://localhost:8080/v1/ops/health

# Dev login (if AUTH_DEV_MODE=true)
curl -X POST http://localhost:8080/v1/auth/dev

# Create a commute (with token from dev login)
curl -X POST http://localhost:8080/v1/me/commutes \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"label": "Test", "origin": {"point": {"lat": 52.37, "lon": 4.89}}, "destination": {"point": {"lat": 52.31, "lon": 4.76}}, "preferredArrivalTimeLocal": "09:00"}'
```

### 4. View the Data

- **Traces**: http://localhost:16686 (Jaeger)
- **Metrics**: http://localhost:9090 (Prometheus)
- **Dashboards**: http://localhost:3000 (Grafana - admin/localdev)

---

## Logs

Logs are written to stdout in structured JSON format. Each log entry includes:

```json
{
  "level": "info",
  "service": "breatheroute-api",
  "version": "dev",
  "request_id": "req_abc123",
  "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736",
  "span_id": "00f067aa0ba902b7",
  "method": "GET",
  "path": "/v1/ops/health",
  "status": 200,
  "duration": 0.5,
  "time": "2024-01-17T10:00:00Z",
  "message": "request completed"
}
```

### Key Fields

| Field | Description |
|-------|-------------|
| `request_id` | Unique ID for the request (also in response header `X-Request-ID`) |
| `trace_id` | OpenTelemetry trace ID for distributed tracing |
| `span_id` | OpenTelemetry span ID |
| `duration` | Request duration in milliseconds |

### Filtering Logs

```bash
# View only errors
make run 2>&1 | jq 'select(.level == "error")'

# View requests to a specific path
make run 2>&1 | jq 'select(.path | startswith("/v1/me"))'

# View slow requests (>100ms)
make run 2>&1 | jq 'select(.duration > 100)'
```

---

## Traces (Jaeger)

Jaeger provides a UI for viewing distributed traces, showing the flow of requests through the system.

### Accessing Jaeger

1. Open http://localhost:16686
2. Select **Service**: `breatheroute-api`
3. Click **Find Traces**

### Understanding Traces

Each trace shows:
- **Timeline**: Visual representation of spans and their duration
- **Spans**: Individual operations within the request
- **Tags**: Metadata like HTTP method, status code, user ID
- **Logs**: Events that occurred during the span

### Example: Tracing a Request

1. Make a request:
   ```bash
   curl -v http://localhost:8080/v1/ops/health
   ```

2. Note the `X-Request-ID` header in the response

3. In Jaeger, search by the trace ID or find it in the recent traces

### Trace Propagation

Traces are automatically propagated via HTTP headers:
- `traceparent`: W3C Trace Context
- `tracestate`: Additional trace state

To correlate traces across services, pass these headers in outgoing requests.

---

## Metrics (Prometheus)

Prometheus collects and stores time-series metrics from the application.

### Accessing Prometheus

1. Open http://localhost:9090
2. Use the query box to explore metrics

### Available Metrics

#### HTTP Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `breatheroute_http_requests_total` | Counter | Total HTTP requests |
| `breatheroute_http_request_duration_seconds` | Histogram | Request latency |
| `breatheroute_http_request_size_bytes` | Histogram | Request body size |
| `breatheroute_http_response_size_bytes` | Histogram | Response body size |

#### Example Queries

```promql
# Request rate (requests per second)
rate(breatheroute_http_requests_total[5m])

# 95th percentile latency
histogram_quantile(0.95, rate(breatheroute_http_request_duration_seconds_bucket[5m]))

# Error rate (5xx responses)
sum(rate(breatheroute_http_requests_total{status=~"5.."}[5m])) / sum(rate(breatheroute_http_requests_total[5m]))

# Requests by endpoint
sum by (path) (rate(breatheroute_http_requests_total[5m]))

# Slow endpoints (avg latency > 100ms)
(
  sum by (path) (rate(breatheroute_http_request_duration_seconds_sum[5m]))
  /
  sum by (path) (rate(breatheroute_http_request_duration_seconds_count[5m]))
) > 0.1
```

---

## Dashboards (Grafana)

Grafana provides visualization and dashboards for metrics.

### Accessing Grafana

1. Open http://localhost:3000
2. Login with **admin** / **localdev**

### Pre-configured Data Sources

- **Prometheus**: For metrics queries
- **Jaeger**: For trace exploration

### Pre-built Dashboard

A **BreatheRoute API** dashboard is automatically provisioned with:

| Panel | Description |
|-------|-------------|
| Request Rate | Requests per second |
| Error Rate (5xx) | Percentage of server errors |
| P95 / P50 Latency | Response time percentiles |
| Request Rate by Endpoint | Traffic breakdown by path |
| Request Rate by Status | Traffic breakdown by HTTP status |
| Latency Percentiles | p50, p90, p95, p99 over time |
| P95 Latency by Endpoint | Slowest endpoints |
| 5xx / 4xx Errors | Error breakdown by endpoint |
| Rate Limited Requests | 429 responses from rate limiting |

Find it under **Dashboards** → **BreatheRoute API**

### Creating a Dashboard

1. Click **+** → **New Dashboard**
2. Add a panel
3. Select **Prometheus** as the data source
4. Enter a PromQL query
5. Configure visualization options

### Example Panel: Request Rate

```promql
sum(rate(breatheroute_http_requests_total[1m])) by (path)
```

Visualization: Time series graph

### Example Panel: Latency Heatmap

```promql
sum(rate(breatheroute_http_request_duration_seconds_bucket[1m])) by (le)
```

Visualization: Heatmap

---

## Troubleshooting

### Traces Not Appearing in Jaeger

1. Check that `OTEL_ENABLED=true` is set
2. Verify the OTel Collector is running:
   ```bash
   docker compose ps otel-collector
   ```
3. Check collector logs:
   ```bash
   docker compose logs otel-collector
   ```

### Metrics Not Appearing in Prometheus

1. Check Prometheus targets: http://localhost:9090/targets
2. Verify the `breatheroute` target is UP
3. Check OTel Collector metrics endpoint:
   ```bash
   curl http://localhost:8889/metrics
   ```

### Connection Refused to localhost:4317

The OTel Collector isn't running. Start it with:
```bash
make dev-observability
```

### High Memory Usage

The OTel Collector batches data before sending. If you're generating a lot of traffic, you may need to adjust batch settings in `config/otel-collector-config.yaml`.

---

## Production Considerations

In production (Cloud Run), observability is handled differently:

| Component | Production Setup |
|-----------|------------------|
| **Logs** | Cloud Logging (automatic) |
| **Traces** | Cloud Trace (via OTel exporter) |
| **Metrics** | Cloud Monitoring (via OTel exporter) |

The `OTEL_EXPORTER_OTLP_ENDPOINT` should point to the Google Cloud collector endpoint, and authentication is handled via workload identity.

---

## Stopping the Stack

```bash
# Stop all development containers including observability
make dev-down

# Or stop and remove volumes (fresh start)
make dev-clean
```
