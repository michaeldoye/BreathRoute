// Package middleware provides HTTP middleware for the BreatheRoute API.
package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

// requestIDKey is the context key for the request ID.
type requestIDKey struct{}

// RequestID generates a unique request ID and adds it to the request context.
// The ID is also set in the X-Request-Id response header.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if request already has an ID
		requestID := r.Header.Get("X-Request-Id")
		if requestID == "" {
			// Generate new request ID with prefix
			requestID = "req_" + uuid.New().String()[:22]
		}

		// Set in response header
		w.Header().Set("X-Request-Id", requestID)

		// Add to context
		ctx := context.WithValue(r.Context(), requestIDKey{}, requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetRequestID retrieves the request ID from the context.
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey{}).(string); ok {
		return id
	}
	return ""
}
