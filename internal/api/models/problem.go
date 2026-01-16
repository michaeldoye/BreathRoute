package models

import (
	"encoding/json"
	"net/http"
)

// Problem represents an RFC7807 error response.
// This is used for all API error responses with Content-Type: application/problem+json.
type Problem struct {
	// Type is a URI reference that identifies the problem type.
	Type string `json:"type"`

	// Title is a short, human-readable summary of the problem type.
	Title string `json:"title"`

	// Status is the HTTP status code for this occurrence of the problem.
	Status int `json:"status"`

	// Detail is a human-readable explanation specific to this occurrence.
	Detail string `json:"detail,omitempty"`

	// Instance is a URI reference that identifies the specific occurrence.
	Instance string `json:"instance,omitempty"`

	// TraceID is the request trace identifier for debugging.
	TraceID string `json:"traceId"`

	// Errors contains structured field validation errors.
	Errors []FieldError `json:"errors,omitempty"`
}

// FieldError represents a validation error on a specific field.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

// ProblemType constants for standard error types.
const (
	ProblemTypeValidation     = "https://api.breatheroute.nl/problems/validation-error"
	ProblemTypeUnauthorized   = "https://api.breatheroute.nl/problems/unauthorized"
	ProblemTypeNotFound       = "https://api.breatheroute.nl/problems/not-found"
	ProblemTypeConflict       = "https://api.breatheroute.nl/problems/conflict"
	ProblemTypeTooManyRequests = "https://api.breatheroute.nl/problems/too-many-requests"
	ProblemTypeInternal       = "https://api.breatheroute.nl/problems/internal-error"
	ProblemTypeUnavailable    = "https://api.breatheroute.nl/problems/service-unavailable"
)

// NewProblem creates a new Problem with the given parameters.
func NewProblem(problemType, title string, status int, traceID string) *Problem {
	return &Problem{
		Type:    problemType,
		Title:   title,
		Status:  status,
		TraceID: traceID,
	}
}

// WithDetail adds a detail message to the Problem.
func (p *Problem) WithDetail(detail string) *Problem {
	p.Detail = detail
	return p
}

// WithInstance adds the request instance URI to the Problem.
func (p *Problem) WithInstance(instance string) *Problem {
	p.Instance = instance
	return p
}

// WithErrors adds field errors to the Problem.
func (p *Problem) WithErrors(errors []FieldError) *Problem {
	p.Errors = errors
	return p
}

// Write writes the Problem as JSON to the ResponseWriter.
func (p *Problem) Write(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.Header().Set("X-Request-Id", p.TraceID)
	w.WriteHeader(p.Status)
	_ = json.NewEncoder(w).Encode(p)
}

// NewBadRequest creates a 400 Bad Request problem.
func NewBadRequest(traceID, detail string, errors []FieldError) *Problem {
	p := NewProblem(ProblemTypeValidation, "Validation error", http.StatusBadRequest, traceID)
	p.Detail = detail
	p.Errors = errors
	return p
}

// NewUnauthorized creates a 401 Unauthorized problem.
func NewUnauthorized(traceID, detail string) *Problem {
	p := NewProblem(ProblemTypeUnauthorized, "Unauthorized", http.StatusUnauthorized, traceID)
	p.Detail = detail
	return p
}

// NewNotFound creates a 404 Not Found problem.
func NewNotFound(traceID, detail string) *Problem {
	p := NewProblem(ProblemTypeNotFound, "Not found", http.StatusNotFound, traceID)
	p.Detail = detail
	return p
}

// NewConflict creates a 409 Conflict problem.
func NewConflict(traceID, detail string) *Problem {
	p := NewProblem(ProblemTypeConflict, "Conflict", http.StatusConflict, traceID)
	p.Detail = detail
	return p
}

// NewTooManyRequests creates a 429 Too Many Requests problem.
func NewTooManyRequests(traceID, detail string) *Problem {
	p := NewProblem(ProblemTypeTooManyRequests, "Too many requests", http.StatusTooManyRequests, traceID)
	p.Detail = detail
	return p
}

// NewInternalError creates a 500 Internal Server Error problem.
func NewInternalError(traceID, detail string) *Problem {
	p := NewProblem(ProblemTypeInternal, "Internal server error", http.StatusInternalServerError, traceID)
	p.Detail = detail
	return p
}

// NewServiceUnavailable creates a 503 Service Unavailable problem.
func NewServiceUnavailable(traceID, detail string) *Problem {
	p := NewProblem(ProblemTypeUnavailable, "Service unavailable", http.StatusServiceUnavailable, traceID)
	p.Detail = detail
	return p
}
