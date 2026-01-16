package models_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/breatheroute/breatheroute/internal/api/models"
)

func TestProblem_NewProblem(t *testing.T) {
	p := models.NewProblem(
		models.ProblemTypeValidation,
		"Validation error",
		http.StatusBadRequest,
		"req_test123",
	)

	assert.Equal(t, models.ProblemTypeValidation, p.Type)
	assert.Equal(t, "Validation error", p.Title)
	assert.Equal(t, http.StatusBadRequest, p.Status)
	assert.Equal(t, "req_test123", p.TraceID)
	assert.Empty(t, p.Detail)
	assert.Empty(t, p.Instance)
	assert.Nil(t, p.Errors)
}

func TestProblem_WithDetail(t *testing.T) {
	p := models.NewProblem(
		models.ProblemTypeValidation,
		"Validation error",
		http.StatusBadRequest,
		"req_test123",
	).WithDetail("origin.lat must be between -90 and 90")

	assert.Equal(t, "origin.lat must be between -90 and 90", p.Detail)
}

func TestProblem_WithInstance(t *testing.T) {
	p := models.NewProblem(
		models.ProblemTypeValidation,
		"Validation error",
		http.StatusBadRequest,
		"req_test123",
	).WithInstance("/v1/routes:compute")

	assert.Equal(t, "/v1/routes:compute", p.Instance)
}

func TestProblem_WithErrors(t *testing.T) {
	fieldErrors := []models.FieldError{
		{Field: "origin.lat", Message: "must be between -90 and 90", Code: "OUT_OF_RANGE"},
		{Field: "origin.lon", Message: "required", Code: "REQUIRED"},
	}

	p := models.NewProblem(
		models.ProblemTypeValidation,
		"Validation error",
		http.StatusBadRequest,
		"req_test123",
	).WithErrors(fieldErrors)

	require.Len(t, p.Errors, 2)
	assert.Equal(t, "origin.lat", p.Errors[0].Field)
	assert.Equal(t, "must be between -90 and 90", p.Errors[0].Message)
	assert.Equal(t, "OUT_OF_RANGE", p.Errors[0].Code)
}

func TestProblem_Write(t *testing.T) {
	p := models.NewBadRequest("req_test123", "invalid input", []models.FieldError{
		{Field: "email", Message: "invalid format"},
	})
	p.Instance = "/v1/me/profile"

	w := httptest.NewRecorder()
	p.Write(w)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "application/problem+json", w.Header().Get("Content-Type"))
	assert.Equal(t, "req_test123", w.Header().Get("X-Request-Id"))

	var result models.Problem
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, models.ProblemTypeValidation, result.Type)
	assert.Equal(t, "Validation error", result.Title)
	assert.Equal(t, http.StatusBadRequest, result.Status)
	assert.Equal(t, "invalid input", result.Detail)
	assert.Equal(t, "/v1/me/profile", result.Instance)
	assert.Equal(t, "req_test123", result.TraceID)
	require.Len(t, result.Errors, 1)
	assert.Equal(t, "email", result.Errors[0].Field)
}

func TestNewBadRequest(t *testing.T) {
	p := models.NewBadRequest("req_123", "invalid data", nil)

	assert.Equal(t, models.ProblemTypeValidation, p.Type)
	assert.Equal(t, "Validation error", p.Title)
	assert.Equal(t, http.StatusBadRequest, p.Status)
	assert.Equal(t, "invalid data", p.Detail)
	assert.Equal(t, "req_123", p.TraceID)
}

func TestNewUnauthorized(t *testing.T) {
	p := models.NewUnauthorized("req_123", "token expired")

	assert.Equal(t, models.ProblemTypeUnauthorized, p.Type)
	assert.Equal(t, "Unauthorized", p.Title)
	assert.Equal(t, http.StatusUnauthorized, p.Status)
	assert.Equal(t, "token expired", p.Detail)
}

func TestNewNotFound(t *testing.T) {
	p := models.NewNotFound("req_123", "user not found")

	assert.Equal(t, models.ProblemTypeNotFound, p.Type)
	assert.Equal(t, "Not found", p.Title)
	assert.Equal(t, http.StatusNotFound, p.Status)
	assert.Equal(t, "user not found", p.Detail)
}

func TestNewConflict(t *testing.T) {
	p := models.NewConflict("req_123", "duplicate entry")

	assert.Equal(t, models.ProblemTypeConflict, p.Type)
	assert.Equal(t, "Conflict", p.Title)
	assert.Equal(t, http.StatusConflict, p.Status)
	assert.Equal(t, "duplicate entry", p.Detail)
}

func TestNewTooManyRequests(t *testing.T) {
	p := models.NewTooManyRequests("req_123", "rate limit exceeded")

	assert.Equal(t, models.ProblemTypeTooManyRequests, p.Type)
	assert.Equal(t, "Too many requests", p.Title)
	assert.Equal(t, http.StatusTooManyRequests, p.Status)
	assert.Equal(t, "rate limit exceeded", p.Detail)
}

func TestNewInternalError(t *testing.T) {
	p := models.NewInternalError("req_123", "database error")

	assert.Equal(t, models.ProblemTypeInternal, p.Type)
	assert.Equal(t, "Internal server error", p.Title)
	assert.Equal(t, http.StatusInternalServerError, p.Status)
	assert.Equal(t, "database error", p.Detail)
}

func TestNewServiceUnavailable(t *testing.T) {
	p := models.NewServiceUnavailable("req_123", "upstream unavailable")

	assert.Equal(t, models.ProblemTypeUnavailable, p.Type)
	assert.Equal(t, "Service unavailable", p.Title)
	assert.Equal(t, http.StatusServiceUnavailable, p.Status)
	assert.Equal(t, "upstream unavailable", p.Detail)
}
