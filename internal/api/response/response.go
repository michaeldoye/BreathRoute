// Package response provides utilities for HTTP response handling.
package response

import (
	"encoding/json"
	"net/http"

	"github.com/breatheroute/breatheroute/internal/api/middleware"
	"github.com/breatheroute/breatheroute/internal/api/models"
)

// JSON writes a JSON response with the given status code.
func JSON(w http.ResponseWriter, status int, data interface{}) {
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

// TooManyRequests writes a 429 Too Many Requests error response.
func TooManyRequests(w http.ResponseWriter, r *http.Request, detail string) {
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
func Created(w http.ResponseWriter, location string, data interface{}) {
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
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// Accepted writes a 202 Accepted response with Location header.
func Accepted(w http.ResponseWriter, location string, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if location != "" {
		w.Header().Set("Location", location)
	}
	w.WriteHeader(http.StatusAccepted)
	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}
