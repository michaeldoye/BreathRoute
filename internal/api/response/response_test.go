package response_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/breatheroute/breatheroute/internal/api/middleware"
	"github.com/breatheroute/breatheroute/internal/api/models"
	"github.com/breatheroute/breatheroute/internal/api/response"
)

// requestWithContext creates an HTTP request that has been processed by the RequestID middleware
// to populate the context with a request ID.
func requestWithContext(t *testing.T, method, path string) (*http.Request, *httptest.ResponseRecorder) {
	t.Helper()
	req := httptest.NewRequest(method, path, http.NoBody)
	rec := httptest.NewRecorder()

	// Process through RequestID middleware to set up context
	var processedReq *http.Request
	handler := middleware.RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		processedReq = r
	}))
	handler.ServeHTTP(rec, req)

	// Reset the recorder for actual test use
	rec = httptest.NewRecorder()

	return processedReq, rec
}

func TestJSON_IncludesRequestID(t *testing.T) {
	req, rec := requestWithContext(t, http.MethodGet, "/test")

	response.JSON(rec, req, http.StatusOK, map[string]string{"message": "hello"})

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	requestID := rec.Header().Get("X-Request-Id")
	if requestID == "" {
		t.Error("expected X-Request-Id header to be set")
	}
	if len(requestID) < 10 {
		t.Errorf("expected request ID to be a valid ID, got %q", requestID)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", contentType)
	}
}

func TestJSON_WithoutRequestID(t *testing.T) {
	// Create request without middleware (no request ID in context)
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	rec := httptest.NewRecorder()

	response.JSON(rec, req, http.StatusOK, map[string]string{"message": "hello"})

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	// Should not have X-Request-Id if context doesn't have it
	requestID := rec.Header().Get("X-Request-Id")
	if requestID != "" {
		t.Errorf("expected no X-Request-Id header when not in context, got %q", requestID)
	}
}

func TestCreated_IncludesRequestIDAndLocation(t *testing.T) {
	req, rec := requestWithContext(t, http.MethodPost, "/test")

	response.Created(rec, req, "/v1/items/123", map[string]string{"id": "123"})

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", rec.Code)
	}

	requestID := rec.Header().Get("X-Request-Id")
	if requestID == "" {
		t.Error("expected X-Request-Id header to be set")
	}

	location := rec.Header().Get("Location")
	if location != "/v1/items/123" {
		t.Errorf("expected Location /v1/items/123, got %q", location)
	}
}

func TestNoContent_IncludesRequestID(t *testing.T) {
	req, rec := requestWithContext(t, http.MethodDelete, "/test")

	response.NoContent(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", rec.Code)
	}

	requestID := rec.Header().Get("X-Request-Id")
	if requestID == "" {
		t.Error("expected X-Request-Id header to be set")
	}

	// 204 should have no body
	if rec.Body.Len() != 0 {
		t.Errorf("expected empty body for 204, got %q", rec.Body.String())
	}
}

func TestAccepted_IncludesRequestIDAndLocation(t *testing.T) {
	req, rec := requestWithContext(t, http.MethodPost, "/test")

	response.Accepted(rec, req, "/v1/jobs/456", map[string]string{"status": "pending"})

	if rec.Code != http.StatusAccepted {
		t.Errorf("expected status 202, got %d", rec.Code)
	}

	requestID := rec.Header().Get("X-Request-Id")
	if requestID == "" {
		t.Error("expected X-Request-Id header to be set")
	}

	location := rec.Header().Get("Location")
	if location != "/v1/jobs/456" {
		t.Errorf("expected Location /v1/jobs/456, got %q", location)
	}
}

func TestTooManyRequests_IncludesRateLimitHeaders(t *testing.T) {
	req, rec := requestWithContext(t, http.MethodGet, "/test")

	info := &response.RateLimitInfo{
		Limit:      100,
		Remaining:  0,
		ResetAt:    1704067200,
		RetryAfter: 60,
	}
	response.TooManyRequestsWithInfo(rec, req, "rate limit exceeded", info)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("expected status 429, got %d", rec.Code)
	}

	if h := rec.Header().Get("X-RateLimit-Limit"); h != "100" {
		t.Errorf("expected X-RateLimit-Limit 100, got %q", h)
	}
	if h := rec.Header().Get("X-RateLimit-Remaining"); h != "0" {
		t.Errorf("expected X-RateLimit-Remaining 0, got %q", h)
	}
	if h := rec.Header().Get("X-RateLimit-Reset"); h != "1704067200" {
		t.Errorf("expected X-RateLimit-Reset 1704067200, got %q", h)
	}
	if h := rec.Header().Get("Retry-After"); h != "60" {
		t.Errorf("expected Retry-After 60, got %q", h)
	}

	// Verify Problem response body
	var problem models.Problem
	if err := json.NewDecoder(rec.Body).Decode(&problem); err != nil {
		t.Fatalf("failed to decode Problem response: %v", err)
	}
	if problem.Status != http.StatusTooManyRequests {
		t.Errorf("expected problem status 429, got %d", problem.Status)
	}
}

func TestTooManyRequests_WithoutRateLimitInfo(t *testing.T) {
	req, rec := requestWithContext(t, http.MethodGet, "/test")

	response.TooManyRequests(rec, req, "rate limit exceeded")

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("expected status 429, got %d", rec.Code)
	}

	// Should not have rate limit headers when info is nil
	if h := rec.Header().Get("X-RateLimit-Limit"); h != "" {
		t.Errorf("expected no X-RateLimit-Limit header, got %q", h)
	}
	if h := rec.Header().Get("Retry-After"); h != "" {
		t.Errorf("expected no Retry-After header, got %q", h)
	}
}

func TestBadRequest_IncludesTraceID(t *testing.T) {
	req, rec := requestWithContext(t, http.MethodPost, "/v1/test")

	fieldErrors := []models.FieldError{
		{Field: "name", Message: "is required"},
	}
	response.BadRequest(rec, req, "validation failed", fieldErrors)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}

	var problem models.Problem
	if err := json.NewDecoder(rec.Body).Decode(&problem); err != nil {
		t.Fatalf("failed to decode Problem response: %v", err)
	}

	if problem.TraceID == "" {
		t.Error("expected traceId to be set in Problem response")
	}
	if problem.Instance != "/v1/test" {
		t.Errorf("expected instance /v1/test, got %q", problem.Instance)
	}
}

func TestUnauthorized_ReturnsCorrectProblem(t *testing.T) {
	req, rec := requestWithContext(t, http.MethodGet, "/v1/me")

	response.Unauthorized(rec, req, "invalid token")

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rec.Code)
	}

	var problem models.Problem
	if err := json.NewDecoder(rec.Body).Decode(&problem); err != nil {
		t.Fatalf("failed to decode Problem response: %v", err)
	}

	if problem.Status != http.StatusUnauthorized {
		t.Errorf("expected problem status 401, got %d", problem.Status)
	}
	if problem.TraceID == "" {
		t.Error("expected traceId to be set")
	}
}

func TestNotFound_ReturnsCorrectProblem(t *testing.T) {
	req, rec := requestWithContext(t, http.MethodGet, "/v1/items/missing")

	response.NotFound(rec, req, "item not found")

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}

	var problem models.Problem
	if err := json.NewDecoder(rec.Body).Decode(&problem); err != nil {
		t.Fatalf("failed to decode Problem response: %v", err)
	}

	if problem.Status != http.StatusNotFound {
		t.Errorf("expected problem status 404, got %d", problem.Status)
	}
}

func TestConflict_ReturnsCorrectProblem(t *testing.T) {
	req, rec := requestWithContext(t, http.MethodPost, "/v1/items")

	response.Conflict(rec, req, "item already exists")

	if rec.Code != http.StatusConflict {
		t.Errorf("expected status 409, got %d", rec.Code)
	}

	var problem models.Problem
	if err := json.NewDecoder(rec.Body).Decode(&problem); err != nil {
		t.Fatalf("failed to decode Problem response: %v", err)
	}

	if problem.Status != http.StatusConflict {
		t.Errorf("expected problem status 409, got %d", problem.Status)
	}
}

func TestInternalError_ReturnsCorrectProblem(t *testing.T) {
	req, rec := requestWithContext(t, http.MethodGet, "/v1/test")

	response.InternalError(rec, req, "something went wrong")

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rec.Code)
	}

	var problem models.Problem
	if err := json.NewDecoder(rec.Body).Decode(&problem); err != nil {
		t.Fatalf("failed to decode Problem response: %v", err)
	}

	if problem.Status != http.StatusInternalServerError {
		t.Errorf("expected problem status 500, got %d", problem.Status)
	}
}

func TestServiceUnavailable_ReturnsCorrectProblem(t *testing.T) {
	req, rec := requestWithContext(t, http.MethodGet, "/v1/test")

	response.ServiceUnavailable(rec, req, "service temporarily unavailable")

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", rec.Code)
	}

	var problem models.Problem
	if err := json.NewDecoder(rec.Body).Decode(&problem); err != nil {
		t.Fatalf("failed to decode Problem response: %v", err)
	}

	if problem.Status != http.StatusServiceUnavailable {
		t.Errorf("expected problem status 503, got %d", problem.Status)
	}
}

func TestJSON_NilData(t *testing.T) {
	req, rec := requestWithContext(t, http.MethodGet, "/test")

	response.JSON(rec, req, http.StatusOK, nil)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	// Body should be empty when data is nil
	if rec.Body.Len() != 0 {
		t.Errorf("expected empty body for nil data, got %q", rec.Body.String())
	}
}

func TestRequestIDPropagation(t *testing.T) {
	// Test that incoming X-Request-Id header is preserved
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.Header.Set("X-Request-Id", "client-request-123")
	rec := httptest.NewRecorder()

	// Process through RequestID middleware
	var processedReq *http.Request
	handler := middleware.RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		processedReq = r
	}))
	handler.ServeHTTP(rec, req)

	// Verify the client's request ID was preserved in context
	requestID := middleware.GetRequestID(processedReq.Context())
	if requestID != "client-request-123" {
		t.Errorf("expected client request ID to be preserved, got %q", requestID)
	}

	// Now use the response functions with the processed request
	rec = httptest.NewRecorder()
	response.JSON(rec, processedReq, http.StatusOK, map[string]string{"status": "ok"})

	// Verify the response contains the client's request ID
	respRequestID := rec.Header().Get("X-Request-Id")
	if respRequestID != "client-request-123" {
		t.Errorf("expected response X-Request-Id to match client's, got %q", respRequestID)
	}
}

// Verify context.Background() returns empty request ID.
func TestGetRequestID_EmptyContext(t *testing.T) {
	requestID := middleware.GetRequestID(context.Background())
	if requestID != "" {
		t.Errorf("expected empty request ID for background context, got %q", requestID)
	}
}
