package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/breatheroute/breatheroute/internal/api/models"
	"github.com/breatheroute/breatheroute/internal/auth"
)

// userIDKey is the context key for the authenticated user ID.
type userIDKey struct{}

// Auth creates authentication middleware that validates JWT bearer tokens.
func Auth(authService *auth.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract bearer token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeUnauthorized(w, r, "missing authorization header")
				return
			}

			// Check for Bearer prefix (case-insensitive)
			const bearerPrefix = "Bearer "
			if len(authHeader) < len(bearerPrefix) ||
				!strings.EqualFold(authHeader[:len(bearerPrefix)], bearerPrefix) {
				writeUnauthorized(w, r, "invalid authorization header format")
				return
			}

			tokenString := authHeader[len(bearerPrefix):]
			if tokenString == "" {
				writeUnauthorized(w, r, "missing bearer token")
				return
			}

			// Validate the token
			userID, err := authService.ValidateAccessToken(tokenString)
			if err != nil {
				switch {
				case errors.Is(err, auth.ErrAccessTokenExpired):
					writeUnauthorized(w, r, "access token has expired")
				case errors.Is(err, auth.ErrInvalidAccessToken):
					writeUnauthorized(w, r, "invalid access token")
				default:
					writeUnauthorized(w, r, "authentication failed")
				}
				return
			}

			// Add user ID to context
			ctx := context.WithValue(r.Context(), userIDKey{}, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// writeUnauthorized writes a 401 Unauthorized response.
// This is implemented directly here to avoid import cycle with response package.
func writeUnauthorized(w http.ResponseWriter, r *http.Request, detail string) {
	traceID := GetRequestID(r.Context())
	problem := models.NewUnauthorized(traceID, detail)
	problem.Instance = r.URL.Path
	problem.Write(w)
}

// GetUserID retrieves the authenticated user ID from the context.
// Returns an empty string if not authenticated.
func GetUserID(ctx context.Context) string {
	if id, ok := ctx.Value(userIDKey{}).(string); ok {
		return id
	}
	return ""
}
