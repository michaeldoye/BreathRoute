// Package response provides utilities for HTTP response handling.
package response

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/breatheroute/breatheroute/internal/api/middleware"
	"github.com/breatheroute/breatheroute/internal/api/models"
)

// JSON writes a JSON response with the given status code.
// Includes X-Request-Id header for correlation.
func JSON(w http.ResponseWriter, r *http.Request, status int, data interface{}) {
	requestID := middleware.GetRequestID(r.Context())
	if requestID != "" {
		w.Header().Set("X-Request-Id", requestID)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}

// Error writes a Problem+JSON error response.
func Error(w http.ResponseWriter, r *http.Request, problem *models.Problem) {
	problem.Instance = r.URL.Path
	problem.Write(w)
}

// BadRequest writes a 400 Bad Request error response.
func BadRequest(w http.ResponseWriter, r *http.Request, detail string, errors []models.FieldError) {
	traceID := middleware.GetRequestID(r.Context())
	problem := models.NewBadRequest(traceID, detail, errors)
	Error(w, r, problem)
}

// Unauthorized writes a 401 Unauthorized error response.
func Unauthorized(w http.ResponseWriter, r *http.Request, detail string) {
	traceID := middleware.GetRequestID(r.Context())
	problem := models.NewUnauthorized(traceID, detail)
	Error(w, r, problem)
}

// NotFound writes a 404 Not Found error response.
func NotFound(w http.ResponseWriter, r *http.Request, detail string) {
	traceID := middleware.GetRequestID(r.Context())
	problem := models.NewNotFound(traceID, detail)
	Error(w, r, problem)
}

// Conflict writes a 409 Conflict error response.
func Conflict(w http.ResponseWriter, r *http.Request, detail string) {
	traceID := middleware.GetRequestID(r.Context())
	problem := models.NewConflict(traceID, detail)
	Error(w, r, problem)
}

// RateLimitInfo contains rate limit information for 429 responses.
type RateLimitInfo struct {
	// Limit is the maximum number of requests allowed in the window.
	Limit int
	// Remaining is the number of requests remaining in the current window.
	Remaining int
	// ResetAt is the Unix timestamp when the rate limit window resets.
	ResetAt int64
	// RetryAfter is the number of seconds until the client should retry.
	RetryAfter int
}

// TooManyRequests writes a 429 Too Many Requests error response.
func TooManyRequests(w http.ResponseWriter, r *http.Request, detail string) {
	TooManyRequestsWithInfo(w, r, detail, nil)
}

// TooManyRequestsWithInfo writes a 429 Too Many Requests error response with rate limit headers.
func TooManyRequestsWithInfo(w http.ResponseWriter, r *http.Request, detail string, info *RateLimitInfo) {
	if info != nil {
		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(info.Limit))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(info.Remaining))
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(info.ResetAt, 10))
		if info.RetryAfter > 0 {
			w.Header().Set("Retry-After", strconv.Itoa(info.RetryAfter))
		}
	}
	traceID := middleware.GetRequestID(r.Context())
	problem := models.NewTooManyRequests(traceID, detail)
	Error(w, r, problem)
}

// InternalError writes a 500 Internal Server Error response.
func InternalError(w http.ResponseWriter, r *http.Request, detail string) {
	traceID := middleware.GetRequestID(r.Context())
	problem := models.NewInternalError(traceID, detail)
	Error(w, r, problem)
}

// ServiceUnavailable writes a 503 Service Unavailable error response.
func ServiceUnavailable(w http.ResponseWriter, r *http.Request, detail string) {
	traceID := middleware.GetRequestID(r.Context())
	problem := models.NewServiceUnavailable(traceID, detail)
	Error(w, r, problem)
}

// Created writes a 201 Created response with Location header.
// Includes X-Request-Id header for correlation.
func Created(w http.ResponseWriter, r *http.Request, location string, data interface{}) {
	requestID := middleware.GetRequestID(r.Context())
	if requestID != "" {
		w.Header().Set("X-Request-Id", requestID)
	}
	w.Header().Set("Content-Type", "application/json")
	if location != "" {
		w.Header().Set("Location", location)
	}
	w.WriteHeader(http.StatusCreated)
	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}

// NoContent writes a 204 No Content response.
// Includes X-Request-Id header for correlation.
func NoContent(w http.ResponseWriter, r *http.Request) {
	requestID := middleware.GetRequestID(r.Context())
	if requestID != "" {
		w.Header().Set("X-Request-Id", requestID)
	}
	w.WriteHeader(http.StatusNoContent)
}

// Accepted writes a 202 Accepted response with Location header.
// Includes X-Request-Id header for correlation.
func Accepted(w http.ResponseWriter, r *http.Request, location string, data interface{}) {
	requestID := middleware.GetRequestID(r.Context())
	if requestID != "" {
		w.Header().Set("X-Request-Id", requestID)
	}
	w.Header().Set("Content-Type", "application/json")
	if location != "" {
		w.Header().Set("Location", location)
	}
	w.WriteHeader(http.StatusAccepted)
	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}
