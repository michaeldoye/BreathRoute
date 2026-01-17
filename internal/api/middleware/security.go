package middleware

import (
	"net/http"
	"os"

	"github.com/breatheroute/breatheroute/internal/api/models"
)

// SecurityHeaders adds standard security headers to all HTTP responses.
// Headers set:
//   - X-Content-Type-Options: nosniff
//   - X-Frame-Options: DENY
//   - Strict-Transport-Security: max-age=31536000; includeSubDomains
//   - Content-Security-Policy: default-src 'none'; frame-ancestors 'none'
//   - Referrer-Policy: strict-origin-when-cross-origin
//   - Permissions-Policy: geolocation=(), camera=(), microphone=()
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prevent MIME type sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking
		w.Header().Set("X-Frame-Options", "DENY")

		// Enforce HTTPS for 1 year with subdomains
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		// Restrictive Content Security Policy for API
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")

		// Control referrer information
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Disable browser features
		w.Header().Set("Permissions-Policy", "geolocation=(), camera=(), microphone=()")

		next.ServeHTTP(w, r)
	})
}

// RequireTLS middleware enforces HTTPS connections.
// It checks the X-Forwarded-Proto header (set by Cloud Run/load balancers).
// Enable with REQUIRE_TLS=true environment variable.
func RequireTLS(next http.Handler) http.Handler {
	requireTLS := os.Getenv("REQUIRE_TLS") == "true"

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requireTLS {
			// Check X-Forwarded-Proto header (set by load balancer/Cloud Run)
			proto := r.Header.Get("X-Forwarded-Proto")
			if proto != "" && proto != "https" {
				traceID := GetRequestID(r.Context())
				problem := models.NewProblem(
					"https://api.breatheroute.nl/problems/tls-required",
					"TLS required",
					http.StatusForbidden,
					traceID,
				)
				problem.Detail = "This endpoint requires HTTPS"
				problem.Instance = r.URL.Path
				problem.Write(w)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}
