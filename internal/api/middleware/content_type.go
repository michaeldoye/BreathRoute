package middleware

import (
	"net/http"
	"strings"
)

// ContentTypeJSON sets the Content-Type header to application/json.
func ContentTypeJSON(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only set if not already set (allows handlers to override)
		if w.Header().Get("Content-Type") == "" {
			w.Header().Set("Content-Type", "application/json")
		}
		next.ServeHTTP(w, r)
	})
}

// RequireJSON checks that the request Content-Type is application/json for
// POST, PUT, and PATCH requests.
func RequireJSON(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only check for methods that typically have a body
		if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
			contentType := r.Header.Get("Content-Type")
			if contentType != "" && !strings.HasPrefix(contentType, "application/json") {
				http.Error(w, `{"error":"Content-Type must be application/json"}`, http.StatusUnsupportedMediaType)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
